package subsonic

import (
	"context"
	"errors"
	"slices"
	"strconv"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// As playlists são as listas de internal/lists (as mesmas do Jellyfin, do
// RomM e do app): aqui aparecem as que têm faixa (ou estão vazias), só com
// as faixas.

func (a *API) lists() *lists.Lists { return lists.New(a.d.Store) }

// plLoad: a playlist é do usuário ou é pública.
func (a *API) plLoad(c *call, id string) (*lists.List, error) {
	l, err := a.lists().Get(c.ctx, cleanID(id))
	if errors.Is(err, store.ErrNotFound) || (err == nil && !l.CanSee(c.user.ID)) {
		return nil, fail(errNotFound, "playlist not found")
	}
	return l, err
}

// plItemIDs são as faixas da lista, na ordem (com repetição).
func (a *API) plItemIDs(ctx context.Context, id string) ([]string, error) {
	es, err := a.lists().Entries(ctx, id)
	if err != nil {
		return nil, err
	}
	return lists.ItemIDs(es, store.KindTrack), nil
}

func (a *API) plBuild(c *call, l *lists.List, ts []*track) playlist {
	pl := playlist{ID: l.ID, Name: l.Name, Comment: l.Comment, Public: l.Public, Owner: l.Owner,
		Created: ms(l.Created), Changed: ms(l.Updated), SongCount: len(ts), Readonly: l.UserID != c.user.ID}
	var dur float64
	for _, t := range ts {
		dur += t.Duration
		if pl.CoverArt == "" {
			pl.CoverArt = t.cover()
		}
	}
	pl.Duration = int(dur)
	return pl
}

func (a *API) getPlaylists(c *call) (*resp, error) {
	ls, err := a.lists().Visible(c.ctx, c.user.ID)
	if err != nil {
		return nil, err
	}
	out := &playlists{}
	for _, l := range ls {
		es, err := a.lists().Entries(c.ctx, l.ID)
		if err != nil {
			return nil, err
		}
		if !lists.HasKinds(es, store.KindTrack) {
			continue
		}
		ts, err := a.tracksByIDs(c.ctx, c.user.ID, lists.ItemIDs(es, store.KindTrack))
		if err != nil {
			return nil, err
		}
		out.Playlist = append(out.Playlist, a.plBuild(c, l, ts))
	}
	res := a.newResp()
	res.Playlists = out
	return res, nil
}

func (a *API) playlistResp(c *call, id string) (*resp, error) {
	l, err := a.plLoad(c, id)
	if err != nil {
		return nil, err
	}
	ids, err := a.plItemIDs(c.ctx, l.ID)
	if err != nil {
		return nil, err
	}
	ts, err := a.tracksByIDs(c.ctx, c.user.ID, ids)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.Playlist = &playlistFull{playlist: a.plBuild(c, l, ts), Entry: trackChildren(ts)}
	return res, nil
}

func (a *API) getPlaylist(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	return a.playlistResp(c, id)
}

// own carrega a playlist e exige que seja do usuário (senão 50).
func (a *API) own(c *call, id string) (*lists.List, error) {
	l, err := a.plLoad(c, id)
	if err != nil {
		return nil, err
	}
	if l.UserID != c.user.ID {
		return nil, fail(errNotAllowed, "")
	}
	return l, nil
}

func cleanIDs(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = cleanID(id)
	}
	return out
}

// setTracks troca as faixas da lista (o resto da lista fica).
func (a *API) setTracks(ctx context.Context, listID string, ids []string) error {
	return a.lists().SetKinds(ctx, listID, []string{store.KindTrack}, cleanIDs(ids))
}

// createPlaylist cria; com playlistId troca as faixas da existente.
func (a *API) createPlaylist(c *call) (*resp, error) {
	plID, name, songIDs := c.p.str("playlistId"), c.p.str("name"), c.p.strs("songId")
	if plID == "" && name == "" {
		return nil, fail(errMissingParam, "required parameter name is missing")
	}
	if plID != "" {
		l, err := a.own(c, plID)
		if err != nil {
			return nil, err
		}
		if name != "" {
			if err := a.lists().Update(c.ctx, l.ID, lists.Changes{Name: &name}); err != nil {
				return nil, err
			}
		}
		if err := a.setTracks(c.ctx, l.ID, songIDs); err != nil {
			return nil, err
		}
		return a.playlistResp(c, l.ID)
	}
	l, err := a.lists().Create(c.ctx, c.user.ID, name)
	if err != nil {
		return nil, err
	}
	if err := a.setTracks(c.ctx, l.ID, songIDs); err != nil {
		return nil, err
	}
	return a.playlistResp(c, l.ID)
}

// updatePlaylist: name, comment, public, tira por índice (das faixas) e
// acrescenta no fim.
func (a *API) updatePlaylist(c *call) (*resp, error) {
	plID, err := c.p.need("playlistId")
	if err != nil {
		return nil, err
	}
	l, err := a.own(c, plID)
	if err != nil {
		return nil, err
	}
	ids, err := a.plItemIDs(c.ctx, l.ID)
	if err != nil {
		return nil, err
	}
	var drop []int
	for _, s := range c.p.strs("songIndexToRemove") {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < len(ids) {
			drop = append(drop, n)
		}
	}
	var keep []string
	for i, id := range ids {
		if !slices.Contains(drop, i) {
			keep = append(keep, id)
		}
	}
	keep = append(keep, c.p.strs("songIdToAdd")...)
	var ch lists.Changes
	if c.p.has("name") {
		n := c.p.str("name")
		ch.Name = &n
	}
	if c.p.has("comment") {
		cm := c.p.str("comment")
		ch.Comment = &cm
	}
	if c.p.has("public") {
		pub := c.p.bool("public", false)
		ch.Public = &pub
	}
	if err := a.lists().Update(c.ctx, l.ID, ch); err != nil && !errors.Is(err, lists.ErrName) {
		return nil, err
	}
	if err := a.setTracks(c.ctx, l.ID, keep); err != nil {
		return nil, err
	}
	return a.newResp(), nil
}

func (a *API) deletePlaylist(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	l, err := a.own(c, id)
	if err != nil {
		return nil, err
	}
	if err := a.lists().Delete(c.ctx, l.ID); err != nil {
		return nil, err
	}
	return a.newResp(), nil
}
