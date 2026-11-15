package subsonic

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"
)

// checkFolder valida musicFolderId: só existe a pasta 1.
func checkFolder(p params) error {
	for _, v := range p.strs("musicFolderId") {
		if n, err := strconv.Atoi(v); err != nil || n != folderID {
			return fail(errNotFound, "Library %s not found or not accessible", v)
		}
	}
	return nil
}

func (a *API) musicFolders(c *call) (*resp, error) {
	res := a.newResp()
	res.MusicFolders = &musicFolders{Folders: []musicFolder{{ID: folderID, Name: folderName}}}
	return res, nil
}

// lastModified é a última mudança no acervo de música (ms).
func (a *API) lastModified(ctx context.Context) int64 {
	var s string
	a.d.Store.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(updated_at),'') FROM items WHERE library = 'music'`).Scan(&s)
	if t := parseT(s); !t.IsZero() {
		return t.UnixMilli()
	}
	return time.Now().UnixMilli()
}

// indexArtists são os artistas com álbum, na ordem do índice.
func (a *API) indexArtists(c *call) ([]*artistRow, int64, error) {
	if err := checkFolder(c.p); err != nil {
		return nil, 0, err
	}
	mod := a.lastModified(c.ctx)
	if since := c.p.millis("ifModifiedSince"); !since.IsZero() && mod <= since.UnixMilli() {
		return nil, mod, nil
	}
	all, err := a.artists(c.ctx, c.user.ID, `AND EXISTS (SELECT 1 FROM items x WHERE x.kind = 'album' AND x.artist_id = i.id)`)
	if err != nil {
		return nil, 0, err
	}
	if len(all) == 0 {
		return nil, 0, fail(errNotFound, "Library not found or empty")
	}
	sort.SliceStable(all, func(i, j int) bool {
		return norm(stripArticle(all[i].Name)) < norm(stripArticle(all[j].Name))
	})
	return all, mod, nil
}

func (a *API) getIndexes(c *call) (*resp, error) {
	all, mod, err := a.indexArtists(c)
	if err != nil {
		return nil, err
	}
	out := &indexes{LastModified: mod, IgnoredArticles: ignoredArticles}
	for _, ar := range all {
		name := indexName(ar.Name)
		if n := len(out.Index); n == 0 || out.Index[n-1].Name != name {
			out.Index = append(out.Index, index{Name: name})
		}
		last := &out.Index[len(out.Index)-1]
		last.Artists = append(last.Artists, ar.plain())
	}
	res := a.newResp()
	res.Indexes = out
	return res, nil
}

func (a *API) getArtists(c *call) (*resp, error) {
	all, mod, err := a.indexArtists(c)
	if err != nil {
		return nil, err
	}
	out := &artistsID3{LastModified: mod, IgnoredArticles: ignoredArticles}
	for _, ar := range all {
		name := indexName(ar.Name)
		if n := len(out.Index); n == 0 || out.Index[n-1].Name != name {
			out.Index = append(out.Index, indexID3{Name: name})
		}
		last := &out.Index[len(out.Index)-1]
		last.Artists = append(last.Artists, ar.id3())
	}
	res := a.newResp()
	res.Artists = out
	return res, nil
}

const albumOrder = ` ORDER BY i.year, i.sort_name, i.rid`

func (a *API) artistAlbums(c *call, artistID string) ([]*album, error) {
	return a.albums(c.ctx, c.user.ID, `AND i.artist_id = ?`+albumOrder, artistID)
}

func (a *API) albumTracks(c *call, albumID string) ([]*track, error) {
	return a.tracks(c.ctx, c.user.ID, `AND i.album_id = ? ORDER BY i.parent_idx, i.idx, i.sort_name, i.rid`, albumID)
}

// musicDirectory: artista → álbuns, álbum → faixas.
func (a *API) musicDirectory(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	dir := &directory{}
	switch a.kindOf(c.ctx, id) {
	case "artist":
		ar, err := a.oneArtist(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		als, err := a.artistAlbums(c, ar.ID)
		if err != nil {
			return nil, err
		}
		dir.ID, dir.Name, dir.Starred, dir.UserRating, dir.AlbumCount = ar.ID, ar.Name, starred(ar.ud), ar.rating, ar.albums
		for _, al := range als {
			dir.PlayCount += al.plays
		}
		dir.Child = albumChildren(als)
	case "album":
		al, err := a.oneAlbum(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		ts, err := a.albumTracks(c, al.ID)
		if err != nil {
			return nil, err
		}
		dir.ID, dir.Name, dir.Parent, dir.PlayCount = al.ID, al.Name, al.ArtistID, al.plays
		dir.UserRating, dir.SongCount, dir.CoverArt, dir.Starred = al.rating, al.songs, al.coverID(), starred(al.ud)
		if al.plays > 0 {
			dir.Played = tp(al.last)
		}
		dir.Child = trackChildren(ts)
	default:
		return nil, fail(errNotFound, "Directory not found")
	}
	res := a.newResp()
	res.Directory = dir
	return res, nil
}

func (a *API) getArtist(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	ar, err := a.oneArtist(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, fail(errNotFound, "Artist not found")
	}
	als, err := a.artistAlbums(c, ar.ID)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.Artist = &artistFull{artistID3: ar.id3(), Album: albumID3s(als)}
	return res, nil
}

func (a *API) getAlbum(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	al, err := a.oneAlbum(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, fail(errNotFound, "Album not found")
	}
	ts, err := a.albumTracks(c, al.ID)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.Album = &albumFull{albumID3: al.id3(), Song: trackChildren(ts)}
	return res, nil
}

func (a *API) getSong(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	t, err := a.oneTrack(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, fail(errNotFound, "Song not found")
	}
	ch := t.child()
	res := a.newResp()
	res.Song = &ch
	return res, nil
}

// getGenres conta faixas e álbuns por gênero (das tags das faixas).
func (a *API) getGenres(c *call) (*resp, error) {
	rows, err := a.d.Store.DB.QueryContext(c.ctx, `SELECT g.value, COUNT(*), COUNT(DISTINCT i.album_id)
FROM items i, json_each(i.genres) g WHERE i.kind = 'track' GROUP BY g.value`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &genres{}
	for rows.Next() {
		var g genre
		if err := rows.Scan(&g.Name, &g.SongCount, &g.AlbumCount); err != nil {
			return nil, err
		}
		if g.Name == "" {
			g.Name = "<Empty>"
		}
		out.Genre = append(out.Genre, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out.Genre, func(i, j int) bool {
		x, y := out.Genre[i], out.Genre[j]
		if x.SongCount != y.SongCount {
			return x.SongCount > y.SongCount
		}
		if x.AlbumCount != y.AlbumCount {
			return x.AlbumCount > y.AlbumCount
		}
		return x.Name < y.Name
	})
	res := a.newResp()
	res.Genres = out
	return res, nil
}

// artistInfo não tem fonte externa: biografia vem do overview, sem similares.
func (a *API) artistInfoBase(c *call) (*artistInfoBase, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	ar, err := a.oneArtist(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, fail(errNotFound, "Artist not found")
	}
	return &artistInfoBase{Biography: ar.Overview}, nil
}

func (a *API) artistInfo(c *call) (*resp, error) {
	b, err := a.artistInfoBase(c)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.ArtistInfo = &artistInfo{artistInfoBase: *b}
	return res, nil
}

func (a *API) artistInfo2(c *call) (*resp, error) {
	b, err := a.artistInfoBase(c)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.ArtistInfo2 = &artistInfo2{artistInfoBase: *b}
	return res, nil
}

func (a *API) albumInfo(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	al, err := a.oneAlbum(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, fail(errNotFound, "Album not found")
	}
	res := a.newResp()
	res.AlbumInfo = &albumInfo{Notes: al.Overview}
	return res, nil
}

// similar sorteia faixas do mesmo artista e dos mesmos gêneros.
func (a *API) similar(c *call) ([]child, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	count := c.p.int("count", 50)
	var artistID, self string
	var gs []string
	switch a.kindOf(c.ctx, id) {
	case "track":
		t, err := a.oneTrack(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		artistID, self, gs = t.ArtistID, t.ID, t.Genres
	case "album":
		al, err := a.oneAlbum(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		artistID, gs = al.ArtistID, al.Genres
	case "artist":
		artistID = cleanID(id)
		var raw string
		a.d.Store.DB.QueryRowContext(c.ctx, `SELECT COALESCE(json_group_array(DISTINCT g.value),'[]') FROM items i, json_each(i.genres) g
WHERE i.kind = 'track' AND i.artist_id = ?`, artistID).Scan(&raw)
		json.Unmarshal([]byte(raw), &gs)
	default:
		return nil, fail(errNotFound, "")
	}
	tail := `AND i.id <> ? AND (i.artist_id = ?`
	args := []any{self, artistID}
	for _, g := range gs {
		tail += ` OR EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE lower(g.value) = lower(?))`
		args = append(args, g)
	}
	ts, err := a.tracks(c.ctx, c.user.ID, tail+`) ORDER BY RANDOM() LIMIT ?`, append(args, max(count, 0))...)
	if err != nil {
		return nil, err
	}
	return trackChildren(ts), nil
}

func (a *API) similarSongs(c *call) (*resp, error) {
	s, err := a.similar(c)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.SimilarSongs = &songs{Songs: s}
	return res, nil
}

func (a *API) similarSongs2(c *call) (*resp, error) {
	s, err := a.similar(c)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.SimilarSongs2 = &songs{Songs: s}
	return res, nil
}

// topSongs: as mais tocadas do artista (por nome ou, OpenSubsonic, por id).
func (a *API) topSongs(c *call) (*resp, error) {
	name, id := c.p.str("artist"), c.p.str("id")
	if name == "" && id == "" {
		return nil, fail(errMissingParam, "missing parameter: 'artist'")
	}
	count := c.p.int("count", 50)
	if name == "" {
		name = "\x00" // não casa com faixa sem artista
	}
	tail := `AND (i.artist_id = ? OR lower(i.artist) = lower(?) OR lower(i.album_artist) = lower(?))
ORDER BY COALESCE(u.play_count,0) DESC, i.album_id, i.parent_idx, i.idx LIMIT ?`
	ts, err := a.tracks(c.ctx, c.user.ID, tail, cleanID(id), strings.TrimSpace(name), strings.TrimSpace(name), max(count, 0))
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.TopSongs = &songs{Songs: trackChildren(ts)}
	return res, nil
}

// shuffle embaralha no lugar.
func shuffle[T any](s []T) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}
