// Package store guarda tudo do servidor num SQLite só (storage/arkflix.db):
// usuários, tokens, o acervo escaneado e o estado de cada usuário (onde
// parou, favoritos, nota). Cada API (Jellyfin, Subsonic, RomM) lê daqui; o
// que é só dela (saves do RomM, playlists do Subsonic) entra por Register.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"

	_ "modernc.org/sqlite"
)

// Store é o banco aberto. DB fica exposto pra SQL próprio de cada API.
type Store struct {
	DB *sql.DB
}

// Migration é um passo de esquema; Name identifica e nunca muda.
type Migration struct {
	Name string
	SQL  string
}

var memSeq atomic.Int64

var (
	regMu sync.Mutex
	extra []Migration
)

// Register acrescenta migrações de um pacote (chamar no init). Name
// precisa ser único, prefixado com o pacote ("romm/001-saves").
func Register(m ...Migration) {
	regMu.Lock()
	defer regMu.Unlock()
	extra = append(extra, m...)
}

// Open abre (e cria/migra) o banco em dir/arkflix.db. dir == ":memory:"
// abre um banco em memória (testes).
func Open(dir string) (*Store, error) {
	dsn := "file:" + filepath.Join(dir, "arkflix.db") + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	if dir == ":memory:" {
		// um banco por chamada: testes não se enxergam
		dsn = fmt.Sprintf("file:mem%d?mode=memory&cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", memSeq.Add(1))
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close fecha o banco.
func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY, at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	regMu.Lock()
	all := append(append([]Migration{}, core...), extra...)
	regMu.Unlock()
	// as do núcleo primeiro, na ordem; as dos pacotes por nome
	tail := all[len(core):]
	sort.SliceStable(tail, func(i, j int) bool { return tail[i].Name < tail[j].Name })
	for _, m := range all {
		var n int
		if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM migrations WHERE name = ?`, m.Name).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("migração %s: %w", m.Name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO migrations (name) VALUES (?)`, m.Name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

var core = []Migration{{Name: "core/001", SQL: `
CREATE TABLE users (
  rid        INTEGER PRIMARY KEY AUTOINCREMENT, -- id numérico (RomM)
  id         TEXT NOT NULL UNIQUE,              -- 32 hex (Jellyfin)
  name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
  pass_hash  TEXT NOT NULL,                     -- bcrypt
  pass_enc   TEXT NOT NULL,                     -- AES-GCM (token do Subsonic precisa da senha)
  admin      INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_login TEXT
);

CREATE TABLE tokens (
  token      TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind       TEXT NOT NULL,   -- jellyfin | romm | romm-refresh | subsonic | client
  name       TEXT NOT NULL DEFAULT '',
  client     TEXT NOT NULL DEFAULT '',
  device     TEXT NOT NULL DEFAULT '',
  device_id  TEXT NOT NULL DEFAULT '',
  version    TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_used  TEXT,
  expires_at TEXT              -- NULL = não expira
);
CREATE INDEX tokens_user ON tokens(user_id);

CREATE TABLE items (
  rid          INTEGER PRIMARY KEY AUTOINCREMENT, -- id numérico (RomM)
  id           TEXT NOT NULL UNIQUE,              -- 32 hex estável (md5 de tipo+caminho)
  kind         TEXT NOT NULL,                     -- ver library.Kind*
  library      TEXT NOT NULL,                     -- movies | shows | music | games
  parent_id    TEXT,                              -- pasta lógica (temporada, álbum, console)
  series_id    TEXT,
  season_id    TEXT,
  artist_id    TEXT,                              -- artista do álbum/faixa
  album_id     TEXT,
  path         TEXT NOT NULL UNIQUE,              -- caminho absoluto (pasta, pra série/temporada/console/álbum/artista virtual: chave)
  rel_path     TEXT NOT NULL DEFAULT '',          -- relativo à pasta da biblioteca
  name         TEXT NOT NULL,
  sort_name    TEXT NOT NULL DEFAULT '',
  overview     TEXT NOT NULL DEFAULT '',
  year         INTEGER NOT NULL DEFAULT 0,
  premiere     TEXT NOT NULL DEFAULT '',          -- AAAA-MM-DD
  genres       TEXT NOT NULL DEFAULT '[]',        -- JSON
  idx          INTEGER NOT NULL DEFAULT 0,        -- episódio, faixa, temporada
  parent_idx   INTEGER NOT NULL DEFAULT 0,        -- temporada do episódio, disco da faixa
  artist       TEXT NOT NULL DEFAULT '',
  album_artist TEXT NOT NULL DEFAULT '',
  album        TEXT NOT NULL DEFAULT '',
  platform     TEXT NOT NULL DEFAULT '',          -- slug do console (RomM)
  duration     REAL NOT NULL DEFAULT 0,           -- segundos
  size         INTEGER NOT NULL DEFAULT 0,
  mtime        INTEGER NOT NULL DEFAULT 0,        -- unix
  container    TEXT NOT NULL DEFAULT '',          -- mkv, mp4, mp3, flac...
  bitrate      INTEGER NOT NULL DEFAULT 0,        -- bits/s
  width        INTEGER NOT NULL DEFAULT 0,
  height       INTEGER NOT NULL DEFAULT 0,
  streams      TEXT NOT NULL DEFAULT '[]',        -- JSON []Stream (ffprobe)
  image        TEXT NOT NULL DEFAULT '',          -- capa/pôster: arquivo, ou "embedded:<faixa>"
  backdrop     TEXT NOT NULL DEFAULT '',
  logo         TEXT NOT NULL DEFAULT '',
  extra        TEXT NOT NULL DEFAULT '{}',        -- JSON livre por tipo (arquivos da ROM...)
  added_at     TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX items_kind ON items(kind, sort_name);
CREATE INDEX items_parent ON items(parent_id, parent_idx, idx);
CREATE INDEX items_series ON items(series_id);
CREATE INDEX items_artist ON items(artist_id);
CREATE INDEX items_album ON items(album_id);
CREATE INDEX items_added ON items(kind, added_at);

CREATE TABLE userdata (
  user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id     TEXT NOT NULL,          -- sem FK: sobrevive a um rescan que tira e põe o arquivo
  played      INTEGER NOT NULL DEFAULT 0,
  play_count  INTEGER NOT NULL DEFAULT 0,
  position    REAL NOT NULL DEFAULT 0, -- segundos
  last_played TEXT,
  favorite    INTEGER NOT NULL DEFAULT 0,
  favorite_at TEXT,
  rating      INTEGER NOT NULL DEFAULT 0, -- 0..5 (Subsonic); Jellyfin usa só favorito
  PRIMARY KEY (user_id, item_id)
);
CREATE INDEX userdata_resume ON userdata(user_id, last_played);
`}}
