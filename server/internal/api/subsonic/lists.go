package subsonic

import (
	"strconv"
	"time"
)

const genreMatch = ` AND EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE lower(g.value) = lower(?))`

// albumListQuery traduz o type do getAlbumList em where + ordem.
func albumListQuery(p params) (where, order string, args []any, err error) {
	typ, err := p.need("type")
	if err != nil {
		return "", "", nil, err
	}
	switch typ {
	case "random":
		order = ` ORDER BY RANDOM()`
	case "newest":
		order = ` ORDER BY i.added_at DESC, i.rid DESC`
	case "highest":
		where, order = ` AND COALESCE(u.rating,0) > 0`, ` ORDER BY u.rating DESC, i.sort_name`
	case "frequent":
		where, order = ` AND COALESCE(s.plays,0) > 0`, ` ORDER BY s.plays DESC, i.sort_name`
	case "recent":
		where, order = ` AND s.last IS NOT NULL`, ` ORDER BY s.last DESC`
	case "alphabeticalByName":
		order = ` ORDER BY i.sort_name, i.rid`
	case "alphabeticalByArtist":
		order = ` ORDER BY i.album_artist COLLATE NOCASE, i.year, i.sort_name, i.rid`
	case "starred":
		where, order = ` AND u.favorite = 1`, ` ORDER BY u.favorite_at DESC`
	case "byYear":
		from, err := p.needInt("fromYear")
		if err != nil {
			return "", "", nil, err
		}
		to, err := p.needInt("toYear")
		if err != nil {
			return "", "", nil, err
		}
		where, order = ` AND i.year BETWEEN ? AND ?`, ` ORDER BY i.year, i.sort_name`
		if from > to {
			from, to, order = to, from, ` ORDER BY i.year DESC, i.sort_name`
		}
		args = []any{from, to}
	case "byGenre":
		g, err := p.need("genre")
		if err != nil {
			return "", "", nil, err
		}
		where, order, args = genreMatch, ` ORDER BY i.sort_name`, []any{g}
	default:
		return "", "", nil, fail(errGeneric, "type '%s' not implemented", typ)
	}
	return where, order, args, nil
}

func (a *API) albumListPage(c *call) ([]*album, error) {
	where, order, args, err := albumListQuery(c.p)
	if err != nil {
		return nil, err
	}
	if err := checkFolder(c.p); err != nil {
		return nil, err
	}
	size := min(c.p.int("size", 10), 500)
	offset := max(c.p.int("offset", 0), 0)
	var total int
	a.d.Store.DB.QueryRowContext(c.ctx, `SELECT COUNT(*) FROM (`+albumSelect+where+`)`, append([]any{c.user.ID}, args...)...).Scan(&total)
	c.w.Header().Set("x-total-count", strconv.Itoa(total))
	return a.albums(c.ctx, c.user.ID, where+order+` LIMIT ? OFFSET ?`, append(args, max(size, 0), offset)...)
}

func (a *API) albumList(c *call) (*resp, error) {
	als, err := a.albumListPage(c)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.AlbumList = &albumList{Album: albumChildren(als)}
	return res, nil
}

func (a *API) albumList2(c *call) (*resp, error) {
	als, err := a.albumListPage(c)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.AlbumList2 = &albumList2{Album: albumID3s(als)}
	return res, nil
}

func (a *API) randomSongs(c *call) (*resp, error) {
	if err := checkFolder(c.p); err != nil {
		return nil, err
	}
	size := min(c.p.int("size", 10), 500)
	where, args := "", []any{}
	if g := c.p.str("genre"); g != "" {
		where, args = where+genreMatch, append(args, g)
	}
	if y := c.p.int("fromYear", 0); y > 0 {
		where, args = where+` AND i.year >= ?`, append(args, y)
	}
	if y := c.p.int("toYear", 0); y > 0 {
		where, args = where+` AND i.year <= ?`, append(args, y)
	}
	ts, err := a.tracks(c.ctx, c.user.ID, where+` ORDER BY RANDOM() LIMIT ?`, append(args, max(size, 0))...)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.RandomSongs = &songs{Songs: trackChildren(ts)}
	return res, nil
}

func (a *API) songsByGenre(c *call) (*resp, error) {
	if err := checkFolder(c.p); err != nil {
		return nil, err
	}
	count := min(c.p.int("count", 10), 500)
	offset := max(c.p.int("offset", 0), 0)
	ts, err := a.tracks(c.ctx, c.user.ID, genreMatch+sortLib+` LIMIT ? OFFSET ?`, c.p.str("genre"), max(count, 0), offset)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.SongsByGenre = &songs{Songs: trackChildren(ts)}
	return res, nil
}

// starredAll são os favoritos (estrela = favorito do Jellyfin).
func (a *API) starredAll(c *call) ([]*artistRow, []*album, []*track, error) {
	if err := checkFolder(c.p); err != nil {
		return nil, nil, nil, err
	}
	ars, err := a.artists(c.ctx, c.user.ID, `AND u.favorite = 1 ORDER BY u.favorite_at DESC`)
	if err != nil {
		return nil, nil, nil, err
	}
	als, err := a.albums(c.ctx, c.user.ID, `AND u.favorite = 1 ORDER BY u.favorite_at DESC`)
	if err != nil {
		return nil, nil, nil, err
	}
	ts, err := a.tracks(c.ctx, c.user.ID, `AND u.favorite = 1 ORDER BY u.favorite_at DESC`)
	return ars, als, ts, err
}

func (a *API) starred(c *call) (*resp, error) {
	ars, als, ts, err := a.starredAll(c)
	if err != nil {
		return nil, err
	}
	out := &searchResult2{Album: albumChildren(als), Song: trackChildren(ts)}
	for _, ar := range ars {
		out.Artist = append(out.Artist, ar.plain())
	}
	res := a.newResp()
	res.Starred = out
	return res, nil
}

func (a *API) starred2(c *call) (*resp, error) {
	ars, als, ts, err := a.starredAll(c)
	if err != nil {
		return nil, err
	}
	out := &searchResult3{Album: albumID3s(als), Song: trackChildren(ts)}
	for _, ar := range ars {
		out.Artist = append(out.Artist, ar.id3())
	}
	res := a.newResp()
	res.Starred2 = out
	return res, nil
}

// playing é o que um cliente de um usuário está tocando (só em memória).
type playing struct {
	userID, user, client, itemID string
	start                        time.Time
	posMs                        int64
	dur                          float64
}

// setPlaying marca a faixa como tocando agora (stream e scrobble sem submission).
func (a *API) setPlaying(c *call, itemID string, dur float64, posMs int64) {
	a.npMu.Lock()
	defer a.npMu.Unlock()
	a.np[c.user.ID+"\x00"+c.client] = &playing{userID: c.user.ID, user: c.user.Name, client: c.client,
		itemID: itemID, start: time.Now().Add(-time.Duration(posMs) * time.Millisecond), posMs: posMs, dur: dur}
}

// fresh tira o que já devia ter acabado (duração + 5 min de folga).
func (a *API) fresh() []playing {
	a.npMu.Lock()
	defer a.npMu.Unlock()
	var out []playing
	for k, p := range a.np {
		if time.Since(p.start) > time.Duration(p.dur*float64(time.Second))+5*time.Minute {
			delete(a.np, k)
			continue
		}
		out = append(out, *p)
	}
	return out
}

func (a *API) nowPlaying(c *call) (*resp, error) {
	out := &nowPlaying{}
	list := a.fresh()
	for i, p := range list {
		t, err := a.oneTrack(c.ctx, p.userID, p.itemID)
		if err != nil {
			continue
		}
		pos := time.Since(p.start).Milliseconds()
		if max := int64(p.dur * 1000); max > 0 && pos > max {
			pos = max
		}
		out.Entry = append(out.Entry, nowPlayingEntry{child: t.child(), Username: p.user,
			MinutesAgo: int(time.Since(p.start).Minutes()), PlayerID: i + 1, PlayerName: p.client,
			State: "playing", PositionMs: pos, PlaybackRate: 1})
	}
	res := a.newResp()
	res.NowPlaying = out
	return res, nil
}
