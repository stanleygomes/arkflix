package jellyfin

import (
	"os"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// srcFile é o arquivo por trás de uma fonte de imagem ("embedded:<faixa>",
// "frame:<vídeo>" ou o próprio arquivo).
func srcFile(src string) string {
	for _, p := range []string{"embedded:", "frame:"} {
		if rest, ok := strings.CutPrefix(src, p); ok {
			return rest
		}
	}
	return src
}

// tag da imagem: hash do caminho + mtime ("" = não tem).
func (c *rc) tag(src string) string {
	if src == "" {
		return ""
	}
	st, err := os.Stat(srcFile(src))
	if err != nil {
		return ""
	}
	return hash(src, strconv.FormatInt(st.ModTime().UnixNano(), 10))
}

func (c *rc) canFrame(it *store.Item) bool {
	return c.s.d.Config.FFmpeg != "" && (it.Kind == store.KindMovie || it.Kind == store.KindEpisode)
}

// artistImage: artista não tem foto; usa a capa de um álbum dele.
func (c *rc) artistImage(it *store.Item) string {
	var img string
	c.s.d.Store.DB.QueryRowContext(c.ctx(), `SELECT image FROM items WHERE kind = 'album' AND artist_id = ? AND image != ''
ORDER BY year DESC, sort_name LIMIT 1`, it.ID).Scan(&img)
	return img
}

// images preenche ImageTags/BackdropImageTags do próprio item.
func (c *rc) images(d *itemDto, it *store.Item) {
	primary := it.Image
	switch {
	case primary == "" && it.Kind == store.KindMovie && c.canFrame(it):
		primary = "frame:" + it.Path
	case primary == "" && it.Kind == store.KindArtist:
		primary = c.artistImage(it)
	}
	if t := c.tag(primary); t != "" {
		d.ImageTags["Primary"] = t
	}
	if t := c.tag(it.Backdrop); t != "" {
		d.BackdropImageTags = []string{t}
	}
	if t := c.tag(it.Logo); t != "" {
		d.ImageTags["Logo"] = t
	}
}

// imageSrc escolhe a imagem com a cadeia de reserva de cada tipo.
func (c *rc) imageSrc(it *store.Item, typ string) string {
	var chain []string
	series := c.lookup(it.SeriesID)
	album := c.lookup(it.AlbumID)
	frame := ""
	if c.canFrame(it) {
		frame = "frame:" + it.Path
	}
	switch strings.ToLower(typ) {
	case "primary":
		chain = append(chain, it.Image)
		if series != nil {
			chain = append(chain, series.Image)
		}
		if album != nil {
			chain = append(chain, album.Image)
		}
		if it.Kind == store.KindArtist {
			chain = append(chain, c.artistImage(it))
		}
		chain = append(chain, frame)
	case "backdrop":
		chain = append(chain, it.Backdrop)
		if series != nil {
			chain = append(chain, series.Backdrop)
		}
		chain = append(chain, frame)
	case "logo":
		chain = append(chain, it.Logo)
		if series != nil {
			chain = append(chain, series.Logo)
		}
	case "thumb":
		chain = append(chain, it.Backdrop)
		if series != nil {
			chain = append(chain, series.Backdrop)
		}
		chain = append(chain, frame, it.Image)
		if series != nil {
			chain = append(chain, series.Image)
		}
		if album != nil {
			chain = append(chain, album.Image)
		}
	}
	for _, src := range chain {
		if c.tag(src) != "" {
			return src
		}
	}
	return ""
}

// viewImage: a capa da biblioteca é a do item mais novo com imagem.
func (c *rc) viewImage(v view) string {
	var img string
	c.s.d.Store.DB.QueryRowContext(c.ctx(), `SELECT image FROM items WHERE library = ? AND image != ''
AND kind IN ('movie','series','album') ORDER BY added_at DESC LIMIT 1`, v.Lib).Scan(&img)
	return img
}

func (c *rc) imageOpts() media.ImageOpts {
	o := media.ImageOpts{
		MaxWidth: c.qi("maxWidth", 0), MaxHeight: c.qi("maxHeight", 0),
		Width: c.qi("fillWidth", 0), Height: c.qi("fillHeight", 0),
		Quality: c.qi("quality", 0), Format: strings.ToLower(c.qs("format")),
	}
	// width/height do Jellyfin redimensionam sem cortar
	if w := c.qi("width", 0); w > 0 && o.MaxWidth == 0 && o.Width == 0 {
		o.MaxWidth = w
	}
	if h := c.qi("height", 0); h > 0 && o.MaxHeight == 0 && o.Height == 0 {
		o.MaxHeight = h
	}
	return o
}

// GET/HEAD /Items/{id}/Images/{type}[/{index}] (sem login, como no Jellyfin)
func (s *Server) itemImage(c *rc) {
	id := c.pid("id")
	var src string
	if v, ok := s.viewByID(id); ok {
		if strings.EqualFold(c.p["type"], "primary") {
			src = c.viewImage(v)
		}
	} else if it := c.lookup(id); it != nil {
		src = c.imageSrc(it, c.p["type"])
	}
	if src == "" {
		c.notFound()
		return
	}
	s.d.Media.Image(c.w, c.r, src, c.imageOpts())
}

// GET /Artists/{name}/Images/{type}[/{index}]
func (s *Server) artistImage(c *rc) {
	it, err := s.artistByName(c, c.p["name"])
	if err != nil {
		c.notFound()
		return
	}
	src := c.imageSrc(it, c.p["type"])
	if src == "" {
		c.notFound()
		return
	}
	s.d.Media.Image(c.w, c.r, src, c.imageOpts())
}

// GET /Items/{id}/Images: a lista das imagens que o item tem.
func (s *Server) imageInfos(c *rc) {
	it := c.lookup(c.pid("id"))
	out := []map[string]any{}
	if it != nil {
		for _, typ := range []string{"Primary", "Backdrop", "Logo"} {
			src := c.imageSrc(it, typ)
			if src == "" {
				continue
			}
			info := map[string]any{"ImageType": typ, "ImageTag": c.tag(src), "Width": 0, "Height": 0}
			if typ == "Backdrop" {
				info["ImageIndex"] = 0
			}
			if f, err := os.Stat(srcFile(src)); err == nil && !strings.HasPrefix(src, "embedded:") && !strings.HasPrefix(src, "frame:") {
				info["Path"], info["Size"] = src, f.Size()
			}
			out = append(out, info)
		}
	}
	c.json(out)
}
