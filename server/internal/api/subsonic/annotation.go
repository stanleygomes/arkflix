package subsonic

import (
	"strconv"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

func starIDs(p params) ([]string, error) {
	ids := append(append(p.strs("id"), p.strs("albumId")...), p.strs("artistId")...)
	if len(ids) == 0 {
		return nil, fail(errMissingParam, "Required id parameter is missing")
	}
	return ids, nil
}

// setStar grava favorito (o mesmo do Jellyfin); id desconhecido é ignorado.
func (a *API) setStar(c *call, on bool) (*resp, error) {
	ids, err := starIDs(c.p)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		switch a.kindOf(c.ctx, id) {
		case "artist", "album", "track":
			if _, err := a.d.Store.SetFavorite(c.ctx, c.user.ID, cleanID(id), on); err != nil {
				return nil, err
			}
		}
	}
	return a.newResp(), nil
}

func (a *API) star(c *call) (*resp, error)   { return a.setStar(c, true) }
func (a *API) unstar(c *call) (*resp, error) { return a.setStar(c, false) }

func (a *API) setRating(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	rating, err := c.p.needInt("rating")
	if err != nil {
		return nil, err
	}
	if rating < 0 || rating > 5 {
		return nil, fail(errGeneric, "Invalid rating: %d. Must be between 0 and 5", rating)
	}
	if k := a.kindOf(c.ctx, id); k != "artist" && k != "album" && k != "track" {
		return nil, fail(errNotFound, "data not found")
	}
	_, err = a.d.Store.UpdateUserData(c.ctx, c.user.ID, cleanID(id), func(d *store.UserData) { d.Rating = rating })
	if err != nil {
		return nil, err
	}
	return a.newResp(), nil
}

// scrobble: submission=true conta a execução; false só avisa "tocando agora".
func (a *API) scrobble(c *call) (*resp, error) {
	ids := c.p.strs("id")
	if len(ids) == 0 {
		return nil, fail(errMissingParam, "missing parameter: 'id'")
	}
	times := c.p.strs("time")
	if len(times) > 0 && len(times) != len(ids) {
		return nil, fail(errGeneric, "Wrong number of timestamps: %d, should be %d", len(times), len(ids))
	}
	if !c.p.bool("submission", true) {
		if t, err := a.oneTrack(c.ctx, c.user.ID, ids[0]); err == nil {
			a.setPlaying(c, t.ID, t.Duration, int64(c.p.int("position", 0))*1000)
		}
		return a.newResp(), nil
	}
	for i, id := range ids {
		if a.kindOf(c.ctx, id) != "track" {
			continue
		}
		at := time.Now()
		if len(times) > 0 {
			if v, err := strconv.ParseInt(times[i], 10, 64); err == nil {
				at = time.UnixMilli(v)
			}
		}
		// não mexe na posição: marcador (audiolivro) continua valendo
		_, err := a.d.Store.UpdateUserData(c.ctx, c.user.ID, cleanID(id), func(d *store.UserData) {
			d.PlayCount++
			d.Played = true
			if at.After(d.LastPlayed) {
				d.LastPlayed = at.UTC()
			}
		})
		if err != nil {
			return nil, err
		}
	}
	return a.newResp(), nil
}
