package importer

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// rommFixture monta o banco do RomM (SQLite com as tabelas do MariaDB) e a
// pasta de assets como o RomM grava: users/<pasta do usuário>/<tipo>/<fs_slug>/<rom>[/<emulador>]/arquivo.
func rommFixture(t *testing.T) (src SQLRomM, assets string) {
	t.Helper()
	dir := t.TempDir()
	assets = filepath.Join(dir, "assets")
	files := map[string]string{
		"users/6c656f/saves/snes/10/snes9x/Super Teste (USA).srm":    "save do leo",
		"users/6c656f/states/snes/10/snes9x/Super Teste (USA).state": "state do leo",
		"users/6c656f/screenshots/psx/11/Jogo Disco (USA).png":       "png do leo",
		"users/6c656f/saves/snes/12/Fantasma.srm":                    "rom sumiu",
	}
	for rel, body := range files {
		p := filepath.Join(assets, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	leo, _ := bcrypt.GenerateFromPassword([]byte("leo123"), bcrypt.MinCost)
	other, _ := bcrypt.GenerateFromPassword([]byte("outra"), bcrypt.MinCost)
	file := fixtureDB(t, dir, "romm.db", rommDDL,
		`INSERT INTO users (id, username, hashed_password, enabled, role) VALUES
  (1, 'stanley', `+q(string(other))+`, 1, 'ADMIN'), (2, 'leo', `+q(string(leo))+`, 1, 'VIEWER'), (3, 'fora', NULL, 0, 'VIEWER')`,
		`INSERT INTO platforms (id, slug, fs_slug, name) VALUES (1, 'snes', 'snes', 'SNES'), (2, 'ps', 'psx', 'PlayStation')`,
		`INSERT INTO roms (id, fs_name, fs_path, platform_id) VALUES
  (10, 'Super Teste (USA).sfc', 'roms/snes', 1), (11, 'Jogo Disco (USA)', 'outro/lugar/psx', 2), (12, 'Fantasma.sfc', 'roms/snes', 1)`,
		`INSERT INTO saves (file_name, file_path, rom_id, user_id, emulator, created_at, updated_at) VALUES
  ('Super Teste (USA).srm', 'users/6c656f/saves/snes/10/snes9x', 10, 2, 'snes9x', '2025-02-01 10:00:00', '2025-02-03 10:00:00'),
  ('Fantasma.srm', 'users/6c656f/saves/snes/12', 12, 2, NULL, '2025-02-01 10:00:00', '2025-02-01 10:00:00'),
  ('Sumido.srm', 'users/6c656f/saves/snes/10', 10, 2, NULL, '2025-02-01 10:00:00', '2025-02-01 10:00:00')`,
		`INSERT INTO states (file_name, file_path, rom_id, user_id, emulator) VALUES
  ('Super Teste (USA).state', 'users/6c656f/states/snes/10/snes9x', 10, 2, 'snes9x')`,
		`INSERT INTO screenshots (file_name, file_path, rom_id, user_id) VALUES
  ('Jogo Disco (USA).png', 'users/6c656f/screenshots/psx/11', 11, 2)`,
		`INSERT INTO rom_user (rom_id, user_id, last_played, backlogged, rating, status) VALUES
  (10, 2, '2025-02-03 11:00:00', 1, 8, 'FINISHED'), (11, 2, NULL, 0, 0, NULL)`,
		`INSERT INTO collections (id, name, user_id) VALUES (1, 'Favoritos da infância', 2)`,
		`INSERT INTO collections_roms (collection_id, rom_id) VALUES (1, 11), (1, 10), (1, 12)`)
	db, err := sql.Open("sqlite", "file:"+file+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return SQLRomM{Q: db}, assets
}

func TestRomM(t *testing.T) {
	ctx := context.Background()
	k, ids := kit(t)
	src, assets := rommFixture(t)
	opt := RomMOptions{Assets: assets}
	im, lists := newImporter(k, "/romm/library="+k.Config.Games)
	if err := im.RomM(ctx, src, opt); err != nil {
		t.Fatal(err)
	}
	r := im.Report()
	// 10 casa pelo caminho; 11 (fs_path estranho) pelo console + nome; 12 não existe
	if r.Count("roms casadas") != 2 || r.Misses("rom") != 1 || r.Count("usuários criados") != 1 || r.Count("usuários desativados (pulados)") != 1 {
		t.Fatalf("%+v %+v", r.n, r.misses)
	}
	if r.Count("saves copiados") != 1 || r.Count("states copiados") != 1 || r.Count("screenshots copiados") != 1 ||
		r.Misses("saves (arquivo sumiu)") != 1 || r.Count("saves de rom sem par") != 1 {
		t.Fatalf("%+v %+v", r.n, r.misses)
	}

	// bcrypt do RomM serve como está; a cópia pro Subsonic nasce no 1º login
	leo, err := k.Auth.UserByName(ctx, "leo")
	if err != nil || leo.Admin {
		t.Fatalf("%+v %v", leo, err)
	}
	sum := md5.Sum([]byte("leo123" + "s"))
	if _, err := k.Auth.SubsonicToken(ctx, "leo", hex.EncodeToString(sum[:]), "s"); err == nil {
		t.Fatal("antes do login não tem como conferir o token")
	}
	if _, err := k.Auth.Login(ctx, "leo", "leo123"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Auth.SubsonicToken(ctx, "leo", hex.EncodeToString(sum[:]), "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Auth.Login(ctx, testkit.Admin, testkit.AdminPass); err != nil {
		t.Fatal("stanley mantém a senha:", err)
	}

	// o save foi pro layout do arkflix, com a linha
	rom, _ := k.Store.Item(ctx, ids.RomSuper)
	rel := filepath.Join("romm", "saves", strconv.FormatInt(leo.RID, 10), "snes", strconv.FormatInt(rom.RID, 10), "snes9x")
	body, err := os.ReadFile(filepath.Join(k.Config.Data, rel, "Super Teste (USA).srm"))
	if err != nil || string(body) != "save do leo" {
		t.Fatalf("%q %v", body, err)
	}
	var fp, created, updated string
	var size int64
	if err := k.Store.DB.QueryRow(`SELECT file_path, size, created_at, updated_at FROM romm_assets WHERE kind = 'saves' AND user_id = ?`, leo.ID).
		Scan(&fp, &size, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if fp != filepath.ToSlash(rel) || size != int64(len("save do leo")) || created != "2025-02-01T10:00:00Z" || updated != "2025-02-03T10:00:00Z" {
		t.Fatalf("%s %d %s %s", fp, size, created, updated)
	}
	// screenshot sem emulador
	disco, _ := k.Store.Item(ctx, ids.RomDisco)
	shot := filepath.Join(k.Config.Data, "romm", "screenshots", strconv.FormatInt(leo.RID, 10), "psx", strconv.FormatInt(disco.RID, 10), "Jogo Disco (USA).png")
	if _, err := os.Stat(shot); err != nil {
		t.Fatal(err)
	}

	// props e last_played
	var status string
	var backlogged bool
	var rating int
	if err := k.Store.DB.QueryRow(`SELECT status, backlogged, rating FROM romm_rom_users WHERE user_id = ? AND rom_rid = ?`, leo.ID, rom.RID).
		Scan(&status, &backlogged, &rating); err != nil || status != "finished" || !backlogged || rating != 8 {
		t.Fatalf("props: %s %v %d %v", status, backlogged, rating, err)
	}
	if d, _ := k.Store.UserData(ctx, leo.ID, ids.RomSuper); d.LastPlayed.IsZero() {
		t.Fatal("last_played")
	}
	// coleção: ids do acervo, sem a rom que não existe
	if got := lists.lists[leo.ID+"/Favoritos da infância"]; !slices.Equal(got, []string{ids.RomSuper, ids.RomDisco}) {
		t.Fatalf("coleção: %v", got)
	}

	// de novo: nada copiado, nada duplicado
	im, _ = newImporter(k, "/romm/library="+k.Config.Games)
	if err := im.RomM(ctx, src, opt); err != nil {
		t.Fatal(err)
	}
	r = im.Report()
	if r.Count("saves copiados") != 0 || r.Count("saves sem mudança") != 1 || r.Count("props gravadas") != 0 || r.Count("usuários criados") != 0 {
		t.Fatalf("idempotente: %+v", r.n)
	}
	var n int
	k.Store.DB.QueryRow(`SELECT COUNT(*) FROM romm_assets`).Scan(&n)
	if n != 3 {
		t.Fatalf("%d assets", n)
	}

	// save mudou no RomM: copia de novo por cima
	p := filepath.Join(assets, "users/6c656f/saves/snes/10/snes9x/Super Teste (USA).srm")
	os.WriteFile(p, []byte("save novo"), 0o644)
	im, _ = newImporter(k, "/romm/library="+k.Config.Games)
	if err := im.RomM(ctx, src, opt); err != nil {
		t.Fatal(err)
	}
	if im.Report().Count("saves copiados") != 1 {
		t.Fatalf("%+v", im.Report().n)
	}
	body, _ = os.ReadFile(filepath.Join(k.Config.Data, rel, "Super Teste (USA).srm"))
	k.Store.DB.QueryRow(`SELECT COUNT(*) FROM romm_assets`).Scan(&n)
	if string(body) != "save novo" || n != 3 {
		t.Fatalf("%q %d", body, n)
	}
}

func TestRomMDryRunAndMissingAssets(t *testing.T) {
	ctx := context.Background()
	k, _ := kit(t)
	src, assets := rommFixture(t)
	im, _ := newImporter(k, "/romm/library="+k.Config.Games)
	im.Dry = true
	if err := im.RomM(ctx, src, RomMOptions{Assets: assets}); err != nil {
		t.Fatal(err)
	}
	if r := im.Report(); r.Count("saves copiados") != 1 || r.Count("props gravadas") != 2 {
		t.Fatalf("%+v", r.n)
	}
	if _, err := os.Stat(filepath.Join(k.Config.Data, "romm")); !os.IsNotExist(err) {
		t.Fatal("dry-run copiou arquivo")
	}
	var n int
	k.Store.DB.QueryRow(`SELECT (SELECT COUNT(*) FROM users) + (SELECT COUNT(*) FROM romm_assets) + (SELECT COUNT(*) FROM romm_rom_users)`).Scan(&n)
	if n != 2 {
		t.Fatalf("dry-run gravou (%d)", n)
	}
	im, _ = newImporter(k, "/romm/library="+k.Config.Games)
	if err := im.RomM(ctx, src, RomMOptions{}); err == nil {
		t.Fatal("sem --assets devia falhar")
	}
}
