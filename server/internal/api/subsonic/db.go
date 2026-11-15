package subsonic

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Tabelas só do Subsonic: playlists, fila de reprodução e marcadores.
func init() {
	store.Register(store.Migration{Name: "subsonic/001-playlists", SQL: `
CREATE TABLE subsonic_playlists (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  comment    TEXT NOT NULL DEFAULT '',
  public     INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  changed_at TEXT NOT NULL
);
CREATE INDEX subsonic_playlists_user ON subsonic_playlists(user_id);
CREATE TABLE subsonic_playlist_items (
  playlist_id TEXT NOT NULL REFERENCES subsonic_playlists(id) ON DELETE CASCADE,
  pos         INTEGER NOT NULL,
  item_id     TEXT NOT NULL, -- sem FK: sobrevive a rescan
  PRIMARY KEY (playlist_id, pos)
);`}, store.Migration{Name: "subsonic/002-playqueue", SQL: `
CREATE TABLE subsonic_playqueue (
  user_id    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  items      TEXT NOT NULL DEFAULT '[]', -- JSON com os ids, na ordem
  current    INTEGER NOT NULL DEFAULT 0, -- índice em items
  position   INTEGER NOT NULL DEFAULT 0, -- ms
  changed_at TEXT NOT NULL,
  changed_by TEXT NOT NULL DEFAULT ''
);`}, store.Migration{Name: "subsonic/003-bookmarks", SQL: `
CREATE TABLE subsonic_bookmarks (
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id    TEXT NOT NULL,
  position   INTEGER NOT NULL DEFAULT 0, -- ms (também vai pro userdata.position)
  comment    TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  changed_at TEXT NOT NULL,
  PRIMARY KEY (user_id, item_id)
);`})
}

// extra acrescenta colunas depois das de ItemColumns no Scan.
type extra struct {
	r    interface{ Scan(...any) error }
	more []any
}

func (e extra) Scan(dest ...any) error { return e.r.Scan(append(dest, e.more...)...) }

// ud é o estado do usuário que vem junto nas consultas.
type ud struct {
	plays    int
	last     time.Time
	fav      bool
	favAt    time.Time
	rating   int
	bookmark int64 // ms
}

func scanUD(last, favAt string) (time.Time, time.Time) {
	return parseT(last), parseT(favAt)
}

func parseT(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// track é uma faixa com o que a resposta precisa além do item.
type track struct {
	*store.Item
	ud
	albumImage    string
	albumArtistID string
}

// album é um álbum com totais das faixas e o estado do usuário.
type album struct {
	*store.Item
	ud
	songs int
	dur   float64
	cover string // imagem do álbum ou da primeira faixa com capa
}

// artistRow é um artista com contagem de álbuns e capa emprestada.
type artistRow struct {
	*store.Item
	ud
	albums     int
	coverAlbum string
}

const trackSelect = `SELECT ` + store.ItemColumns + `,
  COALESCE(u.play_count,0), COALESCE(u.last_played,''), COALESCE(u.favorite,0), COALESCE(u.favorite_at,''), COALESCE(u.rating,0),
  COALESCE(b.position,0), COALESCE(al.image,''), COALESCE(al.artist_id,'')
FROM items i
LEFT JOIN userdata u ON u.item_id = i.id AND u.user_id = ?1
LEFT JOIN subsonic_bookmarks b ON b.item_id = i.id AND b.user_id = ?1
LEFT JOIN items al ON al.id = i.album_id
WHERE i.kind = 'track' `

const albumSelect = `SELECT ` + store.ItemColumns + `,
  COALESCE(s.n,0), COALESCE(s.dur,0), COALESCE(s.plays,0), COALESCE(s.last,''),
  COALESCE(u.favorite,0), COALESCE(u.favorite_at,''), COALESCE(u.rating,0),
  CASE WHEN i.image <> '' THEN i.image ELSE COALESCE((SELECT t.image FROM items t WHERE t.album_id = i.id AND t.kind = 'track' AND t.image <> '' LIMIT 1),'') END
FROM items i
LEFT JOIN (SELECT t.album_id aid, COUNT(*) n, SUM(t.duration) dur, SUM(COALESCE(ut.play_count,0)) plays, MAX(ut.last_played) last
  FROM items t LEFT JOIN userdata ut ON ut.item_id = t.id AND ut.user_id = ?1
  WHERE t.kind = 'track' GROUP BY t.album_id) s ON s.aid = i.id
LEFT JOIN userdata u ON u.item_id = i.id AND u.user_id = ?1
WHERE i.kind = 'album' `

const artistSelect = `SELECT ` + store.ItemColumns + `,
  (SELECT COUNT(*) FROM items a WHERE a.kind = 'album' AND a.artist_id = i.id),
  COALESCE((SELECT a.id FROM items a WHERE a.kind = 'album' AND a.artist_id = i.id AND (a.image <> '' OR EXISTS
    (SELECT 1 FROM items t WHERE t.album_id = a.id AND t.image <> '')) ORDER BY a.year, a.sort_name LIMIT 1),''),
  COALESCE(u.favorite,0), COALESCE(u.favorite_at,''), COALESCE(u.rating,0)
FROM items i
LEFT JOIN userdata u ON u.item_id = i.id AND u.user_id = ?1
WHERE i.kind = 'artist' `

// tracks roda trackSelect + where/order (args depois do usuário).
func (a *API) tracks(ctx context.Context, uid, tail string, args ...any) ([]*track, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, trackSelect+tail, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*track
	for rows.Next() {
		t := &track{}
		var last, favAt string
		it, err := store.ScanItem(extra{rows, []any{&t.plays, &last, &t.fav, &favAt, &t.rating, &t.bookmark, &t.albumImage, &t.albumArtistID}})
		if err != nil {
			return nil, err
		}
		t.Item = it
		t.last, t.favAt = scanUD(last, favAt)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (a *API) albums(ctx context.Context, uid, tail string, args ...any) ([]*album, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, albumSelect+tail, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*album
	for rows.Next() {
		al := &album{}
		var last, favAt string
		it, err := store.ScanItem(extra{rows, []any{&al.songs, &al.dur, &al.plays, &last, &al.fav, &favAt, &al.rating, &al.cover}})
		if err != nil {
			return nil, err
		}
		al.Item = it
		al.last, al.favAt = scanUD(last, favAt)
		out = append(out, al)
	}
	return out, rows.Err()
}

func (a *API) artists(ctx context.Context, uid, tail string, args ...any) ([]*artistRow, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, artistSelect+tail, append([]any{uid}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*artistRow
	for rows.Next() {
		ar := &artistRow{}
		var favAt string
		it, err := store.ScanItem(extra{rows, []any{&ar.albums, &ar.coverAlbum, &ar.fav, &favAt, &ar.rating}})
		if err != nil {
			return nil, err
		}
		ar.Item = it
		ar.favAt = parseT(favAt)
		out = append(out, ar)
	}
	return out, rows.Err()
}

// inList monta "(?,?,?)" pra n valores.
func inList(n int) string {
	return "(" + strings.TrimSuffix(strings.Repeat("?,", n), ",") + ")"
}

func anys(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// tracksByIDs devolve as faixas na ordem dos ids (repetidos ficam, sumidos saem).
func (a *API) tracksByIDs(ctx context.Context, uid string, ids []string) ([]*track, error) {
	uniq := map[string]bool{}
	var keys []string
	for _, id := range ids {
		id = cleanID(id)
		if !uniq[id] {
			uniq[id] = true
			keys = append(keys, id)
		}
	}
	byID := map[string]*track{}
	for len(keys) > 0 {
		n := min(len(keys), 500)
		ts, err := a.tracks(ctx, uid, `AND i.id IN `+inList(n), anys(keys[:n])...)
		if err != nil {
			return nil, err
		}
		for _, t := range ts {
			byID[t.ID] = t
		}
		keys = keys[n:]
	}
	var out []*track
	for _, id := range ids {
		if t := byID[cleanID(id)]; t != nil {
			out = append(out, t)
		}
	}
	return out, nil
}

func (a *API) albumsByIDs(ctx context.Context, uid string, ids []string) ([]*album, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	all, err := a.albums(ctx, uid, `AND i.id IN `+inList(len(ids)), anys(ids)...)
	if err != nil {
		return nil, err
	}
	return reorder(all, ids, func(x *album) string { return x.ID }), nil
}

func (a *API) artistsByIDs(ctx context.Context, uid string, ids []string) ([]*artistRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	all, err := a.artists(ctx, uid, `AND i.id IN `+inList(len(ids)), anys(ids)...)
	if err != nil {
		return nil, err
	}
	return reorder(all, ids, func(x *artistRow) string { return x.ID }), nil
}

func reorder[T any](items []T, ids []string, key func(T) string) []T {
	by := map[string]T{}
	for _, it := range items {
		by[key(it)] = it
	}
	out := make([]T, 0, len(ids))
	for _, id := range ids {
		if it, ok := by[id]; ok {
			out = append(out, it)
		}
	}
	return out
}

func (a *API) oneTrack(ctx context.Context, uid, id string) (*track, error) {
	ts, err := a.tracks(ctx, uid, `AND i.id = ?`, cleanID(id))
	if err != nil {
		return nil, err
	}
	if len(ts) == 0 {
		return nil, store.ErrNotFound
	}
	return ts[0], nil
}

func (a *API) oneAlbum(ctx context.Context, uid, id string) (*album, error) {
	as, err := a.albums(ctx, uid, `AND i.id = ?`, cleanID(id))
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, store.ErrNotFound
	}
	return as[0], nil
}

func (a *API) oneArtist(ctx context.Context, uid, id string) (*artistRow, error) {
	as, err := a.artists(ctx, uid, `AND i.id = ?`, cleanID(id))
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, store.ErrNotFound
	}
	return as[0], nil
}

// kindOf diz o tipo de um id (vazio se não existe).
func (a *API) kindOf(ctx context.Context, id string) string {
	var k string
	err := a.d.Store.DB.QueryRowContext(ctx, `SELECT kind FROM items WHERE id = ?`, cleanID(id)).Scan(&k)
	if err == sql.ErrNoRows {
		var n int
		a.d.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM subsonic_playlists WHERE id = ?`, cleanID(id)).Scan(&n)
		if n > 0 {
			return "playlist"
		}
	}
	return k
}

// cleanID aceita os prefixos do Navidrome (al-, ar-, mf-, pl-) e o sufixo
// "_<hash>" das capas.
func cleanID(id string) string {
	id = strings.TrimSpace(id)
	for _, p := range []string{"al-", "ar-", "mf-", "pl-", "tr-"} {
		if rest, ok := strings.CutPrefix(id, p); ok {
			id = rest
			break
		}
	}
	if i := strings.IndexByte(id, '_'); i > 0 {
		id = id[:i]
	}
	return store.NormalizeID(id)
}

// sortLib é a ordem padrão de faixas: artista do álbum, álbum, disco, faixa.
const sortLib = ` ORDER BY i.album_artist COLLATE NOCASE, i.album COLLATE NOCASE, i.album_id, i.parent_idx, i.idx, i.sort_name, i.rid`
