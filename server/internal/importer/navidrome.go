package importer

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// navidromeDefaultKey é a PasswordEncryptionKey de quando o Navidrome não
// tem uma configurada.
const navidromeDefaultKey = "just for obfuscation"

// navidromePassword abre a senha do Navidrome: AES-GCM com
// sha256(chave), nonce na frente, base64.
func navidromePassword(key, enc string) (string, error) {
	if key == "" {
		key = navidromeDefaultKey
	}
	sum := sha256.Sum256([]byte(key))
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("senha cifrada curta")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(plain), err
}

// ndTrack é uma faixa do Navidrome já casada (ou não) com o acervo.
type ndTrack struct {
	it              *item // nil = sem par
	albumID, artist string
}

// Navidrome importa do navidrome.db (0.55+: media_file.path relativo à
// pasta da biblioteca; antes era absoluto, os dois funcionam):
//
//   - user: a senha é decifrada com a chave e o usuário nasce com a senha
//     de verdade (o token do Subsonic funciona na hora).
//   - annotation: estrela → favorito, vezes, última vez e nota de faixas,
//     álbuns e artistas.
//   - playqueue → subsonic_playqueue; bookmark → subsonic_bookmarks (e onde
//     parou no estado do item).
//   - playlist (as normais; as inteligentes ficam de fora) → ListSink.
func (im *Importer) Navidrome(ctx context.Context, file, key string) error {
	db, err := openSQLite(file)
	if err != nil {
		return err
	}
	defer db.Close()
	for _, t := range []string{"user", "annotation", "media_file", "album", "artist"} {
		if !hasTable(ctx, db, t) {
			return errors.New("navidrome.db sem a tabela " + t)
		}
	}
	if err := im.begin(ctx, "navidrome"); err != nil {
		return err
	}
	users, err := im.navidromeUsers(ctx, db, key)
	if err != nil {
		return err
	}
	tracks, err := im.navidromeTracks(ctx, db)
	if err != nil {
		return err
	}
	albums, artists, err := im.navidromeAlbums(ctx, db, tracks)
	if err != nil {
		return err
	}
	if err := im.navidromeAnnotations(ctx, db, users, tracks, albums, artists); err != nil {
		return err
	}
	if err := im.navidromeQueues(ctx, db, users, tracks); err != nil {
		return err
	}
	if err := im.navidromeBookmarks(ctx, db, users, tracks); err != nil {
		return err
	}
	if err := im.navidromePlaylists(ctx, db, users, tracks); err != nil {
		return err
	}
	return im.finish(ctx)
}

func (im *Importer) navidromeUsers(ctx context.Context, db *sql.DB, key string) (map[string]userRef, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, user_name, password, is_admin FROM user ORDER BY user_name`)
	if err != nil {
		return nil, err
	}
	type row struct {
		id, name, enc string
		admin         bool
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.enc, &r.admin); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[string]userRef{}
	for _, r := range list {
		in := auth.Imported{Name: r.name, Admin: r.admin}
		if r.enc != "" {
			pass, err := navidromePassword(key, r.enc)
			if err != nil {
				im.rep.Note("%s: senha não abriu com essa chave (--key = PasswordEncryptionKey do Navidrome); importado sem senha", r.name)
			}
			in.Password = pass
		}
		u, err := im.user(ctx, in)
		if err != nil {
			return nil, err
		}
		out[r.id] = u
	}
	return out, nil
}

// navidromeTracks casa cada media_file com uma faixa do acervo pelo
// caminho (pasta da biblioteca + path, ou path absoluto das versões velhas).
func (im *Importer) navidromeTracks(ctx context.Context, db *sql.DB) (map[string]ndTrack, error) {
	q := `SELECT mf.id, mf.path, '', COALESCE(mf.album_id,''), COALESCE(mf.album_artist_id,'') FROM media_file mf`
	if hasTable(ctx, db, "library") && hasColumn(ctx, db, "media_file", "library_id") {
		q = `SELECT mf.id, mf.path, COALESCE(l.path,''), COALESCE(mf.album_id,''), COALESCE(mf.album_artist_id,'')
FROM media_file mf LEFT JOIN library l ON l.id = mf.library_id`
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ndTrack{}
	for rows.Next() {
		var id, p, lib string
		var t ndTrack
		if err := rows.Scan(&id, &p, &lib, &t.albumID, &t.artist); err != nil {
			return nil, err
		}
		if !path.IsAbs(filepath.ToSlash(p)) && lib != "" {
			p = path.Join(filepath.ToSlash(lib), filepath.ToSlash(p))
		}
		mapped := im.Maps.Apply(p)
		t.it = im.idx.byPath[mapped]
		if t.it == nil || t.it.Kind != store.KindTrack {
			t.it = nil
			im.rep.Miss("faixa", mapped)
		}
		out[id] = t
	}
	return out, rows.Err()
}

// navidromeAlbums casa álbuns e artistas: primeiro pelas faixas (o álbum da
// faixa casada), senão pela chave de nome que o scanner usa.
func (im *Importer) navidromeAlbums(ctx context.Context, db *sql.DB, tracks map[string]ndTrack) (albums, artists map[string]*item, err error) {
	albums, artists = map[string]*item{}, map[string]*item{}
	byID := map[string]*item{}
	for _, it := range im.idx.byPath {
		if it.Kind == store.KindAlbum || it.Kind == store.KindArtist {
			byID[it.ID] = it
		}
	}
	for _, t := range tracks {
		if t.it == nil {
			continue
		}
		al := byID[t.it.AlbumID]
		if al == nil {
			continue
		}
		if t.albumID != "" && albums[t.albumID] == nil {
			albums[t.albumID] = al
		}
		if ar := byID[al.ArtistID]; ar != nil && t.artist != "" && artists[t.artist] == nil {
			artists[t.artist] = ar
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT id, name, COALESCE(album_artist,'') FROM album`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id, name, artist string
		if err := rows.Scan(&id, &name, &artist); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if albums[id] == nil {
			if it := im.idx.byPath[albumKey(artist, name)]; it != nil {
				albums[id] = it
			}
		}
	}
	rows.Close()
	rows, err = db.QueryContext(ctx, `SELECT id, name FROM artist`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, err
		}
		if artists[id] == nil {
			if it := im.idx.byPath[artistKey(name)]; it != nil {
				artists[id] = it
			}
		}
	}
	return albums, artists, rows.Err()
}

func (im *Importer) navidromeAnnotations(ctx context.Context, db *sql.DB, users map[string]userRef, tracks map[string]ndTrack,
	albums, artists map[string]*item) error {
	rows, err := db.QueryContext(ctx, `SELECT user_id, item_id, item_type, COALESCE(play_count,0), play_date, COALESCE(rating,0),
  COALESCE(starred,0), starred_at FROM annotation`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, id, typ string
		var plays, rating int
		var starred bool
		var played, starredAt any
		if err := rows.Scan(&uid, &id, &typ, &plays, &played, &rating, &starred, &starredAt); err != nil {
			return err
		}
		u, ok := users[uid]
		if !ok {
			im.rep.Add("estado de usuário não importado", 1)
			continue
		}
		d := store.UserData{Favorite: starred, Rating: max(0, min(5, rating))}
		if starred {
			d.FavoriteAt = toTime(starredAt)
		}
		var it *item
		switch typ {
		case "media_file":
			it = tracks[id].it
			d.PlayCount, d.LastPlayed, d.Played = plays, toTime(played), plays > 0
		case "album":
			it = albums[id]
		case "artist":
			it = artists[id]
		default:
			continue // playlist, etc.
		}
		if it == nil {
			im.rep.Add("anotações sem par ("+typ+")", 1)
			continue
		}
		im.rep.Add("anotações casadas", 1)
		im.addUserData(u.ID, it.Kind, it, d)
	}
	return rows.Err()
}

// ndIDs troca os ids de faixa do Navidrome pelos do acervo (sem par sai).
func ndIDs(tracks map[string]ndTrack, src []string) []string {
	out := []string{}
	for _, id := range src {
		if t := tracks[strings.TrimSpace(id)]; t.it != nil {
			out = append(out, t.it.ID)
		}
	}
	return out
}

// navidromeQueues traz a fila. current é índice (0.56+) ou id da faixa
// (antes); a fila só entra se for mais nova que a que o arkflix já tem.
func (im *Importer) navidromeQueues(ctx context.Context, db *sql.DB, users map[string]userRef, tracks map[string]ndTrack) error {
	if !hasTable(ctx, db, "playqueue") {
		return nil
	}
	rows, err := db.QueryContext(ctx, `SELECT user_id, COALESCE(CAST(current AS TEXT),''), CAST(COALESCE(position,0) AS INTEGER),
  COALESCE(items,''), COALESCE(changed_by,''), updated_at FROM playqueue`)
	if err != nil {
		return err
	}
	type queue struct {
		u         userRef
		ids       []string
		cur       int
		pos       int64
		by        string
		changedAt time.Time
	}
	var list []queue
	for rows.Next() {
		var uid, current, items, by string
		var pos int64
		var upd any
		if err := rows.Scan(&uid, &current, &pos, &items, &by, &upd); err != nil {
			rows.Close()
			return err
		}
		u, ok := users[uid]
		if !ok || items == "" {
			continue
		}
		src := strings.Split(items, ",")
		curID := current
		if n, err := strconv.Atoi(current); err == nil && n >= 0 && n < len(src) {
			curID = src[n]
		}
		q := queue{u: u, by: by, changedAt: toTime(upd)}
		found := false
		for _, id := range src {
			id = strings.TrimSpace(id)
			t := tracks[id]
			if t.it == nil {
				continue
			}
			if !found && id == curID {
				found, q.cur, q.pos = true, len(q.ids), pos
			}
			q.ids = append(q.ids, t.it.ID)
		}
		if len(q.ids) > 0 {
			list = append(list, q)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, q := range list {
		var old string
		err := im.D.Store.DB.QueryRowContext(ctx, `SELECT changed_at FROM subsonic_playqueue WHERE user_id = ?`, q.u.ID).Scan(&old)
		if err == nil && !q.changedAt.After(toTime(old)) {
			im.rep.Add("filas sem mudança", 1)
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		im.rep.Add("filas gravadas", 1)
		if im.Dry {
			continue
		}
		raw, _ := json.Marshal(q.ids)
		_, err = im.D.Store.DB.ExecContext(ctx, `INSERT INTO subsonic_playqueue (user_id, items, current, position, changed_at, changed_by)
VALUES (?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET items=excluded.items, current=excluded.current, position=excluded.position,
  changed_at=excluded.changed_at, changed_by=excluded.changed_by`, q.u.ID, string(raw), q.cur, q.pos, fmtTime(q.changedAt), q.by)
		if err != nil {
			return err
		}
	}
	return nil
}

// navidromeBookmarks traz os marcadores (posição em ms) e põe a posição
// no estado do item, como o savebookmark do arkflix faz.
func (im *Importer) navidromeBookmarks(ctx context.Context, db *sql.DB, users map[string]userRef, tracks map[string]ndTrack) error {
	if !hasTable(ctx, db, "bookmark") {
		return nil
	}
	rows, err := db.QueryContext(ctx, `SELECT user_id, item_id, COALESCE(comment,''), COALESCE(position,0), created_at, updated_at
FROM bookmark WHERE item_type = 'media_file'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, id, comment string
		var pos int64
		var created, updated any
		if err := rows.Scan(&uid, &id, &comment, &pos, &created, &updated); err != nil {
			return err
		}
		u, ok := users[uid]
		t := tracks[id]
		if !ok || t.it == nil {
			im.rep.Add("marcadores sem par", 1)
			continue
		}
		up := toTime(updated)
		var old string
		err := im.D.Store.DB.QueryRowContext(ctx, `SELECT changed_at FROM subsonic_bookmarks WHERE user_id = ? AND item_id = ?`, u.ID, t.it.ID).Scan(&old)
		if err == nil && !up.After(toTime(old)) {
			im.rep.Add("marcadores sem mudança", 1)
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		im.rep.Add("marcadores gravados", 1)
		im.addUserData(u.ID, t.it.Kind, t.it, store.UserData{Position: float64(pos) / 1000, LastPlayed: up})
		if im.Dry {
			continue
		}
		_, err = im.D.Store.DB.ExecContext(ctx, `INSERT INTO subsonic_bookmarks (user_id, item_id, position, comment, created_at, changed_at)
VALUES (?,?,?,?,?,?) ON CONFLICT(user_id, item_id) DO UPDATE SET position=excluded.position, comment=excluded.comment,
  changed_at=excluded.changed_at`, u.ID, t.it.ID, pos, comment, fmtTime(toTime(created)), fmtTime(up))
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

// navidromePlaylists manda as playlists normais pro ListSink; as
// inteligentes (rules) são uma consulta, não uma lista, e ficam de fora.
func (im *Importer) navidromePlaylists(ctx context.Context, db *sql.DB, users map[string]userRef, tracks map[string]ndTrack) error {
	if !hasTable(ctx, db, "playlist") || !hasTable(ctx, db, "playlist_tracks") {
		return nil
	}
	rules := "''"
	if hasColumn(ctx, db, "playlist", "rules") {
		rules = "COALESCE(rules,'')"
	}
	rows, err := db.QueryContext(ctx, `SELECT id, name, owner_id, `+rules+` FROM playlist ORDER BY name`)
	if err != nil {
		return err
	}
	type pl struct{ id, name, owner, rules string }
	var list []pl
	for rows.Next() {
		var p pl
		if err := rows.Scan(&p.id, &p.name, &p.owner, &p.rules); err != nil {
			rows.Close()
			return err
		}
		list = append(list, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range list {
		u, ok := users[p.owner]
		if !ok {
			continue
		}
		if p.rules != "" && p.rules != "null" {
			im.rep.Add("playlists inteligentes (puladas)", 1)
			continue
		}
		ids, err := playlistTracks(ctx, db, p.id)
		if err != nil {
			return err
		}
		mapped := ndIDs(tracks, ids)
		if n := len(ids) - len(mapped); n > 0 {
			im.rep.Add("faixas de playlist sem par", n)
		}
		if err := im.list(ctx, u, p.name, mapped); err != nil {
			return err
		}
	}
	return nil
}

func playlistTracks(ctx context.Context, db *sql.DB, id string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT media_file_id FROM playlist_tracks WHERE playlist_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
