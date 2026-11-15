package jellyfin

import (
	"net/http"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// udResponse devolve o UserItemDataDto atual do item.
func (c *rc) udResponse(it *store.Item) {
	ud, err := c.s.d.Store.UserData(c.ctx(), c.uid(), it.ID)
	if err != nil {
		c.err(err)
		return
	}
	c.json(c.userData(it, ud, nil))
}

func (c *rc) item() *store.Item {
	it := c.lookup(c.pid("id"))
	if it == nil || kindType[it.Kind] == "" {
		c.notFound()
		return nil
	}
	return it
}

// POST/DELETE /Users/{userId}/FavoriteItems/{id} e /UserFavoriteItems/{id}
func (s *Server) favorite(c *rc) {
	it := c.item()
	if it == nil {
		return
	}
	if _, err := s.d.Store.SetFavorite(c.ctx(), c.uid(), it.ID, c.r.Method == http.MethodPost); err != nil {
		c.err(err)
		return
	}
	c.udResponse(it)
}

// children: o que "marcar como visto" uma pasta alcança.
func (c *rc) children(it *store.Item) ([]*store.Item, error) {
	cols := `SELECT ` + store.ItemColumns + ` FROM items i WHERE `
	switch it.Kind {
	case store.KindSeries:
		return c.s.d.Store.Items(c.ctx(), cols+`i.kind = 'episode' AND i.series_id = ?`, it.ID)
	case store.KindSeason:
		return c.s.d.Store.Items(c.ctx(), cols+`i.kind = 'episode' AND i.season_id = ?`, it.ID)
	case store.KindAlbum:
		return c.s.d.Store.Items(c.ctx(), cols+`i.kind = 'track' AND i.album_id = ?`, it.ID)
	case store.KindArtist:
		return c.s.d.Store.Items(c.ctx(), cols+`i.kind = 'track' AND (i.artist_id = ? OR i.album_id IN
(SELECT id FROM items WHERE kind = 'album' AND artist_id = ?))`, it.ID, it.ID)
	}
	return nil, nil
}

// POST/DELETE /Users/{userId}/PlayedItems/{id} e /UserPlayedItems/{id}.
// Série/temporada/álbum marcam todos os filhos.
func (s *Server) played(c *rc) {
	it := c.item()
	if it == nil {
		return
	}
	on := c.r.Method == http.MethodPost
	at := time.Now()
	if d := parseTime(c.qs("DatePlayed")); !d.IsZero() {
		at = d
	}
	kids, err := c.children(it)
	if err != nil {
		c.err(err)
		return
	}
	for _, k := range append(kids, it) {
		if _, err := s.d.Store.SetPlayed(c.ctx(), c.uid(), k.ID, on, at); err != nil {
			c.err(err)
			return
		}
	}
	c.udResponse(it)
}

// GET/POST /Items/{id}/UserData, /UserItems/{id}/UserData
func (s *Server) itemUserData(c *rc) {
	it := c.item()
	if it == nil {
		return
	}
	if c.r.Method == http.MethodPost {
		var body struct {
			PlaybackPositionTicks *int64
			PlayCount             *int
			IsFavorite            *bool
			Played                *bool
			LastPlayedDate        *string
		}
		if err := c.body(&body); err != nil {
			c.fail(http.StatusBadRequest, "corpo inválido")
			return
		}
		_, err := s.d.Store.UpdateUserData(c.ctx(), c.uid(), it.ID, func(d *store.UserData) {
			if body.PlaybackPositionTicks != nil {
				d.Position = float64(*body.PlaybackPositionTicks) / ticksPerSecond
			}
			if body.PlayCount != nil {
				d.PlayCount = *body.PlayCount
			}
			if body.IsFavorite != nil && *body.IsFavorite != d.Favorite {
				d.Favorite = *body.IsFavorite
				d.FavoriteAt = time.Time{}
				if d.Favorite {
					d.FavoriteAt = time.Now().UTC()
				}
			}
			if body.Played != nil {
				d.Played = *body.Played
			}
			if body.LastPlayedDate != nil {
				d.LastPlayed = parseTime(*body.LastPlayedDate)
			}
		})
		if err != nil {
			c.err(err)
			return
		}
	}
	c.udResponse(it)
}

// POST/DELETE .../Rating: Likes vira favorito (é o que o Jellyfin mostra).
func (s *Server) rating(c *rc) {
	it := c.item()
	if it == nil {
		return
	}
	on := c.r.Method == http.MethodPost
	if v, ok := c.qb("Likes"); ok && on {
		on = v
	}
	if _, err := s.d.Store.SetFavorite(c.ctx(), c.uid(), it.ID, on); err != nil {
		c.err(err)
		return
	}
	c.udResponse(it)
}

// playReport é o PlaybackStartInfo/ProgressInfo/StopInfo.
type playReport struct {
	ItemId        string
	PositionTicks *int64
	PlaySessionId string
	MediaSourceId string
	IsPaused      bool
}

// report grava o progresso: start só mexe se vier posição; stop perto do
// fim vira assistido (store.Progress).
func (s *Server) report(c *rc, event string, rep playReport) {
	it := c.lookup(store.NormalizeID(rep.ItemId))
	if it != nil && kindType[it.Kind] != "" && rep.PositionTicks != nil && (event != "start" || *rep.PositionTicks > 0) {
		if _, err := s.d.Store.Progress(c.ctx(), c.uid(), it, float64(*rep.PositionTicks)/ticksPerSecond); err != nil {
			c.err(err)
			return
		}
	}
	var pos int64
	if rep.PositionTicks != nil {
		pos = *rep.PositionTicks
	}
	if event == "stop" {
		s.sessions.playing(c.tok.Token, "", "", 0, false)
		s.hls.stopSession(rep.PlaySessionId)
	} else if it != nil {
		s.sessions.playing(c.tok.Token, it.ID, rep.PlaySessionId, pos, rep.IsPaused)
	}
	c.noContent()
}

// POST /Sessions/Playing[/Progress|/Stopped]
func (s *Server) reportBody(event string) func(*rc) {
	return func(c *rc) {
		var rep playReport
		if err := c.body(&rep); err != nil {
			c.fail(http.StatusBadRequest, "corpo inválido")
			return
		}
		s.report(c, event, rep)
	}
}

// legado: /Users/{userId}/PlayingItems/{id}[/Progress] (POST) e DELETE = parou
func (s *Server) reportLegacy(event string) func(*rc) {
	return func(c *rc) {
		if event == "start" && c.r.Method == http.MethodDelete {
			event = "stop"
		}
		rep := playReport{ItemId: c.p["id"], PlaySessionId: c.qs("PlaySessionId"), MediaSourceId: c.qs("MediaSourceId")}
		if c.qs("PositionTicks") != "" {
			p := c.qi64("PositionTicks")
			rep.PositionTicks = &p
		}
		rep.IsPaused, _ = c.qb("IsPaused")
		s.report(c, event, rep)
	}
}

// POST /Sessions/Playing/Ping: mantém o HLS vivo.
func (s *Server) playingPing(c *rc) {
	ps := c.qs("PlaySessionId")
	s.hls.mu.Lock()
	for _, j := range s.hls.jobs {
		if ps != "" && j.playSession == ps {
			j.mu.Lock()
			j.lastAccess = time.Now()
			j.mu.Unlock()
		}
	}
	s.hls.mu.Unlock()
	c.noContent()
}
