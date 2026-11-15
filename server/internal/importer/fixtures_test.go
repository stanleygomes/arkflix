package importer

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// Os esquemas abaixo são os dos bancos de verdade (migrações do EF Core do
// Jellyfin, goose do Navidrome, alembic do RomM), cortados nas tabelas e
// colunas que o import lê, mais algumas vizinhas pra não ficar ideal demais.

const jellyfinDDL = `
CREATE TABLE "Users" (
  "Id" TEXT NOT NULL CONSTRAINT "PK_Users" PRIMARY KEY,
  "Username" TEXT NOT NULL,
  "NormalizedUsername" TEXT NULL,
  "Password" TEXT NULL,
  "MustUpdatePassword" INTEGER NOT NULL DEFAULT 0,
  "AuthenticationProviderId" TEXT NOT NULL DEFAULT '',
  "PasswordResetProviderId" TEXT NOT NULL DEFAULT '',
  "InvalidLoginAttemptCount" INTEGER NOT NULL DEFAULT 0,
  "LastActivityDate" TEXT NULL,
  "LastLoginDate" TEXT NULL,
  "InternalId" INTEGER NOT NULL DEFAULT 0,
  "RowVersion" INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE "Permissions" (
  "Id" INTEGER NOT NULL CONSTRAINT "PK_Permissions" PRIMARY KEY AUTOINCREMENT,
  "UserId" TEXT NULL,
  "Kind" INTEGER NOT NULL,
  "Value" INTEGER NOT NULL,
  "RowVersion" INTEGER NOT NULL DEFAULT 0,
  CONSTRAINT "FK_Permissions_Users_UserId" FOREIGN KEY ("UserId") REFERENCES "Users" ("Id") ON DELETE CASCADE
);
CREATE UNIQUE INDEX "IX_Permissions_UserId_Kind" ON "Permissions" ("UserId", "Kind");
CREATE TABLE "BaseItems" (
  "Id" TEXT NOT NULL CONSTRAINT "PK_BaseItems" PRIMARY KEY,
  "Type" TEXT NOT NULL,
  "Name" TEXT NULL,
  "Path" TEXT NULL,
  "ParentId" TEXT NULL,
  "IsFolder" INTEGER NOT NULL DEFAULT 0,
  "IsVirtualItem" INTEGER NOT NULL DEFAULT 0,
  "DateCreated" TEXT NULL
);
CREATE TABLE "UserData" (
  "ItemId" TEXT NOT NULL,
  "UserId" TEXT NOT NULL,
  "CustomDataKey" TEXT NOT NULL,
  "Rating" REAL NULL,
  "PlaybackPositionTicks" INTEGER NOT NULL,
  "PlayCount" INTEGER NOT NULL,
  "IsFavorite" INTEGER NOT NULL,
  "LastPlayedDate" TEXT NULL,
  "Played" INTEGER NOT NULL,
  "AudioStreamIndex" INTEGER NULL,
  "SubtitleStreamIndex" INTEGER NULL,
  "Likes" INTEGER NULL,
  "RetentionDate" TEXT NULL,
  CONSTRAINT "PK_UserData" PRIMARY KEY ("ItemId", "UserId", "CustomDataKey"),
  CONSTRAINT "FK_UserData_BaseItems_ItemId" FOREIGN KEY ("ItemId") REFERENCES "BaseItems" ("Id") ON DELETE CASCADE,
  CONSTRAINT "FK_UserData_Users_UserId" FOREIGN KEY ("UserId") REFERENCES "Users" ("Id") ON DELETE CASCADE
);`

const navidromeDDL = `
create table user (
  id varchar(255) not null primary key,
  user_name varchar(255) default '' not null unique,
  name varchar(255) default '' not null,
  email varchar(255) default '' not null,
  password varchar(255) default '' not null,
  is_admin bool default FALSE not null,
  last_login_at datetime,
  last_access_at datetime,
  created_at datetime not null,
  updated_at datetime not null
);
create table library (
  id integer primary key autoincrement,
  name text not null unique,
  path text not null unique,
  remote_path text null default '',
  last_scan_at datetime not null default '0000-00-00 00:00:00',
  updated_at datetime not null default current_timestamp,
  created_at datetime not null default current_timestamp
);
create table artist (
  id varchar(255) not null primary key,
  name varchar(255) default '' not null
);
create table album (
  id varchar(255) not null primary key,
  name varchar(255) default '' not null,
  album_artist_id varchar(255) default '' not null,
  album_artist varchar(255) default '' not null,
  library_id integer not null default 1
);
create table media_file (
  id varchar(255) not null primary key,
  path varchar(255) default '' not null,
  title varchar(255) default '' not null,
  album varchar(255) default '' not null,
  artist varchar(255) default '' not null,
  album_artist varchar(255) default '' not null,
  album_id varchar(255) default '' not null,
  artist_id varchar(255) default '' not null,
  album_artist_id varchar(255) default '' not null,
  library_id integer not null default 1,
  missing boolean default false not null
);
create table annotation (
  user_id varchar(255) not null references user(id) on delete cascade on update cascade,
  item_id varchar(255) default '' not null,
  item_type varchar(255) default '' not null,
  play_count integer default 0,
  play_date datetime,
  rating integer default 0,
  starred bool default FALSE not null,
  starred_at datetime,
  rated_at datetime,
  unique (user_id, item_id, item_type)
);
create table playqueue (
  id varchar(255) not null,
  user_id varchar(255) not null references user(id) on update cascade on delete cascade,
  current integer not null default 0,
  changed_by varchar(255),
  items varchar(255),
  created_at datetime,
  updated_at datetime,
  position integer
);
create table bookmark (
  user_id varchar(255) not null references user on update cascade on delete cascade,
  item_id varchar(255) not null,
  item_type varchar(255) not null,
  comment varchar(255),
  position integer,
  changed_by varchar(255),
  created_at datetime,
  updated_at datetime,
  constraint bookmark_pk unique (user_id, item_id, item_type)
);
create table playlist (
  id varchar(255) not null primary key,
  name varchar(255) default '' not null,
  comment varchar(255) default '' not null,
  duration real default 0 not null,
  song_count integer default 0 not null,
  public bool default FALSE not null,
  created_at datetime,
  updated_at datetime,
  path string default '' not null,
  sync bool default false not null,
  size integer default 0 not null,
  rules varchar,
  evaluated_at datetime,
  owner_id varchar(255) not null
);
create table playlist_tracks (
  id integer default 0 not null,
  playlist_id varchar(255) not null references playlist(id) on update cascade on delete cascade,
  media_file_id varchar(255) not null
);
create unique index playlist_tracks_pos on playlist_tracks (playlist_id, id);`

// rommDDL é o esquema do RomM 5.x (alembic) escrito em SQL que o SQLite
// aceita: as consultas do import são as mesmas no MariaDB.
const rommDDL = `
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username VARCHAR(255) NOT NULL UNIQUE,
  hashed_password VARCHAR(255),
  email VARCHAR(255),
  enabled BOOLEAN NOT NULL DEFAULT 1,
  role VARCHAR(10) NOT NULL,
  avatar_path VARCHAR(255) NOT NULL DEFAULT '',
  last_login TIMESTAMP,
  last_active TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE platforms (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug VARCHAR(100) NOT NULL,
  fs_slug VARCHAR(100) NOT NULL,
  name VARCHAR(400) NOT NULL
);
CREATE TABLE roms (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  fs_name VARCHAR(450) NOT NULL,
  fs_name_no_tags VARCHAR(450) NOT NULL DEFAULT '',
  fs_name_no_ext VARCHAR(450) NOT NULL DEFAULT '',
  fs_extension VARCHAR(100) NOT NULL DEFAULT '',
  fs_path VARCHAR(1000) NOT NULL,
  fs_size_bytes BIGINT NOT NULL DEFAULT 0,
  name VARCHAR(350),
  platform_id INTEGER NOT NULL REFERENCES platforms(id) ON DELETE CASCADE
);
CREATE TABLE saves (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file_name VARCHAR(450) NOT NULL,
  file_name_no_tags VARCHAR(450) NOT NULL DEFAULT '',
  file_name_no_ext VARCHAR(450) NOT NULL DEFAULT '',
  file_extension VARCHAR(100) NOT NULL DEFAULT '',
  file_path VARCHAR(1000) NOT NULL,
  file_size_bytes BIGINT NOT NULL DEFAULT 0,
  missing_from_fs BOOLEAN NOT NULL DEFAULT 0,
  rom_id INTEGER REFERENCES roms(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  emulator VARCHAR(50),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE states (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file_name VARCHAR(450) NOT NULL,
  file_path VARCHAR(1000) NOT NULL,
  file_size_bytes BIGINT NOT NULL DEFAULT 0,
  rom_id INTEGER REFERENCES roms(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  emulator VARCHAR(50),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE screenshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file_name VARCHAR(450) NOT NULL,
  file_path VARCHAR(1000) NOT NULL,
  file_size_bytes BIGINT NOT NULL DEFAULT 0,
  rom_id INTEGER REFERENCES roms(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE rom_user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_raw_markdown TEXT NOT NULL DEFAULT '',
  note_is_public BOOLEAN NOT NULL DEFAULT 0,
  is_main_sibling BOOLEAN NOT NULL DEFAULT 0,
  last_played TIMESTAMP,
  backlogged BOOLEAN NOT NULL DEFAULT 0,
  now_playing BOOLEAN NOT NULL DEFAULT 0,
  hidden BOOLEAN NOT NULL DEFAULT 0,
  rating INTEGER NOT NULL DEFAULT 0,
  difficulty INTEGER NOT NULL DEFAULT 0,
  completion INTEGER NOT NULL DEFAULT 0,
  status VARCHAR(13),
  rom_id INTEGER NOT NULL REFERENCES roms(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (rom_id, user_id)
);
CREATE TABLE collections (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name VARCHAR(400) NOT NULL,
  description TEXT,
  is_public BOOLEAN NOT NULL DEFAULT 0,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE collections_roms (
  collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  rom_id INTEGER NOT NULL REFERENCES roms(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (collection_id, rom_id)
);`

// fixtureDB cria um SQLite em dir/name com o esquema e as linhas.
func fixtureDB(t *testing.T, dir, name, ddl string, rows ...string) string {
	t.Helper()
	file := filepath.Join(dir, name)
	db, err := sql.Open("sqlite", "file:"+file)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range append([]string{ddl}, rows...) {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	return file
}

// kit é o arkflix de teste com o acervo semeado.
func kit(t *testing.T) (*testkit.Kit, testkit.SeedIDs) {
	t.Helper()
	k := testkit.New(t)
	return k, k.Seed(t)
}

// newImporter monta um Importer com as listas capturadas.
func newImporter(k *testkit.Kit, maps ...string) (*Importer, *captured) {
	c := &captured{}
	im := &Importer{D: k.Deps, Lists: c, Show: 10}
	for _, m := range maps {
		if err := im.Maps.Set(m); err != nil {
			panic(err)
		}
	}
	return im, c
}

// captured guarda as listas recebidas (junta por usuário + nome).
type captured struct {
	mu    sync.Mutex
	lists map[string][]string
}

func (c *captured) AddList(_ context.Context, userID, name string, ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lists == nil {
		c.lists = map[string][]string{}
	}
	c.lists[userID+"/"+name] = ids
	return nil
}

// jellyfinHash monta o hash como o Jellyfin grava (hex maiúsculo).
func jellyfinHash(t *testing.T, pass string) string {
	t.Helper()
	salt := []byte("sal-do-jellyfin!")
	key, err := pbkdf2.Key(sha512.New, pass, salt, 1000, 64)
	if err != nil {
		t.Fatal(err)
	}
	return "$PBKDF2-SHA512$iterations=1000$" + strings.ToUpper(hex.EncodeToString(salt)) + "$" + strings.ToUpper(hex.EncodeToString(key))
}

// navidromeEnc cifra a senha como o Navidrome (AES-GCM, sha256 da chave).
func navidromeEnc(t *testing.T, key, pass string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(key))
	block, _ := aes.NewCipher(sum[:])
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	copy(nonce, "nonce-fixo!!")
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(pass), nil))
}

func q(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
