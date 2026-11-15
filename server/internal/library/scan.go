package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// scanner é o estado de um scan de uma biblioteca.
type scanner struct {
	ctx  context.Context
	cfg  *config.Config
	st   *store.Store
	lib  string
	root string
	old  map[string]*store.Item // por caminho, do scan anterior
	seen map[string]bool
	dirs map[string]map[string]string // pasta → nome minúsculo → nome real
}

// scanAll escaneia cada biblioteca configurada. Pasta que não existe (disco
// desmontado) é pulada sem apagar nada do banco.
func scanAll(ctx context.Context, cfg *config.Config, st *store.Store) (int, error) {
	libs := []struct {
		name, root string
		walk       func(*scanner) error
	}{
		{"movies", cfg.Movies, (*scanner).movies},
		{"shows", cfg.Shows, (*scanner).shows},
		{"music", cfg.Music, (*scanner).music},
		{"games", cfg.Games, (*scanner).games},
	}
	total := 0
	for _, l := range libs {
		if l.root == "" {
			continue
		}
		if fi, err := os.Stat(l.root); err != nil || !fi.IsDir() {
			slog.Warn("scan: pasta da biblioteca indisponível", "lib", l.name, "path", l.root, "err", err)
			continue
		}
		n, err := scanLib(ctx, cfg, st, l.name, l.root, l.walk)
		total += n
		if err != nil {
			return total, fmt.Errorf("scan %s: %w", l.name, err)
		}
	}
	return total, nil
}

func scanLib(ctx context.Context, cfg *config.Config, st *store.Store, lib, root string, walk func(*scanner) error) (int, error) {
	olds, err := st.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE i.library = ?`, lib)
	if err != nil {
		return 0, err
	}
	s := &scanner{ctx: ctx, cfg: cfg, st: st, lib: lib, root: root,
		old: map[string]*store.Item{}, seen: map[string]bool{}, dirs: map[string]map[string]string{}}
	for _, it := range olds {
		s.old[it.Path] = it
	}
	if err := walk(s); err != nil {
		return len(s.seen), err
	}
	if err := ctx.Err(); err != nil {
		return len(s.seen), err
	}
	gone := s.gone()
	slog.Info("scan: biblioteca", "lib", lib, "itens", len(s.seen), "removidos", len(gone))
	return len(s.seen), st.DeleteItems(ctx, gone)
}

// gone são os itens do banco que não apareceram neste scan.
func (s *scanner) gone() []string {
	var ids []string
	for p, it := range s.old {
		if !s.seen[p] {
			ids = append(ids, it.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

// put grava o item se mudou desde o último scan.
func (s *scanner) put(it *store.Item) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	it.Library = s.lib
	if it.ID == "" {
		it.ID = store.ID(it.Kind, it.Path)
	}
	if it.SortName == "" {
		it.SortName = store.SortName(it.Name)
	}
	s.seen[it.Path] = true
	if old := s.old[it.Path]; old != nil && same(old, it) {
		it.RID = old.RID
		return nil
	}
	return s.st.UpsertItem(s.ctx, it)
}

// same compara ignorando rid e datas.
func same(a, b *store.Item) bool {
	return string(canon(a)) == string(canon(b))
}

func canon(it *store.Item) []byte {
	c := *it
	c.RID, c.AddedAt, c.UpdatedAt = 0, time.Time{}, time.Time{}
	if c.Genres == nil {
		c.Genres = []string{}
	}
	if c.Streams == nil {
		c.Streams = []store.Stream{}
	}
	raw, _ := json.Marshal(c.Extra)
	var extra map[string]any
	json.Unmarshal(raw, &extra)
	if extra == nil {
		extra = map[string]any{}
	}
	c.Extra = extra
	out, _ := json.Marshal(c)
	return out
}

// unchanged devolve o item antigo se o arquivo tem o mesmo tamanho e mtime.
func (s *scanner) unchanged(path string, fi fs.FileInfo) *store.Item {
	old := s.old[path]
	if old == nil || old.Size != fi.Size() || old.MTime != fi.ModTime().Unix() {
		return nil
	}
	return old
}

func (s *scanner) rel(path string) string {
	r, err := filepath.Rel(s.root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

func (s *scanner) warn(path string, err error) {
	slog.Warn("scan: arquivo", "path", path, "err", err)
}

// files lista a pasta (cache por scan): nome minúsculo → nome real.
func (s *scanner) files(dir string) map[string]string {
	if m, ok := s.dirs[dir]; ok {
		return m
	}
	m := map[string]string{}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if !e.IsDir() {
			m[strings.ToLower(e.Name())] = e.Name()
		}
	}
	s.dirs[dir] = m
	return m
}

// find acha em dir o primeiro arquivo base.ext (sem diferenciar
// maiúsculas) pra cada base em ordem.
func (s *scanner) find(dir string, bases []string, exts []string) string {
	m := s.files(dir)
	for _, b := range bases {
		for _, e := range exts {
			if real, ok := m[strings.ToLower(b+"."+e)]; ok {
				return filepath.Join(dir, real)
			}
		}
	}
	return ""
}

var imageExts = []string{"jpg", "jpeg", "png", "webp"}

// art são as imagens de um item: pôster, fundo e logo.
type art struct{ image, backdrop, logo string }

// findArt procura as imagens de base (nome do arquivo sem extensão) em dir;
// generic = também poster.jpg, fanart.jpg... (pasta própria do item).
func (s *scanner) findArt(dir, base string, generic bool) art {
	poster := []string{base, base + "-poster", base + "-cover"}
	back := []string{base + "-fanart", base + "-backdrop"}
	logo := []string{base + "-logo", base + "-clearlogo"}
	if base == "" {
		poster, back, logo = nil, nil, nil
	}
	if generic {
		poster = append(poster, "poster", "folder", "cover")
		back = append(back, "fanart", "backdrop", "background")
		logo = append(logo, "logo", "clearlogo")
	}
	return art{s.find(dir, poster, imageExts), s.find(dir, back, imageExts), s.find(dir, logo, imageExts)}
}

// skip diz se o nome é lixo de sistema/oculto.
func skip(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "@") || strings.HasPrefix(name, "#") {
		return true
	}
	switch strings.ToLower(name) {
	case "thumbs.db", "desktop.ini", "lost+found", "$recycle.bin":
		return true
	}
	return false
}

func ext(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

func stem(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// mediaFile é um arquivo achado na varredura.
type mediaFile struct {
	path string
	info fs.FileInfo
}

// walkFiles lista (ordenado) os arquivos com extensão em exts debaixo de
// dir, pulando ocultos e pastas em skipDir.
func (s *scanner) walkFiles(dir string, exts map[string]bool, skipDir func(string) bool) ([]mediaFile, error) {
	var out []mediaFile
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir {
				return err
			}
			s.warn(p, err)
			return nil
		}
		if cerr := s.ctx.Err(); cerr != nil {
			return cerr
		}
		if p != dir && skip(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != dir && skipDir != nil && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !exts[ext(d.Name())] {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			s.warn(p, err)
			return nil
		}
		out = append(out, mediaFile{p, fi})
		return nil
	})
	return out, err
}
