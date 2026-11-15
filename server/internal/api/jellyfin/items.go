package jellyfin

import (
	"net/http"
	"slices"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// parentCond: filhos diretos ou (recursive) tudo abaixo de um item.
func parentCond(q *query, parent string, recursive bool) {
	if recursive {
		q.and("i.parent_id = ? OR i.series_id = ? OR i.season_id = ? OR i.artist_id = ? OR i.album_id = ?",
			parent, parent, parent, parent, parent)
		return
	}
	q.and("i.parent_id = ?", parent)
}

// itemsQuery traduz a query do /Items. mode: "views" (raiz), "empty".
func (c *rc) itemsQuery() (query, string) {
	q := query{start: max(c.qi("StartIndex", 0), 0), limit: c.qi("Limit", 0)}
	recursive, _ := c.qb("Recursive")
	kinds, ok := c.kinds()
	if !ok {
		return q, "empty"
	}
	parent := store.NormalizeID(c.qs("ParentId"))
	ids := normIDs(c.ql("Ids", ",|"))
	search := c.qs("SearchTerm") != ""
	switch {
	case len(ids) > 0:
		q.in("i.id", ids)
	case parent == rootID || (parent == "" && !recursive && kinds == nil && !search):
		if !recursive && kinds == nil {
			return q, "views"
		}
	case parent != "":
		if v, ok := c.s.viewByID(parent); ok {
			q.and("i.library = ?", v.Lib)
			if !recursive && kinds == nil {
				q.and("i.kind = ?", topKind(v.Lib))
			}
		} else {
			parentCond(&q, parent, recursive)
		}
	}
	if kinds != nil {
		q.in("i.kind", kinds)
	}
	c.filters(&q)
	q.order = c.sortOrder("i.parent_idx", "i.idx")
	return q, ""
}

// GET /Items e /Users/{userId}/Items
func (s *Server) getItems(c *rc) {
	if s.playlistQuery(c) {
		return
	}
	q, mode := c.itemsQuery()
	switch mode {
	case "views":
		s.userViews(c)
		return
	case "empty":
		c.json(result(nil, 0, q.start))
		return
	}
	if ids := normIDs(c.ql("Ids", ",|")); len(ids) > 0 {
		// Ids pode trazer bibliotecas
		var views []itemDto
		for _, id := range ids {
			if v, ok := s.viewByID(id); ok {
				views = append(views, c.viewDto(v))
			} else if p, ok := c.userList(id); ok {
				views = append(views, c.playlistDto(p))
			}
		}
		if len(views) > 0 {
			c.json(result(views, len(views), 0))
			return
		}
	}
	rows, total, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	c.json(result(c.dtos(rows), total, q.start))
}

// GET /Items/{id} e /Users/{userId}/Items/{id}
func (s *Server) getItem(c *rc) {
	id := c.pid("id")
	if id == rootID {
		c.json(c.rootDto())
		return
	}
	if id == playlistsViewID {
		ps, err := c.playlists()
		if err != nil {
			c.err(err)
			return
		}
		c.json(c.playlistsViewDto(len(ps)))
		return
	}
	if p, ok := c.userList(id); ok {
		c.json(c.playlistDto(p))
		return
	}
	d, err := c.itemDtoByID(id)
	if err != nil {
		c.err(err)
		return
	}
	c.json(d)
}

// GET /Users/{userId}/Items/Root
func (s *Server) rootItem(c *rc) { c.json(c.rootDto()) }

// GET /Items/Resume: o que parou no meio, mais recente primeiro.
func (s *Server) resume(c *rc) {
	q := query{start: max(c.qi("StartIndex", 0), 0), limit: c.qi("Limit", 0)}
	kinds, ok := c.kinds()
	if !ok {
		c.json(result(nil, 0, q.start))
		return
	}
	if kinds == nil {
		kinds = []string{store.KindMovie, store.KindEpisode, store.KindTrack}
	}
	q.in("i.kind", kinds)
	q.and("COALESCE(u.position,0) > 0 AND COALESCE(u.played,0) = 0")
	if parent := store.NormalizeID(c.qs("ParentId")); parent != "" {
		if v, ok := c.s.viewByID(parent); ok {
			q.and("i.library = ?", v.Lib)
		} else {
			parentCond(&q, parent, true)
		}
	}
	if t := c.qs("SearchTerm"); t != "" {
		q.and("fold(i.name) LIKE ?", "%"+store.Fold(t)+"%")
	}
	q.order = []string{"u.last_played DESC"}
	rows, total, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	c.json(result(c.dtos(rows), total, q.start))
}

// GET /Items/Latest: o que chegou por último (episódios agrupados na
// série, faixas no álbum). Devolve uma lista pura.
func (s *Server) latest(c *rc) {
	q := query{limit: c.qi("Limit", 20)}
	types := c.ql("IncludeItemTypes", ",|")
	group := map[string]string{"movie": store.KindMovie, "series": store.KindSeries, "season": store.KindSeries,
		"episode": store.KindSeries, "musicalbum": store.KindAlbum, "audio": store.KindAlbum}
	var kinds []string
	for _, t := range types {
		if k, ok := group[strings.ToLower(t)]; ok && !slices.Contains(kinds, k) {
			kinds = append(kinds, k)
		}
	}
	if len(types) > 0 && len(kinds) == 0 {
		c.json([]itemDto{})
		return
	}
	parent := store.NormalizeID(c.qs("ParentId"))
	if v, ok := c.s.viewByID(parent); ok {
		q.and("i.library = ?", v.Lib)
		if kinds == nil {
			kinds = []string{map[string]string{"movies": store.KindMovie, "shows": store.KindSeries, "music": store.KindAlbum}[v.Lib]}
		}
	} else if parent != "" {
		parentCond(&q, parent, true)
	}
	if kinds == nil {
		kinds = []string{store.KindMovie, store.KindSeries, store.KindAlbum}
	}
	q.in("i.kind", kinds)
	if v, ok := c.qb("IsPlayed"); ok {
		played, args := playedCond(c.uid())
		if v {
			q.and(played, args...)
		} else {
			q.and("NOT "+played, args...)
		}
	}
	q.and(`i.kind != 'series' OR EXISTS (SELECT 1 FROM items e WHERE e.kind = 'episode' AND e.series_id = i.id)`)
	q.order = []string{`COALESCE(CASE i.kind
  WHEN 'series' THEN (SELECT MAX(e.added_at) FROM items e WHERE e.kind = 'episode' AND e.series_id = i.id)
  WHEN 'album' THEN (SELECT MAX(t.added_at) FROM items t WHERE t.kind = 'track' AND t.album_id = i.id)
END, i.added_at) DESC`}
	rows, _, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	c.json(c.dtos(rows))
}

// GET /Shows/NextUp: o episódio seguinte ao último visto de cada série.
func (s *Server) nextUp(c *rc) {
	start, limit := max(c.qi("StartIndex", 0), 0), c.qi("Limit", 0)
	only := store.NormalizeID(c.qs("SeriesId"))
	rows, err := s.d.Store.DB.QueryContext(c.ctx(), `SELECT e.series_id, MAX(u.last_played) m FROM items e
JOIN userdata u ON u.item_id = e.id AND u.user_id = ? WHERE e.kind = 'episode' AND u.played = 1
GROUP BY e.series_id ORDER BY m DESC`, c.uid())
	if err != nil {
		c.err(err)
		return
	}
	var series []string
	for rows.Next() {
		var id string
		var m any
		if rows.Scan(&id, &m) == nil && (only == "" || id == only) {
			series = append(series, id)
		}
	}
	rows.Close()
	var next []row
	for _, sid := range series {
		q := query{order: []string{"i.parent_idx", "i.idx"}}
		q.and("i.kind = 'episode' AND i.series_id = ?", sid)
		eps, _, err := c.run(q)
		if err != nil {
			c.err(err)
			return
		}
		last := -1
		for i, e := range eps {
			if e.ud.Played {
				last = i
			}
		}
		for _, e := range eps[last+1:] {
			if !e.ud.Played {
				next = append(next, e)
				break
			}
		}
	}
	total := len(next)
	next = next[min(start, total):]
	if limit > 0 && limit < len(next) {
		next = next[:limit]
	}
	c.json(result(c.dtos(next), total, start))
}

// GET /Shows/{id}/Seasons
func (s *Server) seasons(c *rc) {
	q := query{order: []string{"i.idx"}}
	q.and("i.kind = 'season' AND i.series_id = ?", c.pid("id"))
	c.filters(&q)
	rows, total, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	c.json(result(c.dtos(rows), total, 0))
}

// GET /Shows/{id}/Episodes (SeasonId, Season, StartItemId, paginação)
func (s *Server) episodes(c *rc) {
	q := query{}
	q.and("i.kind = 'episode' AND i.series_id = ?", c.pid("id"))
	if sid := store.NormalizeID(c.qs("SeasonId")); sid != "" {
		q.and("i.season_id = ?", sid)
	} else if c.qs("Season") != "" {
		q.and("i.parent_idx = ?", c.qi("Season", 0))
	}
	c.filters(&q)
	q.order = c.sortOrder("i.parent_idx", "i.idx")
	rows, _, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	if startID := store.NormalizeID(c.qs("StartItemId")); startID != "" {
		for i, r := range rows {
			if r.it.ID == startID {
				rows = rows[i:]
				break
			}
		}
	}
	total, start, limit := len(rows), max(c.qi("StartIndex", 0), 0), c.qi("Limit", 0)
	rows = rows[min(start, total):]
	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}
	c.json(result(c.dtos(rows), total, start))
}

// GET /Items/{id}/Similar: mesmo tipo, algum gênero em comum.
func (s *Server) similar(c *rc) {
	it := c.lookup(c.pid("id"))
	if it == nil {
		c.notFound()
		return
	}
	if it.Kind == store.KindEpisode || it.Kind == store.KindSeason {
		if sr := c.lookup(it.SeriesID); sr != nil {
			it = sr
		}
	}
	q := query{limit: c.qi("Limit", 12), order: []string{"RANDOM()"}}
	q.and("i.kind = ? AND i.id != ?", it.Kind, it.ID)
	if len(it.Genres) > 0 {
		cond, args := in("g.value COLLATE NOCASE", it.Genres)
		q.and("EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE "+cond+")", args...)
	}
	rows, total, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	c.json(result(c.dtos(rows), total, 0))
}

// GET /Items/{id}/Ancestors: pais até a biblioteca.
func (s *Server) ancestors(c *rc) {
	it := c.lookup(c.pid("id"))
	if it == nil {
		c.notFound()
		return
	}
	out := []itemDto{}
	for p := c.lookup(it.ParentID); p != nil; p = c.lookup(p.ParentID) {
		ud, _ := s.d.Store.UserData(c.ctx(), c.uid(), p.ID)
		out = append(out, c.dto(p, ud))
		it = p
	}
	if v, ok := s.viewByID(viewID(it.Library)); ok {
		out = append(out, c.viewDto(v))
	}
	c.json(out)
}

// GET /Items/{id}/InstantMix (e Albums/Artists/Songs): faixas parecidas.
func (s *Server) instantMix(c *rc) {
	id := c.pid("id")
	if id == "" {
		id = store.NormalizeID(c.qs("Id"))
	}
	q := query{limit: c.qi("Limit", 200), order: []string{"RANDOM()"}}
	q.and("i.kind = 'track'")
	var conds []string
	var args []any
	if it := c.lookup(id); it != nil {
		artist := it.ArtistID
		if it.Kind == store.KindArtist {
			artist = it.ID
		}
		if artist != "" {
			conds = append(conds, "i.artist_id = ? OR i.album_id IN (SELECT id FROM items WHERE kind = 'album' AND artist_id = ?)")
			args = append(args, artist, artist)
		}
		if len(it.Genres) > 0 {
			cond, a := in("g.value COLLATE NOCASE", it.Genres)
			conds = append(conds, "EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE "+cond+")")
			args = append(args, a...)
		}
	} else if g, ok := c.genreByID(id); ok {
		conds = append(conds, "EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE g.value = ? COLLATE NOCASE)")
		args = append(args, g)
	}
	if len(conds) > 0 {
		q.and(strings.Join(conds, " OR "), args...)
	}
	rows, total, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	c.json(result(c.dtos(rows), total, 0))
}

// genres agrega os gêneros dos itens (music = gêneros musicais).
func (c *rc) genreList(music bool) ([]string, error) {
	q := query{}
	kinds, ok := c.kinds()
	if !ok {
		return nil, nil
	}
	switch {
	case kinds != nil:
		q.in("i.kind", kinds)
	case music:
		q.and("i.kind IN ('album','track')")
	default:
		q.and("i.kind IN ('movie','series')")
	}
	if parent := store.NormalizeID(c.qs("ParentId")); parent != "" {
		if v, ok := c.s.viewByID(parent); ok {
			q.and("i.library = ?", v.Lib)
		} else {
			parentCond(&q, parent, true)
		}
	}
	if t := c.qs("SearchTerm"); t != "" {
		q.and("fold(g.value) LIKE ?", "%"+store.Fold(t)+"%")
	}
	if t := c.qs("NameStartsWith"); t != "" {
		q.and("g.value LIKE ?", t+"%")
	}
	sqlq := `SELECT g.value FROM items i LEFT JOIN userdata u ON u.item_id = i.id AND u.user_id = ?
JOIN json_each(i.genres) g WHERE i.kind IN ` + jfKinds
	for _, w := range q.where {
		sqlq += " AND (" + w + ")"
	}
	rows, err := c.s.d.Store.DB.QueryContext(c.ctx(), sqlq+` GROUP BY g.value COLLATE NOCASE ORDER BY g.value COLLATE NOCASE`,
		append([]any{c.uid()}, q.args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (c *rc) genreDto(name string, music bool) itemDto {
	d := newItemDto(c.s.serverID)
	d.Name, d.Id, d.SortName, d.IsFolder = name, genreID(name), strings.ToLower(name), true
	d.Type = "Genre"
	if music {
		d.Type = "MusicGenre"
	}
	d.LocationType = "FileSystem"
	d.UserData = &userDataDto{Key: d.Id, ItemId: d.Id}
	return d
}

func (s *Server) genresHandler(music bool) func(*rc) {
	return func(c *rc) {
		names, err := c.genreList(music)
		if err != nil {
			c.err(err)
			return
		}
		out := []itemDto{}
		for _, g := range names {
			out = append(out, c.genreDto(g, music))
		}
		total, start, limit := len(out), max(c.qi("StartIndex", 0), 0), c.qi("Limit", 0)
		out = out[min(start, total):]
		if limit > 0 && limit < len(out) {
			out = out[:limit]
		}
		c.json(result(out, total, start))
	}
}

// GET /Genres/{name}
func (s *Server) genreHandler(music bool) func(*rc) {
	return func(c *rc) {
		name := c.p["name"]
		if g, ok := c.genreByID(name); ok {
			name = g
		}
		c.json(c.genreDto(name, music))
	}
}

// GET /Items/Filters
func (s *Server) itemFilters(c *rc) {
	genres, err := c.genreList(false)
	if err != nil {
		c.err(err)
		return
	}
	if genres == nil {
		genres = []string{}
	}
	var years []int
	q, _ := c.itemsQuery()
	q.start, q.limit = 0, 0
	rows, _, err := c.run(q)
	if err == nil {
		for _, r := range rows {
			if r.it.Year > 0 && !slices.Contains(years, r.it.Year) {
				years = append(years, r.it.Year)
			}
		}
	}
	slices.Sort(years)
	if years == nil {
		years = []int{}
	}
	c.json(map[string]any{"Genres": genres, "Tags": []string{}, "OfficialRatings": []string{}, "Years": years})
}

// GET /Items/Filters2
func (s *Server) itemFilters2(c *rc) {
	genres, err := c.genreList(false)
	if err != nil {
		c.err(err)
		return
	}
	out := []nameID{}
	for _, g := range genres {
		out = append(out, nameID{Name: g, Id: genreID(g)})
	}
	c.json(map[string]any{"Genres": out, "Tags": []string{}})
}

// GET /Artists e /Artists/AlbumArtists
func (s *Server) artists(albumArtists bool) func(*rc) {
	return func(c *rc) {
		q := query{start: max(c.qi("StartIndex", 0), 0), limit: c.qi("Limit", 0)}
		q.and("i.kind = 'artist'")
		if albumArtists {
			q.and("EXISTS (SELECT 1 FROM items a WHERE a.kind = 'album' AND a.artist_id = i.id)")
		}
		c.filters(&q)
		q.order = c.sortOrder()
		rows, total, err := c.run(q)
		if err != nil {
			c.err(err)
			return
		}
		c.json(result(c.dtos(rows), total, q.start))
	}
}

func (s *Server) artistByName(c *rc, name string) (*store.Item, error) {
	if it := c.lookup(store.NormalizeID(name)); it != nil && it.Kind == store.KindArtist {
		return it, nil
	}
	list, err := s.d.Store.Items(c.ctx(), `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = 'artist' AND i.name = ? COLLATE NOCASE LIMIT 1`, name)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, store.ErrNotFound
	}
	return list[0], nil
}

// GET /Artists/{name}
func (s *Server) artist(c *rc) {
	it, err := s.artistByName(c, c.p["name"])
	if err != nil {
		c.err(err)
		return
	}
	ud, _ := s.d.Store.UserData(c.ctx(), c.uid(), it.ID)
	c.json(c.dto(it, ud))
}

// GET /Items/Counts
func (s *Server) counts(c *rc) {
	n := map[string]int{}
	q := query{}
	c.filters(&q)
	from := ` FROM items i LEFT JOIN userdata u ON u.item_id = i.id AND u.user_id = ? WHERE i.kind IN ` + jfKinds
	for _, w := range q.where {
		from += " AND (" + w + ")"
	}
	rows, err := s.d.Store.DB.QueryContext(c.ctx(), `SELECT i.kind, COUNT(*)`+from+` GROUP BY i.kind`, append([]any{c.uid()}, q.args...)...)
	if err != nil {
		c.err(err)
		return
	}
	defer rows.Close()
	total := 0
	for rows.Next() {
		var k string
		var v int
		rows.Scan(&k, &v)
		n[k] = v
		total += v
	}
	c.json(map[string]int{
		"MovieCount": n[store.KindMovie], "SeriesCount": n[store.KindSeries], "EpisodeCount": n[store.KindEpisode],
		"ArtistCount": n[store.KindArtist], "ProgramCount": 0, "TrailerCount": 0, "SongCount": n[store.KindTrack],
		"AlbumCount": n[store.KindAlbum], "MusicVideoCount": 0, "BoxSetCount": 0, "BookCount": 0, "ItemCount": total,
	})
}

// GET /Search/Hints
func (s *Server) searchHints(c *rc) {
	q := query{start: max(c.qi("StartIndex", 0), 0), limit: c.qi("Limit", 0)}
	kinds, ok := c.kinds()
	if !ok || strings.TrimSpace(c.qs("SearchTerm")) == "" {
		c.json(map[string]any{"SearchHints": []any{}, "TotalRecordCount": 0})
		return
	}
	if kinds != nil {
		q.in("i.kind", kinds)
	}
	if parent := store.NormalizeID(c.qs("ParentId")); parent != "" {
		if v, ok := s.viewByID(parent); ok {
			q.and("i.library = ?", v.Lib)
		}
	}
	c.filters(&q)
	q.order = []string{"i.kind = 'track'", "i.kind = 'episode'"}
	rows, total, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	hints := []map[string]any{}
	for _, r := range rows {
		d := c.dto(r.it, r.ud)
		h := map[string]any{
			"ItemId": d.Id, "Id": d.Id, "Name": d.Name, "Type": d.Type, "MediaType": d.MediaType,
			"IsFolder": d.IsFolder, "RunTimeTicks": d.RunTimeTicks, "Artists": nonNil(d.Artists),
		}
		if d.ProductionYear > 0 {
			h["ProductionYear"] = d.ProductionYear
		}
		if t := d.ImageTags["Primary"]; t != "" {
			h["PrimaryImageTag"] = t
		}
		if d.IndexNumber != nil {
			h["IndexNumber"] = *d.IndexNumber
		}
		if d.ParentIndexNumber != nil {
			h["ParentIndexNumber"] = *d.ParentIndexNumber
		}
		for k, v := range map[string]string{"Series": d.SeriesName, "SeriesId": d.SeriesId, "Album": d.Album,
			"AlbumId": d.AlbumId, "AlbumArtist": d.AlbumArtist, "PremiereDate": d.PremiereDate} {
			if v != "" {
				h[k] = v
			}
		}
		hints = append(hints, h)
	}
	c.json(map[string]any{"SearchHints": hints, "TotalRecordCount": total})
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// emptyResult responde um QueryResult vazio (trailers, pessoas, estúdios...).
func emptyResult(c *rc) { c.json(result(nil, 0, 0)) }

// emptyList responde [].
func emptyList(c *rc) { c.json([]any{}) }

func (s *Server) themeMedia(c *rc) {
	id := c.pid("id")
	empty := func() map[string]any {
		return map[string]any{"Items": []any{}, "TotalRecordCount": 0, "StartIndex": 0, "OwnerId": id}
	}
	c.json(map[string]any{"ThemeVideosResult": empty(), "ThemeSongsResult": empty(), "SoundtrackSongsResult": empty()})
}

func methodNotAllowed(c *rc) { c.fail(http.StatusMethodNotAllowed, "não suportado") }
