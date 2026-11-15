package jellyfin

import (
	"database/sql"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// tipos que a API do Jellyfin enxerga
const jfKinds = `('movie','series','season','episode','artist','album','track')`

type row struct {
	it *store.Item
	ud store.UserData
}

// query é um SELECT de itens com o estado do usuário junto.
type query struct {
	where []string
	args  []any
	order []string
	start int
	limit int // <= 0: sem limite
}

func (q *query) and(cond string, args ...any) {
	q.where = append(q.where, cond)
	q.args = append(q.args, args...)
}

// in monta "col IN (?,?,..)".
func in(col string, vals []string) (string, []any) {
	if len(vals) == 0 {
		return "0", nil
	}
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	return col + " IN (" + strings.TrimSuffix(strings.Repeat("?,", len(vals)), ",") + ")", args
}

func (q *query) in(col string, vals []string) {
	cond, args := in(col, vals)
	q.and(cond, args...)
}

const udCols = `, COALESCE(u.played,0), COALESCE(u.play_count,0), COALESCE(u.position,0), COALESCE(u.last_played,''),
COALESCE(u.favorite,0), COALESCE(u.favorite_at,''), COALESCE(u.rating,0)`

// playedCond: série/temporada está vista quando todo episódio está.
func playedCond(uid string) (string, []any) {
	return `(CASE WHEN i.kind IN ('series','season') THEN
  EXISTS (SELECT 1 FROM items e2 WHERE e2.kind = 'episode' AND (e2.series_id = i.id OR e2.season_id = i.id)) AND
  NOT EXISTS (SELECT 1 FROM items e LEFT JOIN userdata eu ON eu.item_id = e.id AND eu.user_id = ?
    WHERE e.kind = 'episode' AND (e.series_id = i.id OR e.season_id = i.id) AND COALESCE(eu.played,0) = 0)
ELSE COALESCE(u.played,0) = 1 END)`, []any{uid}
}

type extraScanner struct {
	rows  *sql.Rows
	extra []any
}

func (e extraScanner) Scan(dest ...any) error { return e.rows.Scan(append(dest, e.extra...)...) }

// run executa e devolve a página e o total.
func (c *rc) run(q query) ([]row, int, error) {
	from := ` FROM items i LEFT JOIN userdata u ON u.item_id = i.id AND u.user_id = ? WHERE i.kind IN ` + jfKinds
	for _, w := range q.where {
		from += " AND (" + w + ")"
	}
	args := append([]any{c.uid()}, q.args...)
	var total int
	if err := c.s.d.Store.DB.QueryRowContext(c.ctx(), `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := append(append([]string{}, q.order...), "i.sort_name", "i.id")
	limit := q.limit
	if limit <= 0 {
		limit = -1
	}
	rows, err := c.s.d.Store.DB.QueryContext(c.ctx(), `SELECT `+store.ItemColumns+udCols+from+
		` ORDER BY `+strings.Join(order, ", ")+` LIMIT ? OFFSET ?`, append(args, limit, max(q.start, 0))...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []row
	for rows.Next() {
		var r row
		var last, fav string
		it, err := store.ScanItem(extraScanner{rows, []any{&r.ud.Played, &r.ud.PlayCount, &r.ud.Position, &last,
			&r.ud.Favorite, &fav, &r.ud.Rating}})
		if err != nil {
			return nil, 0, err
		}
		r.it = it
		r.ud.LastPlayed, r.ud.FavoriteAt = parseTime(last), parseTime(fav)
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func (c *rc) dtos(rows []row) []itemDto {
	out := make([]itemDto, 0, len(rows))
	for _, r := range rows {
		out = append(out, c.dto(r.it, r.ud))
	}
	return out
}

// kinds lê IncludeItemTypes (+ MediaTypes, ExcludeItemTypes). nil = todos;
// ok=false = só tipos que não temos (BoxSet, Playlist...): resultado vazio.
func (c *rc) kinds() (kinds []string, ok bool) {
	types := c.ql("IncludeItemTypes", ",|")
	if len(types) > 0 {
		for _, t := range types {
			kinds = append(kinds, typeKinds[strings.ToLower(t)]...)
		}
		if len(kinds) == 0 {
			return nil, false
		}
	}
	if mts := c.ql("MediaTypes", ",|"); len(mts) > 0 {
		allowed := map[string]bool{}
		for _, m := range mts {
			switch strings.ToLower(m) {
			case "video":
				allowed[store.KindMovie], allowed[store.KindEpisode] = true, true
			case "audio":
				allowed[store.KindTrack] = true
			}
		}
		if kinds == nil {
			kinds = []string{store.KindMovie, store.KindSeries, store.KindSeason, store.KindEpisode, store.KindArtist, store.KindAlbum, store.KindTrack}
		}
		kinds = filter(kinds, func(k string) bool { return allowed[k] })
		if len(kinds) == 0 {
			return nil, false
		}
	}
	if ex := c.ql("ExcludeItemTypes", ",|"); len(ex) > 0 {
		drop := map[string]bool{}
		for _, t := range ex {
			for _, k := range typeKinds[strings.ToLower(t)] {
				drop[k] = true
			}
		}
		if kinds == nil {
			kinds = []string{store.KindMovie, store.KindSeries, store.KindSeason, store.KindEpisode, store.KindArtist, store.KindAlbum, store.KindTrack}
		}
		kinds = filter(kinds, func(k string) bool { return !drop[k] })
		if len(kinds) == 0 {
			return nil, false
		}
	}
	return kinds, true
}

func filter[T any](in []T, keep func(T) bool) []T {
	var out []T
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func normIDs(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = store.NormalizeID(id)
	}
	return out
}

// filters aplica os filtros comuns de /Items, /Artists, /Search...
func (c *rc) filters(q *query) {
	uid := c.uid()
	if t := strings.TrimSpace(c.qs("SearchTerm")); t != "" {
		like := "%" + t + "%"
		q.and("i.name LIKE ? OR i.sort_name LIKE ?", like, strings.ToLower(like))
	}
	if t := c.qs("NameStartsWith"); t != "" {
		q.and("i.sort_name LIKE ?", strings.ToLower(t)+"%")
	}
	if t := c.qs("NameStartsWithOrGreater"); t != "" {
		q.and("i.sort_name >= ?", strings.ToLower(t))
	}
	if t := c.qs("NameLessThan"); t != "" {
		q.and("i.sort_name < ?", strings.ToLower(t))
	}
	played, pargs := playedCond(uid)
	for _, f := range c.ql("Filters", ",|") {
		switch strings.ToLower(f) {
		case "isfavorite", "isfavoriteorlikes", "likes":
			q.and("COALESCE(u.favorite,0) = 1")
		case "isplayed":
			q.and(played, pargs...)
		case "isunplayed":
			q.and("NOT "+played, pargs...)
		case "isresumable":
			q.and("COALESCE(u.position,0) > 0")
		case "isfolder":
			q.and("i.kind IN ('series','season','artist','album')")
		case "isnotfolder":
			q.and("i.kind NOT IN ('series','season','artist','album')")
		}
	}
	if v, ok := c.qb("IsFavorite"); ok {
		q.and("COALESCE(u.favorite,0) = ?", v)
	}
	if v, ok := c.qb("IsPlayed"); ok {
		if v {
			q.and(played, pargs...)
		} else {
			q.and("NOT "+played, pargs...)
		}
	}
	if v, ok := c.qb("IsFolder"); ok {
		if v {
			q.and("i.kind IN ('series','season','artist','album')")
		} else {
			q.and("i.kind NOT IN ('series','season','artist','album')")
		}
	}
	genres := c.ql("Genres", "|")
	for _, id := range c.ql("GenreIds", ",|") {
		if g, ok := c.genreByID(id); ok {
			genres = append(genres, g)
		}
	}
	if len(genres) > 0 || len(c.ql("GenreIds", ",|")) > 0 {
		cond, args := in("g.value COLLATE NOCASE", genres)
		q.and("EXISTS (SELECT 1 FROM json_each(i.genres) g WHERE "+cond+")", args...)
	}
	if years := c.ql("Years", ",|"); len(years) > 0 {
		q.in("i.year", years)
	}
	artists := normIDs(append(c.ql("ArtistIds", ",|"), c.ql("AlbumArtistIds", ",|")...))
	if len(artists) > 0 {
		a1, args1 := in("i.artist_id", artists)
		a2, args2 := in("artist_id", artists)
		q.and(a1+" OR i.album_id IN (SELECT id FROM items WHERE kind = 'album' AND "+a2+")", append(args1, args2...)...)
	}
	if ids := normIDs(c.ql("AlbumIds", ",|")); len(ids) > 0 {
		q.in("i.album_id", ids)
	}
	if names := c.ql("Artists", "|"); len(names) > 0 {
		a1, args1 := in("i.artist COLLATE NOCASE", names)
		a2, args2 := in("i.album_artist COLLATE NOCASE", names)
		q.and(a1+" OR "+a2, append(args1, args2...)...)
	}
	if names := c.ql("Albums", "|"); len(names) > 0 {
		q.in("i.album COLLATE NOCASE", names)
	}
	if ex := normIDs(c.ql("ExcludeItemIds", ",|")); len(ex) > 0 {
		cond, args := in("i.id", ex)
		q.and("NOT "+cond, args...)
	}
	if n := c.qs("ParentIndexNumber"); n != "" {
		q.and("i.parent_idx = ?", c.qi("ParentIndexNumber", 0))
	}
	if n := c.qs("IndexNumber"); n != "" {
		q.and("i.idx = ?", c.qi("IndexNumber", 0))
	}
}

var sortCols = map[string]string{
	"sortname":             "i.sort_name",
	"name":                 "i.name COLLATE NOCASE",
	"datecreated":          "i.added_at",
	"datelastcontentadded": "i.added_at",
	"premieredate":         "(CASE WHEN i.premiere != '' THEN i.premiere ELSE printf('%04d', i.year) END)",
	"productionyear":       "i.year",
	"random":               "RANDOM()",
	"dateplayed":           "u.last_played",
	"indexnumber":          "i.idx",
	"parentindexnumber":    "i.parent_idx",
	"album":                "i.album COLLATE NOCASE",
	"albumartist":          "i.album_artist COLLATE NOCASE",
	"artist":               "i.artist COLLATE NOCASE",
	"runtime":              "i.duration",
	"playcount":            "COALESCE(u.play_count,0)",
	"isfavoriteorliked":    "COALESCE(u.favorite,0)",
	"airedepisodeorder":    "i.parent_idx, i.idx",
	"seriessortname":       "i.sort_name",
	"filename":             "i.rel_path",
}

// sortOrder lê SortBy/SortOrder; def quando não pedirem nada.
func (c *rc) sortOrder(def ...string) []string {
	by := c.ql("SortBy", ",|")
	if len(by) == 0 {
		return def
	}
	orders := c.ql("SortOrder", ",|")
	var out []string
	for i, b := range by {
		col, ok := sortCols[strings.ToLower(b)]
		if !ok {
			continue
		}
		dir := "ASC"
		o := ""
		if i < len(orders) {
			o = orders[i]
		} else if len(orders) > 0 {
			o = orders[len(orders)-1]
		}
		if strings.EqualFold(o, "Descending") {
			dir = "DESC"
		}
		for p := range strings.SplitSeq(col, ", ") {
			out = append(out, p+" "+dir)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}

// genreByID acha o nome de um gênero pelo id.
func (c *rc) genreByID(id string) (string, bool) {
	id = store.NormalizeID(id)
	rows, err := c.s.d.Store.DB.QueryContext(c.ctx(), `SELECT DISTINCT g.value FROM items i, json_each(i.genres) g WHERE i.kind IN `+jfKinds)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	for rows.Next() {
		var g string
		if rows.Scan(&g) == nil && genreID(g) == id {
			return g, true
		}
	}
	return "", false
}
