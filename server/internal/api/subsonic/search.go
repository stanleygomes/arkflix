package subsonic

import (
	"context"
	"strings"
)

type searchParams struct {
	words                                 []string
	artistCount, albumCount, songCount    int
	artistOffset, albumOffset, songOffset int
}

// query "" ou `""` traz tudo (o Symfonium sincroniza assim, paginando).
func readSearch(p params) searchParams {
	q := strings.TrimSpace(p.str("query"))
	q = strings.TrimSuffix(q, "*")
	q = strings.Trim(q, `"`)
	return searchParams{
		words:        strings.Fields(norm(q)),
		artistCount:  p.int("artistCount", 20),
		albumCount:   p.int("albumCount", 20),
		songCount:    p.int("songCount", 20),
		artistOffset: max(p.int("artistOffset", 0), 0),
		albumOffset:  max(p.int("albumOffset", 0), 0),
		songOffset:   max(p.int("songOffset", 0), 0),
	}
}

// matchIDs lê só id + texto, filtra pelas palavras e devolve a página.
func (a *API) matchIDs(ctx context.Context, query string, words []string, offset, count int) ([]string, error) {
	if count <= 0 {
		return nil, nil
	}
	rows, err := a.d.Store.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	skip := offset
	for rows.Next() {
		var id, text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		if !hasAll(norm(text), words) {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		out = append(out, id)
		if len(out) >= count {
			break
		}
	}
	return out, rows.Err()
}

func hasAll(text string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}

func (a *API) searchAll(c *call) ([]*artistRow, []*album, []*track, error) {
	if err := checkFolder(c.p); err != nil {
		return nil, nil, nil, err
	}
	sp := readSearch(c.p)
	ids, err := a.matchIDs(c.ctx, `SELECT id, name FROM items i WHERE kind = 'artist' ORDER BY sort_name, rid`,
		sp.words, sp.artistOffset, sp.artistCount)
	if err != nil {
		return nil, nil, nil, err
	}
	ars, err := a.artistsByIDs(c.ctx, c.user.ID, ids)
	if err != nil {
		return nil, nil, nil, err
	}
	ids, err = a.matchIDs(c.ctx, `SELECT id, name || ' ' || album_artist || ' ' || artist FROM items i WHERE kind = 'album' ORDER BY sort_name, rid`,
		sp.words, sp.albumOffset, sp.albumCount)
	if err != nil {
		return nil, nil, nil, err
	}
	als, err := a.albumsByIDs(c.ctx, c.user.ID, ids)
	if err != nil {
		return nil, nil, nil, err
	}
	ids, err = a.matchIDs(c.ctx, `SELECT id, name || ' ' || artist || ' ' || album || ' ' || album_artist FROM items i WHERE kind = 'track'`+sortLib,
		sp.words, sp.songOffset, sp.songCount)
	if err != nil {
		return nil, nil, nil, err
	}
	ts, err := a.tracksByIDs(c.ctx, c.user.ID, ids)
	return ars, als, ts, err
}

func (a *API) search2(c *call) (*resp, error) {
	ars, als, ts, err := a.searchAll(c)
	if err != nil {
		return nil, err
	}
	out := &searchResult2{Album: albumChildren(als), Song: trackChildren(ts)}
	for _, ar := range ars {
		out.Artist = append(out.Artist, ar.plain())
	}
	res := a.newResp()
	res.SearchResult2 = out
	return res, nil
}

func (a *API) search3(c *call) (*resp, error) {
	ars, als, ts, err := a.searchAll(c)
	if err != nil {
		return nil, err
	}
	out := &searchResult3{Album: albumID3s(als), Song: trackChildren(ts)}
	for _, ar := range ars {
		out.Artist = append(out.Artist, ar.id3())
	}
	res := a.newResp()
	res.SearchResult3 = out
	return res, nil
}
