package admin

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// item acha pelo id (32 hex) ou "rom:<rid>" (o id do RomM).
func (a *api) item(ctx context.Context, id string) (*store.Item, error) {
	if rest, ok := strings.CutPrefix(id, "rom:"); ok {
		rid, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			return nil, store.ErrNotFound
		}
		it, err := a.d.Store.ItemByRID(ctx, rid)
		if err != nil || it.Kind != store.KindRom {
			return nil, store.ErrNotFound
		}
		return it, nil
	}
	return a.d.Store.Item(ctx, id)
}

// srcFile é o arquivo por trás de uma fonte de imagem.
func srcFile(src string) string {
	for _, p := range []string{"embedded:", "frame:"} {
		if rest, ok := strings.CutPrefix(src, p); ok {
			return rest
		}
	}
	return src
}

func exists(src string) bool {
	if src == "" {
		return false
	}
	st, err := os.Stat(srcFile(src))
	return err == nil && !st.IsDir()
}

// imageSrc é a capa com a mesma reserva do Jellyfin: a do item, a da
// série, a do álbum, a de um álbum do artista e, pra vídeo, um quadro.
func (a *api) imageSrc(ctx context.Context, it *store.Item) string {
	chain := []string{it.Image}
	parent := func(id string) {
		if id == "" {
			return
		}
		if p, err := a.d.Store.Item(ctx, id); err == nil {
			chain = append(chain, p.Image)
		}
	}
	parent(it.SeriesID)
	parent(it.AlbumID)
	if it.Kind == store.KindArtist {
		var img string
		a.d.Store.DB.QueryRowContext(ctx, `SELECT image FROM items WHERE kind = 'album' AND artist_id = ? AND image != ''
ORDER BY year DESC, sort_name LIMIT 1`, it.ID).Scan(&img)
		chain = append(chain, img)
	}
	if a.d.Config.FFmpeg != "" && (it.Kind == store.KindMovie || it.Kind == store.KindEpisode) {
		chain = append(chain, "frame:"+it.Path)
	}
	for _, src := range chain {
		if exists(src) {
			return src
		}
	}
	return ""
}

// itemJSON é o Item do contrato.
type itemJSON struct {
	ID         string `json:"id"`
	RID        int64  `json:"rid"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Year       int    `json:"year"`
	Platform   string `json:"platform,omitempty"`
	SeriesName string `json:"seriesName,omitempty"`
	Image      string `json:"image"`
}

func (a *api) itemJSON(ctx context.Context, it *store.Item) itemJSON {
	out := itemJSON{ID: it.ID, RID: it.RID, Kind: it.Kind, Name: it.Name, Year: it.Year, Platform: it.Platform}
	if it.SeriesID != "" {
		if s, err := a.d.Store.Item(ctx, it.SeriesID); err == nil {
			out.SeriesName = s.Name
		}
	}
	if a.imageSrc(ctx, it) != "" {
		out.Image = Prefix + "/items/" + it.ID + "/image"
	}
	return out
}

// GET /items/{id}/image?maxWidth=&maxHeight=
func (a *api) image(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	it, err := a.item(r.Context(), r.PathValue("id"))
	if err != nil {
		internal(w, err)
		return
	}
	src := a.imageSrc(r.Context(), it)
	if src == "" {
		fail(w, http.StatusNotFound, "sem imagem")
		return
	}
	q := r.URL.Query()
	mw, _ := strconv.Atoi(q.Get("maxWidth"))
	mh, _ := strconv.Atoi(q.Get("maxHeight"))
	a.d.Media.Image(w, r, src, media.ImageOpts{MaxWidth: max(mw, 0), MaxHeight: max(mh, 0)})
}
