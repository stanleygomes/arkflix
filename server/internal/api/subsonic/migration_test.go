package subsonic

import (
	"context"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// TestPlaylistsMigration: um banco de antes da 004 (playlists nas tabelas
// do Subsonic) abre com tudo em internal/lists, mesmo id, mesma ordem.
func TestPlaylistsMigration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// volta o banco pra antes da 004
	_, err = st.DB.ExecContext(ctx, `
INSERT INTO users (id, name, pass_hash, pass_enc, admin) VALUES ('u1', 'ana', 'x', 'x', 1), ('u2', 'bia', 'x', 'x', 0);
CREATE TABLE subsonic_playlists (
  id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, name TEXT NOT NULL,
  comment TEXT NOT NULL DEFAULT '', public INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, changed_at TEXT NOT NULL);
CREATE TABLE subsonic_playlist_items (
  playlist_id TEXT NOT NULL REFERENCES subsonic_playlists(id) ON DELETE CASCADE, pos INTEGER NOT NULL,
  item_id TEXT NOT NULL, PRIMARY KEY (playlist_id, pos));
INSERT INTO subsonic_playlists VALUES
  ('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'u1', 'Rock', 'alto', 1, '2024-01-01T00:00:00Z', '2024-02-01T00:00:00Z'),
  ('bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', 'u2', 'Vazia', '', 0, '2024-03-01T00:00:00Z', '2024-03-01T00:00:00Z');
INSERT INTO subsonic_playlist_items VALUES
  ('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 0, 'f2'), ('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 1, 'f1'),
  ('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 2, 'f2');
DELETE FROM migrations WHERE name = 'subsonic/004-lists';`)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	st, err = store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ls := lists.New(st)
	rock, err := ls.Get(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if rock.Name != "Rock" || rock.Comment != "alto" || !rock.Public || rock.UserID != "u1" || rock.Owner != "ana" ||
		rock.Created.Year() != 2024 || rock.Updated.Month() != 2 {
		t.Fatalf("rock: %+v", rock)
	}
	es, _ := ls.Entries(ctx, rock.ID)
	var got []string
	for _, e := range es {
		got = append(got, e.ItemID)
	}
	if strings.Join(got, ",") != "f2,f1,f2" || es[0].ID == es[2].ID {
		t.Fatalf("entradas: %+v", es)
	}
	if own, _ := ls.Own(ctx, "u2"); len(own) != 1 || own[0].Name != "Vazia" {
		t.Fatalf("vazia: %+v", own)
	}
	var n int
	st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'subsonic_playlist%'`).Scan(&n)
	if n != 0 {
		t.Fatal("tabelas velhas ficaram")
	}
}
