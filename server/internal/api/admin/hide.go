package admin

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// withSidecars é o arquivo e o que mora ao lado com o mesmo nome
// (legenda, .nfo, .lrc, capa "Nome.jpg").
func withSidecars(path string) []string {
	out := []string{path}
	dir, base := filepath.Dir(path), filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	es, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range es {
		n := e.Name()
		if n != base && !e.IsDir() && strings.HasPrefix(n, stem+".") {
			out = append(out, filepath.Join(dir, n))
		}
	}
	return out
}

// hidePaths diz o que mover pra esconder o item.
func (a *api) hidePaths(ctx context.Context, it *store.Item, root string) ([]string, error) {
	switch it.Kind {
	case store.KindSeries, store.KindSeason, store.KindRom:
		// pasta da série/temporada, ou arquivo/pasta (multidisco) da ROM
		return []string{it.Path}, nil
	case store.KindMovie:
		// filme na pasta dele: vai a pasta inteira (pôster, legendas...)
		dir := filepath.Dir(it.Path)
		if dir != root && inside(root, dir) {
			var others int
			a.d.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE kind IN ('movie','episode') AND id != ?
AND instr(path, ?) = 1`, it.ID, dir+string(filepath.Separator)).Scan(&others)
			if others == 0 {
				return []string{dir}, nil
			}
		}
		return withSidecars(it.Path), nil
	case store.KindEpisode, store.KindTrack, store.KindFirmware:
		return withSidecars(it.Path), nil
	case store.KindAlbum:
		tracks, err := a.d.Store.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = 'track' AND i.album_id = ?`, it.ID)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, t := range tracks {
			out = append(out, withSidecars(t.Path)...)
		}
		return out, nil
	}
	return nil, bad("não dá pra esconder %s", it.Kind)
}

// move renomeia; entre discos, copia e apaga.
func move(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// POST /items/{id}/hide: move pra <hidden>/<pasta da biblioteca>/<mesmo
// caminho> e tira do acervo.
func (a *api) hide(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	ctx := r.Context()
	if !a.d.Config.Writable {
		failErr(w, errReadOnly)
		return
	}
	hidden := a.d.Config.Hidden
	if hidden == "" {
		fail(w, http.StatusConflict, "pasta de escondidos não configurada (--hidden)")
		return
	}
	it, err := a.item(ctx, r.PathValue("id"))
	if err != nil {
		internal(w, err)
		return
	}
	root, err := a.libRoot(it.Library)
	if err != nil {
		failErr(w, err)
		return
	}
	srcs, err := a.hidePaths(ctx, it, root)
	if err != nil {
		failErr(w, err)
		return
	}
	type pair struct{ src, dst string }
	var moves []pair
	for _, src := range srcs {
		if !inside(root, src) {
			fail(w, http.StatusBadRequest, "item fora da biblioteca")
			return
		}
		if _, err := os.Lstat(src); err != nil {
			continue // já sumiu
		}
		rel, _ := filepath.Rel(root, src)
		dst := filepath.Join(hidden, filepath.Base(root), rel)
		if _, err := os.Lstat(dst); err == nil {
			fail(w, http.StatusConflict, "já existe em escondidos: "+filepath.ToSlash(filepath.Join(filepath.Base(root), rel)))
			return
		}
		moves = append(moves, pair{src, dst})
	}
	for _, m := range moves {
		if err := move(m.src, m.dst); err != nil {
			internal(w, errors.Join(errors.New("esconder"), err))
			return
		}
	}
	// tira do acervo já (o item e o que mora dentro); o scan confirma
	if _, err := a.d.Store.DB.ExecContext(ctx, `DELETE FROM items WHERE id = ? OR parent_id = ? OR series_id = ?
OR season_id = ? OR album_id = ?`, it.ID, it.ID, it.ID, it.ID, it.ID); err != nil {
		internal(w, err)
		return
	}
	a.d.Library.Trigger()
	w.WriteHeader(http.StatusNoContent)
}
