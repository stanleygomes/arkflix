package importer

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Linhas do RomM que o import lê (5.x, MariaDB).
type (
	RomMUser struct {
		ID       int64
		Name     string
		Hash     string // bcrypt; "" = só OIDC
		Role     string // ADMIN | EDITOR | VIEWER (5.x); admin | user (mais novo)
		Disabled bool
	}
	RomMRom struct {
		ID     int64
		FSName string // arquivo ou pasta da ROM
		FSPath string // relativo à biblioteca: roms/<fs_slug>
		Slug   string // platforms.fs_slug
	}
	RomMAsset struct {
		Kind     string // saves | states | screenshots
		UserID   int64
		RomID    int64
		FileName string
		FilePath string // pasta relativa aos assets: users/<u>/saves/<slug>/<rom>[/<emu>]
		Emulator string
		Created  time.Time
		Updated  time.Time
	}
	RomMRomUser struct {
		UserID, RomID                  int64
		IsMainSibling, Backlogged      bool
		NowPlaying, Hidden             bool
		Rating, Difficulty, Completion int
		Status                         string // INCOMPLETE... ("" = nenhum)
		LastPlayed                     time.Time
	}
	RomMCollection struct {
		ID     int64
		Name   string
		UserID int64
		Roms   []int64
	}
)

// RomMSource é de onde vêm as linhas do RomM. A do MariaDB é SQLRomM numa
// transação só de leitura; os testes usam SQLRomM num SQLite com as
// mesmas tabelas.
type RomMSource interface {
	Users(ctx context.Context) ([]RomMUser, error)
	Roms(ctx context.Context) ([]RomMRom, error)
	Assets(ctx context.Context, kind string) ([]RomMAsset, error)
	RomUsers(ctx context.Context) ([]RomMRomUser, error)
	Collections(ctx context.Context) ([]RomMCollection, error)
}

// RomMOptions são as pastas do RomM.
type RomMOptions struct {
	Assets  string // onde está /romm/assets (file_path dos saves é relativo a ela)
	Library string // /romm/library como o RomM vê (passa por --map)
}

// RomM importa: usuários (bcrypt serve como está; a cópia cifrada pro
// Subsonic nasce no 1º login), saves/states/screenshots (arquivo copiado
// pro layout do arkflix + linha em romm_assets), props do jogo
// (romm_rom_users; last_played vai pro estado) e coleções (ListSink).
func (im *Importer) RomM(ctx context.Context, src RomMSource, opt RomMOptions) error {
	if opt.Library == "" {
		opt.Library = "/romm/library"
	}
	if opt.Assets == "" {
		return errors.New("falta --assets (a pasta /romm/assets do RomM, de onde vêm os saves)")
	}
	if err := im.begin(ctx, "romm"); err != nil {
		return err
	}
	users, err := im.rommUsers(ctx, src)
	if err != nil {
		return err
	}
	roms, err := im.rommRoms(ctx, src, opt)
	if err != nil {
		return err
	}
	for _, k := range []string{"saves", "states", "screenshots"} {
		if err := im.rommAssets(ctx, src, opt, k, users, roms); err != nil {
			return err
		}
	}
	if err := im.rommProps(ctx, src, users, roms); err != nil {
		return err
	}
	cols, err := src.Collections(ctx)
	if err != nil {
		return err
	}
	for _, c := range cols {
		u, ok := users[c.UserID]
		if !ok {
			continue
		}
		var ids []string
		for _, r := range c.Roms {
			if it := roms[r]; it != nil {
				ids = append(ids, it.ID)
			} else {
				im.rep.Add("jogos de coleção sem par", 1)
			}
		}
		if err := im.list(ctx, u, c.Name, ids); err != nil {
			return err
		}
	}
	return im.finish(ctx)
}

func (im *Importer) rommUsers(ctx context.Context, src RomMSource) (map[int64]userRef, error) {
	list, err := src.Users(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]userRef{}
	for _, r := range list {
		if r.Disabled {
			im.rep.Add("usuários desativados (pulados)", 1)
			continue
		}
		u, err := im.user(ctx, auth.Imported{Name: r.Name, Admin: strings.EqualFold(r.Role, "admin"), Hash: r.Hash})
		if err != nil {
			return nil, err
		}
		out[r.ID] = u
	}
	return out, nil
}

// rommRoms casa cada ROM pelo caminho (biblioteca + fs_path + fs_name,
// passando por --map) ou, se não der, por console + nome do arquivo.
func (im *Importer) rommRoms(ctx context.Context, src RomMSource, opt RomMOptions) (map[int64]*item, error) {
	list, err := src.Roms(ctx)
	if err != nil {
		return nil, err
	}
	out := map[int64]*item{}
	for _, r := range list {
		p := im.Maps.Apply(path.Join(opt.Library, r.FSPath, r.FSName))
		it := im.idx.byPath[p]
		if it == nil || it.Kind != store.KindRom {
			it = im.idx.roms[r.Slug+"/"+r.FSName]
		}
		if it == nil {
			im.rep.Miss("rom", p)
			continue
		}
		out[r.ID] = it
	}
	im.rep.Add("roms casadas", len(out))
	return out, nil
}

// safeName confere nome de arquivo/pasta que vira caminho no disco.
func safeName(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\\x00") && len(s) <= 255 &&
		strings.IndexFunc(s, unicode.IsControl) < 0
}

func (im *Importer) rommAssets(ctx context.Context, src RomMSource, opt RomMOptions, kind string, users map[int64]userRef,
	roms map[int64]*item) error {
	list, err := src.Assets(ctx, kind)
	if err != nil {
		return err
	}
	for _, a := range list {
		u, ok := users[a.UserID]
		rom := roms[a.RomID]
		switch {
		case !ok:
			im.rep.Add(kind+" de usuário não importado", 1)
			continue
		case rom == nil:
			im.rep.Add(kind+" de rom sem par", 1)
			continue
		case !safeName(a.FileName) || (a.Emulator != "" && !safeName(a.Emulator)):
			im.rep.Note("%s %q: nome inválido, pulado", kind, a.FileName)
			continue
		}
		from := filepath.Join(filepath.FromSlash(a.FilePath), a.FileName)
		if !filepath.IsAbs(from) {
			from = filepath.Join(opt.Assets, from)
		} else {
			from = filepath.FromSlash(im.Maps.Apply(filepath.ToSlash(from)))
		}
		if err := im.copyAsset(ctx, kind, u, rom, a, from); err != nil {
			return err
		}
	}
	return nil
}

// copyAsset copia o arquivo pro mesmo layout que o upload do arkflix usa
// (<data>/romm/<tipo>/<rid do usuário>/<console>/<rid da rom>[/<emulador>])
// e grava a linha. Arquivo igual (mesmo md5) já copiado não é copiado de novo.
func (im *Importer) copyAsset(ctx context.Context, kind string, u userRef, rom *item, a RomMAsset, from string) error {
	st, err := os.Stat(from)
	if err != nil || st.IsDir() {
		im.rep.Miss(kind+" (arquivo sumiu)", from)
		return nil
	}
	parts := []string{"romm", kind, strconv.FormatInt(u.RID, 10), rom.Platform, strconv.FormatInt(rom.RID, 10)}
	if a.Emulator != "" {
		parts = append(parts, a.Emulator)
	}
	rel := path.Join(parts...)
	dir := filepath.Join(im.D.Config.Data, filepath.FromSlash(rel))
	srcHash, err := md5File(from)
	if err != nil {
		return err
	}
	var oldHash string
	err = im.D.Store.DB.QueryRowContext(ctx, `SELECT content_hash FROM romm_assets WHERE kind = ? AND user_id = ? AND rom_rid = ? AND emulator = ? AND file_name = ?`,
		kind, u.ID, rom.RID, a.Emulator, a.FileName).Scan(&oldHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if oldHash == srcHash {
		if dst, err := md5File(filepath.Join(dir, a.FileName)); err == nil && dst == srcHash {
			im.rep.Add(kind+" sem mudança", 1)
			return nil
		}
	}
	im.rep.Add(kind+" copiados", 1)
	if im.Dry {
		return nil
	}
	size, err := copyFile(from, dir, a.FileName)
	if err != nil {
		return fmt.Errorf("%s %s: %w", kind, from, err)
	}
	created, updated := a.Created, a.Updated
	if updated.IsZero() {
		updated = st.ModTime()
	}
	if created.IsZero() {
		created = updated
	}
	_, err = im.D.Store.DB.ExecContext(ctx, `
INSERT INTO romm_assets (kind, user_id, rom_rid, emulator, file_name, file_path, size, content_hash, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(kind, user_id, rom_rid, emulator, file_name) DO UPDATE SET file_path=excluded.file_path, size=excluded.size,
  content_hash=excluded.content_hash, updated_at=excluded.updated_at`,
		kind, u.ID, rom.RID, a.Emulator, a.FileName, rel, size, srcHash, fmtTime(created), fmtTime(updated))
	return err
}

func md5File(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copia pra dir/name (temporário + rename: nunca fica pela metade).
func copyFile(from, dir, name string) (int64, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}
	in, err := os.Open(from)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(dir, ".import-*")
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(tmp, in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), filepath.Join(dir, name))
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return n, err
}

// rommStatuses são os status de rom_user (o MariaDB guarda o nome do
// enum, em maiúsculas).
var rommStatuses = map[string]bool{"incomplete": true, "finished": true, "completed_100": true, "retired": true, "never_playing": true}

// rommProps traz o rom_user (status, nota, backlog...) e o last_played
// (que no arkflix mora no estado do item).
func (im *Importer) rommProps(ctx context.Context, src RomMSource, users map[int64]userRef, roms map[int64]*item) error {
	list, err := src.RomUsers(ctx)
	if err != nil {
		return err
	}
	for _, r := range list {
		u, ok := users[r.UserID]
		rom := roms[r.RomID]
		if !ok || rom == nil {
			im.rep.Add("props sem par", 1)
			continue
		}
		if !r.LastPlayed.IsZero() {
			im.addUserData(u.ID, rom.Kind, rom, store.UserData{LastPlayed: r.LastPlayed})
		}
		var status any
		if st := strings.ToLower(r.Status); rommStatuses[st] {
			status = st
		}
		var same int
		err := im.D.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM romm_rom_users WHERE user_id = ? AND rom_rid = ? AND is_main_sibling = ?
  AND backlogged = ? AND now_playing = ? AND hidden = ? AND rating = ? AND difficulty = ? AND completion = ? AND status IS ?`,
			u.ID, rom.RID, r.IsMainSibling, r.Backlogged, r.NowPlaying, r.Hidden, r.Rating, r.Difficulty, r.Completion, status).Scan(&same)
		if err != nil {
			return err
		}
		if same > 0 {
			im.rep.Add("props sem mudança", 1)
			continue
		}
		im.rep.Add("props gravadas", 1)
		if im.Dry {
			continue
		}
		now := store.Now()
		_, err = im.D.Store.DB.ExecContext(ctx, `
INSERT INTO romm_rom_users (user_id, rom_rid, is_main_sibling, backlogged, now_playing, hidden, rating, difficulty, completion,
  status, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(user_id, rom_rid) DO UPDATE SET is_main_sibling=excluded.is_main_sibling, backlogged=excluded.backlogged,
  now_playing=excluded.now_playing, hidden=excluded.hidden, rating=excluded.rating, difficulty=excluded.difficulty,
  completion=excluded.completion, status=excluded.status, updated_at=excluded.updated_at`,
			u.ID, rom.RID, r.IsMainSibling, r.Backlogged, r.NowPlaying, r.Hidden, r.Rating, r.Difficulty, r.Completion, status, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// Querier é o que SQLRomM precisa (*sql.DB ou *sql.Tx).
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SQLRomM lê o banco do RomM com SQL que roda igual no MariaDB e no
// SQLite (os testes usam SQLite com as mesmas tabelas e colunas).
type SQLRomM struct{ Q Querier }

func (s SQLRomM) Users(ctx context.Context) ([]RomMUser, error) {
	return query(ctx, s.Q, `SELECT id, username, COALESCE(hashed_password,''), COALESCE(role,''), enabled FROM users ORDER BY id`,
		func(r *sql.Rows) (RomMUser, error) {
			var u RomMUser
			var enabled bool
			err := r.Scan(&u.ID, &u.Name, &u.Hash, &u.Role, &enabled)
			u.Disabled = !enabled
			return u, err
		})
}

func (s SQLRomM) Roms(ctx context.Context) ([]RomMRom, error) {
	return query(ctx, s.Q, `SELECT r.id, r.fs_name, r.fs_path, p.fs_slug FROM roms r JOIN platforms p ON p.id = r.platform_id ORDER BY r.id`,
		func(r *sql.Rows) (RomMRom, error) {
			var x RomMRom
			return x, r.Scan(&x.ID, &x.FSName, &x.FSPath, &x.Slug)
		})
}

func (s SQLRomM) Assets(ctx context.Context, kind string) ([]RomMAsset, error) {
	emu := "COALESCE(emulator,'')"
	switch kind {
	case "saves", "states":
	case "screenshots":
		emu = "''"
	default:
		return nil, errors.New("tipo de asset desconhecido: " + kind)
	}
	return query(ctx, s.Q, `SELECT user_id, rom_id, file_name, file_path, `+emu+`, created_at, updated_at FROM `+kind+
		` WHERE rom_id IS NOT NULL ORDER BY id`,
		func(r *sql.Rows) (RomMAsset, error) {
			a := RomMAsset{Kind: kind}
			var created, updated any
			err := r.Scan(&a.UserID, &a.RomID, &a.FileName, &a.FilePath, &a.Emulator, &created, &updated)
			a.Created, a.Updated = toTime(created), toTime(updated)
			return a, err
		})
}

func (s SQLRomM) RomUsers(ctx context.Context) ([]RomMRomUser, error) {
	return query(ctx, s.Q, `SELECT user_id, rom_id, is_main_sibling, backlogged, now_playing, hidden, COALESCE(rating,0),
  COALESCE(difficulty,0), COALESCE(completion,0), COALESCE(status,''), last_played FROM rom_user ORDER BY id`,
		func(r *sql.Rows) (RomMRomUser, error) {
			var x RomMRomUser
			var last any
			err := r.Scan(&x.UserID, &x.RomID, &x.IsMainSibling, &x.Backlogged, &x.NowPlaying, &x.Hidden, &x.Rating,
				&x.Difficulty, &x.Completion, &x.Status, &last)
			x.LastPlayed = toTime(last)
			return x, err
		})
}

func (s SQLRomM) Collections(ctx context.Context) ([]RomMCollection, error) {
	cols, err := query(ctx, s.Q, `SELECT id, name, user_id FROM collections ORDER BY id`, func(r *sql.Rows) (RomMCollection, error) {
		var c RomMCollection
		return c, r.Scan(&c.ID, &c.Name, &c.UserID)
	})
	if err != nil {
		return nil, err
	}
	type pair struct{ col, rom int64 }
	pairs, err := query(ctx, s.Q, `SELECT collection_id, rom_id FROM collections_roms ORDER BY collection_id, rom_id`,
		func(r *sql.Rows) (pair, error) {
			var p pair
			return p, r.Scan(&p.col, &p.rom)
		})
	if err != nil {
		return nil, err
	}
	byID := map[int64]*RomMCollection{}
	for i := range cols {
		byID[cols[i].ID] = &cols[i]
	}
	for _, p := range pairs {
		if c := byID[p.col]; c != nil {
			c.Roms = append(c.Roms, p.rom)
		}
	}
	return cols, nil
}

// query roda q e lê cada linha com scan.
func query[T any](ctx context.Context, q Querier, sqlText string, scan func(*sql.Rows) (T, error)) ([]T, error) {
	rows, err := q.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
