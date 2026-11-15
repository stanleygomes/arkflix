package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// httpErr é um erro com o status que vai pro cliente.
type httpErr struct {
	code int
	msg  string
}

func (e *httpErr) Error() string { return e.msg }

func bad(format string, args ...any) error {
	return &httpErr{http.StatusBadRequest, fmt.Sprintf(format, args...)}
}

var (
	errReadOnly = &httpErr{http.StatusForbidden, "servidor em modo só leitura"}
	errExists   = &httpErr{http.StatusConflict, "já existe um arquivo com esse nome"}
)

func failErr(w http.ResponseWriter, err error) {
	var he *httpErr
	if errors.As(err, &he) {
		fail(w, he.code, he.msg)
		return
	}
	internal(w, err)
}

var (
	videoExts = map[string]bool{"mkv": true, "mp4": true, "m4v": true, "avi": true, "webm": true, "mov": true, "wmv": true,
		"ts": true, "m2ts": true, "mts": true, "mpg": true, "mpeg": true, "flv": true, "ogv": true, "3gp": true,
		"divx": true, "vob": true, "iso": true}
	subExts = map[string]bool{"srt": true, "vtt": true, "ass": true, "ssa": true, "sub": true}
	slugRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	langRe  = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})?$`)
)

// clean deixa um nome seguro pra virar um pedaço de caminho: sem
// controle, sem separador nem caracteres que o Windows/SMB recusa, sem
// espaço/ponto nas pontas ("..", ".oculto" não passam). "" = não sobrou nada.
func clean(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20 || r == 0x7f || r == utf8.RuneError:
			return -1
		case strings.ContainsRune(`/\:*?"<>|`, r):
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, " .")
	for len(s) > 180 {
		_, n := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-n]
	}
	return strings.TrimRight(s, " .")
}

// cleanFile limpa o nome de arquivo mantendo a extensão (minúscula).
func cleanFile(name string) (stem, ext string) {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	stem = clean(strings.TrimSuffix(name, filepath.Ext(name)))
	if !slugRe.MatchString(ext) {
		ext = ""
	}
	return stem, ext
}

func join(stem, ext string) string {
	if ext == "" {
		return stem
	}
	return stem + "." + ext
}

// dest é pra onde vai um arquivo enviado: caminho absoluto e o que a
// resposta mostra ("Movies/Nome (2020)/Nome (2020).mkv").
type dest struct {
	abs, root string
}

func (d dest) shown() string {
	rel, _ := filepath.Rel(d.root, d.abs)
	return filepath.ToSlash(filepath.Join(filepath.Base(d.root), rel))
}

func (a *api) libRoot(lib string) (string, error) {
	cfg := a.d.Config
	root := map[string]string{"movies": cfg.Movies, "shows": cfg.Shows, "music": cfg.Music, "games": cfg.Games}[lib]
	if root == "" {
		return "", bad("biblioteca %s não configurada", lib)
	}
	return root, nil
}

// destFor monta o destino pelo kind e os campos do formulário.
func (a *api) destFor(ctx context.Context, f map[string]string, filename string) (dest, error) {
	stem, ext := cleanFile(filename)
	if stem == "" {
		return dest{}, bad("nome de arquivo inválido")
	}
	kind := f["kind"]
	lib := map[string]string{"movie": "movies", "episode": "shows", "music": "music", "rom": "games", "bios": "games", "subtitle": ""}
	l, ok := lib[kind]
	if !ok {
		return dest{}, bad("kind inválido: use movie, episode, music, rom, bios ou subtitle")
	}
	var root string
	if l != "" {
		var err error
		if root, err = a.libRoot(l); err != nil {
			return dest{}, err
		}
	}
	var parts []string
	switch kind {
	case "movie":
		if !videoExts[ext] {
			return dest{}, bad("filme precisa ser vídeo")
		}
		title := clean(f["title"])
		if title == "" {
			title = stem
		}
		if y := strings.TrimSpace(f["year"]); y != "" {
			n, err := strconv.Atoi(y)
			if err != nil || n < 1800 || n > 2999 {
				return dest{}, bad("ano inválido")
			}
			title += " (" + y + ")"
		}
		parts = []string{title, join(title, ext)}
	case "episode":
		if !videoExts[ext] {
			return dest{}, bad("episódio precisa ser vídeo")
		}
		series := clean(f["series"])
		season, err := strconv.Atoi(strings.TrimSpace(f["season"]))
		if series == "" || err != nil || season < 0 || season > 999 {
			return dest{}, bad("episódio precisa de series e season")
		}
		name := join(stem, ext)
		if e := strings.TrimSpace(f["episode"]); e != "" {
			ep, err := strconv.Atoi(e)
			if err != nil || ep < 0 || ep > 9999 {
				return dest{}, bad("episode inválido")
			}
			name = fmt.Sprintf("%s - S%02dE%02d", series, season, ep)
			if t := clean(f["title"]); t != "" {
				name += " - " + t
			}
			name = join(name, ext)
		}
		parts = []string{series, fmt.Sprintf("Season %02d", season), name}
	case "music":
		for seg := range strings.SplitSeq(strings.ReplaceAll(f["folder"], `\`, "/"), "/") {
			if seg = clean(seg); seg != "" {
				parts = append(parts, seg)
			}
		}
		if len(parts) == 0 {
			parts = []string{"Avulsas"}
		}
		parts = append(parts, join(stem, ext))
	case "rom", "bios":
		slug := strings.ToLower(strings.TrimSpace(f["platform"]))
		if !slugRe.MatchString(slug) {
			return dest{}, bad("platform inválido (use o slug: snes, psx...)")
		}
		dir := map[string]string{"rom": "roms", "bios": "bios"}[kind]
		parts = []string{dir, slug, join(stem, ext)}
	case "subtitle":
		if !subExts[ext] {
			return dest{}, bad("legenda precisa ser srt, vtt, ass, ssa ou sub")
		}
		it, err := a.item(ctx, strings.TrimSpace(f["item"]))
		if err != nil || (it.Kind != store.KindMovie && it.Kind != store.KindEpisode) {
			return dest{}, bad("item precisa ser um filme ou episódio")
		}
		if root, err = a.libRoot(it.Library); err != nil {
			return dest{}, err
		}
		base := strings.TrimSuffix(filepath.Base(it.Path), filepath.Ext(it.Path))
		if lang := strings.TrimSpace(f["lang"]); lang != "" {
			if !langRe.MatchString(lang) {
				return dest{}, bad("lang inválido")
			}
			base += "." + lang
		}
		rel, err := filepath.Rel(root, filepath.Join(filepath.Dir(it.Path), join(base, ext)))
		if err != nil {
			return dest{}, bad("item fora da biblioteca")
		}
		parts = []string{rel}
	}
	abs := filepath.Join(append([]string{root}, parts...)...)
	if !inside(root, abs) {
		return dest{}, bad("caminho inválido")
	}
	return dest{abs: abs, root: root}, nil
}

// inside: p está dentro de root (e não é root).
func inside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// partName é o temporário ao lado do destino (oculto: o scan pula).
func partName(abs string) string {
	return filepath.Join(filepath.Dir(abs), "."+filepath.Base(abs)+".part")
}

// commit põe tmp no lugar de abs sem sobrescrever.
func commit(tmp, abs string) error {
	if err := os.Link(tmp, abs); err == nil {
		return os.Remove(tmp)
	} else if errors.Is(err, fs.ErrExist) {
		os.Remove(tmp)
		return errExists
	}
	// sistema de arquivos sem hard link: confere e renomeia
	if _, err := os.Lstat(abs); err == nil {
		os.Remove(tmp)
		return errExists
	}
	return os.Rename(tmp, abs)
}

// save grava o arquivo direto no destino (via .part).
func save(src io.Reader, abs string) error {
	if _, err := os.Lstat(abs); err == nil {
		return errExists
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp := partName(abs)
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return &httpErr{http.StatusConflict, "esse arquivo já está sendo enviado"}
	}
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, src); err != nil {
		f.Close()
		os.Remove(tmp)
		return &httpErr{http.StatusBadRequest, "envio interrompido"}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return commit(tmp, abs)
}

// place leva um arquivo da área de espera pro destino (rename, ou cópia
// se for outro disco).
func place(staged, abs string) error {
	if _, err := os.Lstat(abs); err == nil {
		return errExists
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp := partName(abs)
	if err := os.Rename(staged, tmp); err != nil {
		in, err := os.Open(staged)
		if err != nil {
			return err
		}
		defer in.Close()
		f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, in); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		os.Remove(staged)
	}
	return commit(tmp, abs)
}

type staged struct{ tmp, name string }

// POST /upload: multipart com os campos de destino e um ou mais "file".
// Arquivo que chega depois de "kind" vai direto pro destino; antes, espera
// em <data>/uploads até os campos chegarem.
func (a *api) upload(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	if !a.d.Config.Writable {
		failErr(w, errReadOnly)
		return
	}
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "multipart/form-data" {
		fail(w, http.StatusBadRequest, "esperava multipart/form-data")
		return
	}
	saved := []string{}
	var wait []staged
	defer func() {
		for _, s := range wait {
			os.Remove(s.tmp)
		}
		if len(saved) > 0 {
			a.d.Library.Trigger()
		}
	}()
	fields := map[string]string{}
	err := a.readUpload(r, fields, &saved, &wait)
	if err == nil {
		for len(wait) > 0 && err == nil {
			s := wait[0]
			var d dest
			if d, err = a.destFor(r.Context(), fields, s.name); err == nil {
				if err = place(s.tmp, d.abs); err == nil {
					saved = append(saved, d.shown())
				}
			}
			if err == nil {
				wait = wait[1:]
			}
		}
	}
	if err == nil && len(saved) == 0 {
		err = bad("nenhum arquivo no campo file")
	}
	if err != nil {
		failErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"saved": saved})
}

// readUpload lê o multipart parte por parte (sem guardar arquivo na memória).
func (a *api) readUpload(r *http.Request, fields map[string]string, saved *[]string, wait *[]staged) error {
	mr, err := r.MultipartReader()
	if err != nil {
		return bad("esperava multipart/form-data")
	}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return bad("multipart inválido")
		}
		name := p.FormName()
		switch {
		case name == "file" && p.FileName() != "":
			err = a.takeFile(r.Context(), p, fields, saved, wait)
		case name != "":
			v, rerr := io.ReadAll(io.LimitReader(p, 4097))
			if rerr != nil || len(v) > 4096 {
				err = bad("campo %s grande demais", name)
			} else {
				fields[name] = string(v)
			}
		}
		p.Close()
		if err != nil {
			return err
		}
	}
}

// takeFile grava um arquivo: direto no destino se já tem kind, senão na
// área de espera.
func (a *api) takeFile(ctx context.Context, p *multipart.Part, fields map[string]string, saved *[]string, wait *[]staged) error {
	if fields["kind"] != "" {
		d, err := a.destFor(ctx, fields, p.FileName())
		if err != nil {
			return err
		}
		if err := save(p, d.abs); err != nil {
			return err
		}
		*saved = append(*saved, d.shown())
		return nil
	}
	dir := filepath.Join(a.d.Config.Data, "uploads")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "upload-*.part")
	if err != nil {
		return err
	}
	*wait = append(*wait, staged{tmp: f.Name(), name: p.FileName()})
	if _, err := io.Copy(f, p); err != nil {
		f.Close()
		return bad("envio interrompido")
	}
	return f.Close()
}
