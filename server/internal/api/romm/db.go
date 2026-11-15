package romm

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

func init() {
	store.Register(store.Migration{Name: "romm/001-init", SQL: `
-- arquivos de cada ROM com id numérico estável (o RomM baixa arquivo por id)
CREATE TABLE romm_files (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  rom_rid INTEGER NOT NULL,
  rel     TEXT NOT NULL,              -- RomFile.Rel
  UNIQUE (rom_rid, rel)
);

-- client tokens (rmm_...): o token mora em tokens; aqui o id e os escopos
CREATE TABLE romm_client_tokens (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  token      TEXT NOT NULL UNIQUE REFERENCES tokens(token) ON DELETE CASCADE,
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  scopes     TEXT NOT NULL,             -- separados por espaço
  device_id  TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX romm_client_tokens_user ON romm_client_tokens(user_id);

-- saves, states e screenshots de cada usuário
CREATE TABLE romm_assets (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  kind         TEXT NOT NULL,           -- saves | states | screenshots
  user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rom_rid      INTEGER NOT NULL,
  emulator     TEXT NOT NULL DEFAULT '',
  file_name    TEXT NOT NULL,
  file_path    TEXT NOT NULL,           -- pasta relativa a <data>
  size         INTEGER NOT NULL DEFAULT 0,
  content_hash TEXT NOT NULL DEFAULT '',
  is_gallery   INTEGER NOT NULL DEFAULT 0,
  is_public    INTEGER NOT NULL DEFAULT 0,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL,
  UNIQUE (kind, user_id, rom_rid, emulator, file_name)
);
CREATE INDEX romm_assets_user ON romm_assets(user_id, kind, rom_rid);

-- props do usuário na ROM (rom_user); last_played mora em userdata
CREATE TABLE romm_rom_users (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rom_rid         INTEGER NOT NULL,
  is_main_sibling INTEGER NOT NULL DEFAULT 0,
  backlogged      INTEGER NOT NULL DEFAULT 0,
  now_playing     INTEGER NOT NULL DEFAULT 0,
  hidden          INTEGER NOT NULL DEFAULT 0,
  rating          INTEGER NOT NULL DEFAULT 0,
  difficulty      INTEGER NOT NULL DEFAULT 0,
  completion      INTEGER NOT NULL DEFAULT 0,
  status          TEXT,
  pinned_media    TEXT,                 -- JSON ou NULL
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  UNIQUE (user_id, rom_rid)
);
`})
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// fileIDs dá (e cria, se faltar) o id de cada arquivo da ROM.
func (a *api) fileIDs(ctx context.Context, romRID int64, rels []string) (map[string]int64, error) {
	db := a.d.Store.DB
	out := make(map[string]int64, len(rels))
	rows, err := db.QueryContext(ctx, `SELECT id, rel FROM romm_files WHERE rom_rid = ?`, romRID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var rel string
		if err := rows.Scan(&id, &rel); err != nil {
			rows.Close()
			return nil, err
		}
		out[rel] = id
	}
	rows.Close()
	for _, rel := range rels {
		if _, ok := out[rel]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO romm_files (rom_rid, rel) VALUES (?, ?)`, romRID, rel); err != nil {
			return nil, err
		}
		var id int64
		if err := db.QueryRowContext(ctx, `SELECT id FROM romm_files WHERE rom_rid = ? AND rel = ?`, romRID, rel).Scan(&id); err != nil {
			return nil, err
		}
		out[rel] = id
	}
	return out, nil
}

// fileByID devolve a ROM e o caminho relativo de um id de arquivo.
func (a *api) fileByID(ctx context.Context, id int64) (romRID int64, rel string, err error) {
	err = a.d.Store.DB.QueryRowContext(ctx, `SELECT rom_rid, rel FROM romm_files WHERE id = ?`, id).Scan(&romRID, &rel)
	if errors.Is(err, sql.ErrNoRows) {
		err = store.ErrNotFound
	}
	return
}

// clientToken é uma linha de romm_client_tokens com o uso vindo de tokens.
type clientToken struct {
	ID       int64
	Token    string
	UserID   string
	Name     string
	Scopes   []string
	DeviceID string
	Created  time.Time
	Expires  time.Time
	LastUsed time.Time
}

const clientTokenCols = `c.id, c.token, c.user_id, c.name, c.scopes, COALESCE(c.device_id,''), c.created_at,
COALESCE(t.expires_at,''), COALESCE(t.last_used,'')`

func scanClientToken(r interface{ Scan(...any) error }) (clientToken, error) {
	var ct clientToken
	var scopes, created, exp, used string
	err := r.Scan(&ct.ID, &ct.Token, &ct.UserID, &ct.Name, &scopes, &ct.DeviceID, &created, &exp, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return ct, store.ErrNotFound
	}
	ct.Scopes = strings.Fields(scopes)
	ct.Created, ct.Expires, ct.LastUsed = parseTime(created), parseTime(exp), parseTime(used)
	return ct, err
}

func (a *api) clientTokens(ctx context.Context, where string, args ...any) ([]clientToken, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT `+clientTokenCols+` FROM romm_client_tokens c
JOIN tokens t ON t.token = c.token WHERE `+where+` ORDER BY c.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []clientToken{}
	for rows.Next() {
		ct, err := scanClientToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}

func (a *api) clientTokenByValue(ctx context.Context, token string) (clientToken, error) {
	return scanClientToken(a.d.Store.DB.QueryRowContext(ctx, `SELECT `+clientTokenCols+` FROM romm_client_tokens c
JOIN tokens t ON t.token = c.token WHERE c.token = ?`, token))
}

func (a *api) clientTokenByID(ctx context.Context, id int64) (clientToken, error) {
	return scanClientToken(a.d.Store.DB.QueryRowContext(ctx, `SELECT `+clientTokenCols+` FROM romm_client_tokens c
JOIN tokens t ON t.token = c.token WHERE c.id = ?`, id))
}

// asset é um save, state ou screenshot.
type asset struct {
	ID          int64
	Kind        string
	UserID      string
	RomRID      int64
	Emulator    string
	FileName    string
	FilePath    string // pasta relativa a <data>
	Size        int64
	ContentHash string
	IsGallery   bool
	IsPublic    bool
	Created     time.Time
	Updated     time.Time
}

const assetCols = `id, kind, user_id, rom_rid, emulator, file_name, file_path, size, content_hash, is_gallery, is_public, created_at, updated_at`

func scanAsset(r interface{ Scan(...any) error }) (*asset, error) {
	var s asset
	var created, updated string
	err := r.Scan(&s.ID, &s.Kind, &s.UserID, &s.RomRID, &s.Emulator, &s.FileName, &s.FilePath, &s.Size, &s.ContentHash,
		&s.IsGallery, &s.IsPublic, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.Created, s.Updated = parseTime(created), parseTime(updated)
	return &s, nil
}

func (a *api) assets(ctx context.Context, where string, args ...any) ([]*asset, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT `+assetCols+` FROM romm_assets WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*asset
	for rows.Next() {
		s, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (a *api) assetByID(ctx context.Context, kind string, id int64) (*asset, error) {
	return scanAsset(a.d.Store.DB.QueryRowContext(ctx, `SELECT `+assetCols+` FROM romm_assets WHERE kind = ? AND id = ?`, kind, id))
}

// romUser é a linha de romm_rom_users (props do usuário na ROM).
type romUser struct {
	ID            int64
	UserID        string
	RomRID        int64
	IsMainSibling bool
	Backlogged    bool
	NowPlaying    bool
	Hidden        bool
	Rating        int
	Difficulty    int
	Completion    int
	Status        sql.NullString
	PinnedMedia   sql.NullString
	Created       time.Time
	Updated       time.Time
}

func (a *api) romUser(ctx context.Context, userID string, romRID int64) (*romUser, error) {
	var u romUser
	var created, updated string
	err := a.d.Store.DB.QueryRowContext(ctx, `SELECT id, user_id, rom_rid, is_main_sibling, backlogged, now_playing, hidden,
rating, difficulty, completion, status, pinned_media, created_at, updated_at FROM romm_rom_users WHERE user_id = ? AND rom_rid = ?`,
		userID, romRID).Scan(&u.ID, &u.UserID, &u.RomRID, &u.IsMainSibling, &u.Backlogged, &u.NowPlaying, &u.Hidden,
		&u.Rating, &u.Difficulty, &u.Completion, &u.Status, &u.PinnedMedia, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Created, u.Updated = parseTime(created), parseTime(updated)
	return &u, nil
}

func (a *api) saveRomUser(ctx context.Context, u *romUser) error {
	now := store.Now()
	_, err := a.d.Store.DB.ExecContext(ctx, `
INSERT INTO romm_rom_users (user_id, rom_rid, is_main_sibling, backlogged, now_playing, hidden, rating, difficulty, completion,
  status, pinned_media, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(user_id, rom_rid) DO UPDATE SET is_main_sibling=excluded.is_main_sibling, backlogged=excluded.backlogged,
  now_playing=excluded.now_playing, hidden=excluded.hidden, rating=excluded.rating, difficulty=excluded.difficulty,
  completion=excluded.completion, status=excluded.status, pinned_media=excluded.pinned_media, updated_at=excluded.updated_at`,
		u.UserID, u.RomRID, u.IsMainSibling, u.Backlogged, u.NowPlaying, u.Hidden, u.Rating, u.Difficulty, u.Completion,
		u.Status, u.PinnedMedia, now, now)
	return err
}
