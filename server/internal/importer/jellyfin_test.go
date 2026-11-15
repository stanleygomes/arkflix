package importer

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

const (
	jfStanley = "5E8D7C4A-1B2C-4D3E-8F90-0A1B2C3D4E5F"
	jfJoana   = "A1B2C3D4-E5F6-4711-8899-AABBCCDDEEFF"
	jfVelho   = "00000000-1111-2222-3333-444444444444"
	jfMovie   = "11111111-0000-0000-0000-000000000001"
	jfEp1     = "11111111-0000-0000-0000-000000000002"
	jfSeries  = "11111111-0000-0000-0000-000000000003"
	jfGone    = "11111111-0000-0000-0000-000000000004"
	jfHolder  = "00000000-0000-0000-0000-000000000001" // PlaceholderId: estado de item apagado
)

func jellyfinFixture(t *testing.T) string {
	return fixtureDB(t, t.TempDir(), "jellyfin.db", jellyfinDDL,
		`INSERT INTO Users (Id, Username, Password) VALUES
  ('`+jfStanley+`', 'Stanley', `+q(jellyfinHash(t, "outra-senha"))+`),
  ('`+jfJoana+`', 'joana', `+q(jellyfinHash(t, "segredo"))+`),
  ('`+jfVelho+`', 'velho', NULL)`,
		`INSERT INTO Permissions (UserId, Kind, Value) VALUES
  ('`+jfStanley+`', 0, 1), ('`+jfJoana+`', 0, 0), ('`+jfJoana+`', 2, 0), ('`+jfVelho+`', 2, 1)`,
		`INSERT INTO BaseItems (Id, Type, Name, Path) VALUES
  ('`+jfMovie+`', 'MediaBrowser.Controller.Entities.Movies.Movie', 'O Grande Teste', '/media/Movies/O Grande Teste (2021)/O Grande Teste (2021).webm'),
  ('`+jfEp1+`', 'MediaBrowser.Controller.Entities.TV.Episode', 'O Começo', '/media/Shows/Série de Teste/Season 01/Série de Teste - S01E01 - O Começo.webm'),
  ('`+jfSeries+`', 'MediaBrowser.Controller.Entities.TV.Series', 'Série de Teste', '/media/Shows/Série de Teste'),
  ('`+jfGone+`', 'MediaBrowser.Controller.Entities.Movies.Movie', 'Sumiu', '/media/Movies/Sumiu (1999)/Sumiu (1999).mkv'),
  ('`+jfHolder+`', 'PLACEHOLDER', NULL, NULL)`,
		`INSERT INTO UserData (ItemId, UserId, CustomDataKey, Rating, PlaybackPositionTicks, PlayCount, IsFavorite, LastPlayedDate, Played) VALUES
  ('`+jfMovie+`', '`+jfJoana+`', 'k1', 8.0, 0, 2, 0, '2025-03-01 20:15:00.1234567', 1),
  ('`+jfEp1+`', '`+jfJoana+`', 'k1', NULL, 15000000, 0, 0, '2025-04-02 10:00:00', 0),
  ('`+jfEp1+`', '`+jfJoana+`', 'k2', NULL, 0, 1, 0, '2025-01-01 10:00:00', 0),
  ('`+jfSeries+`', '`+jfJoana+`', 'k1', NULL, 0, 0, 1, NULL, 0),
  ('`+jfGone+`', '`+jfStanley+`', 'k1', NULL, 0, 1, 0, NULL, 1),
  ('`+jfHolder+`', '`+jfStanley+`', 'k1', NULL, 0, 9, 1, NULL, 1),
  ('`+jfMovie+`', '`+jfVelho+`', 'k1', NULL, 0, 1, 0, NULL, 1)`)
}

func TestJellyfin(t *testing.T) {
	ctx := context.Background()
	k, ids := kit(t)
	db := jellyfinFixture(t)

	im, _ := newImporter(k, "/media="+k.Root)
	if err := im.Jellyfin(ctx, db); err != nil {
		t.Fatal(err)
	}
	r := im.Report()
	if r.Count("usuários criados") != 1 || r.Count("usuários juntados") != 1 || r.Count("usuários desativados (pulados)") != 1 ||
		r.Count("estado gravado (movie)") != 1 || r.Count("itens casados") != 4 {
		t.Fatalf("1ª rodada: %+v", r.n)
	}

	// dry-run num arkflix limpo: conta igual e não grava nada
	k2, ids2 := kit(t)
	im, _ = newImporter(k2, "/media="+k2.Root)
	im.Dry = true
	if err := im.Jellyfin(ctx, db); err != nil {
		t.Fatal(err)
	}
	r = im.Report()
	if r.Count("usuários criados") != 1 || r.Count("usuários juntados") != 1 || r.Count("estado gravado (episode)") != 1 {
		t.Fatalf("dry-run: %+v", r.n)
	}
	if users, _ := k2.Auth.Users(ctx); len(users) != 2 {
		t.Fatalf("dry-run criou usuário: %d", len(users))
	}
	if d, _ := k2.Store.UserData(ctx, k2.AdminUser.ID, ids2.MovieBig); d.PlayCount != 0 {
		t.Fatal("dry-run gravou estado")
	}

	// o import de verdade (o 1º, sem dry-run, já rodou em k)
	joana, err := k.Auth.UserByName(ctx, "JOANA")
	if err != nil {
		t.Fatal(err)
	}
	if joana.Admin {
		t.Fatal("joana não é admin")
	}
	if _, err := k.Auth.UserByName(ctx, "velho"); err == nil {
		t.Fatal("desativado não entra")
	}
	// stanley já existia: junta, a senha fica a dele
	if _, err := k.Auth.Login(ctx, testkit.Admin, testkit.AdminPass); err != nil {
		t.Fatal("senha do arkflix devia ficar:", err)
	}
	if _, err := k.Auth.Login(ctx, testkit.Admin, "outra-senha"); err == nil {
		t.Fatal("senha do Jellyfin não devia trocar sem --passwords=overwrite")
	}
	// senha antiga do Jellyfin: entra, regrava, e o Subsonic passa a valer
	sum := md5.Sum([]byte("segredo" + "s"))
	if _, err := k.Auth.SubsonicToken(ctx, "joana", hex.EncodeToString(sum[:]), "s"); err == nil {
		t.Fatal("token do Subsonic antes do 1º login")
	}
	if _, err := k.Auth.Login(ctx, "joana", "segredo"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Auth.SubsonicToken(ctx, "joana", hex.EncodeToString(sum[:]), "s"); err != nil {
		t.Fatal("token do Subsonic depois do login:", err)
	}

	d, _ := k.Store.UserData(ctx, joana.ID, ids.MovieBig)
	want := time.Date(2025, 3, 1, 20, 15, 0, 123000000, time.UTC)
	if !d.Played || d.PlayCount != 2 || !d.LastPlayed.Equal(want) || d.Rating != 4 {
		t.Fatalf("filme: %+v", d)
	}
	// duas linhas pro mesmo episódio (CustomDataKey): a mais recente manda na posição
	d, _ = k.Store.UserData(ctx, joana.ID, ids.Ep1)
	if d.Position != 1.5 || d.PlayCount != 1 || d.Played {
		t.Fatalf("episódio: %+v", d)
	}
	if d, _ = k.Store.UserData(ctx, joana.ID, ids.Series); !d.Favorite {
		t.Fatalf("série favorita: %+v", d)
	}

	// relatório do import de verdade: o filme sumido aparece pra ajustar o --map
	im, _ = newImporter(k, "/media="+k.Root)
	if err := im.Jellyfin(ctx, db); err != nil {
		t.Fatal(err)
	}
	r = im.Report()
	gone := filepath.Join(k.Root, "Movies", "Sumiu (1999)", "Sumiu (1999).mkv")
	if r.Misses("jellyfin") != 1 || !strings.Contains(strings.Join(r.misses["jellyfin"].first, ","), gone) {
		t.Fatalf("sem par: %+v", r.misses)
	}
	// 2ª rodada: nada muda
	if r.Count("usuários criados") != 0 || r.Count("usuários juntados") != 2 || r.Count("estado gravado (movie)") != 0 ||
		r.Count("estado sem mudança") != 3 {
		t.Fatalf("idempotente: %+v", r.n)
	}
	if users, _ := k.Auth.Users(ctx); len(users) != 3 {
		t.Fatalf("%d usuários", len(users))
	}
}

func TestJellyfinRunFlags(t *testing.T) {
	k, _ := kit(t)
	db := jellyfinFixture(t)
	if err := Run(context.Background(), k.Deps, []string{"jellyfin", "--db", db, "--map", "/media=" + k.Root, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if users, _ := k.Auth.Users(context.Background()); len(users) != 2 {
		t.Fatal("dry-run pela linha de comando gravou")
	}
	for _, args := range [][]string{nil, {"plex"}, {"jellyfin"}, {"jellyfin", "--db", db, "--passwords", "talvez"}, {"romm"}} {
		if err := Run(context.Background(), k.Deps, args); err == nil {
			t.Fatalf("%v devia falhar", args)
		}
	}
}

func TestEmptyLibrary(t *testing.T) {
	k := testkit.New(t)
	im, _ := newImporter(k)
	if err := im.Jellyfin(context.Background(), jellyfinFixture(t)); err == nil || !strings.Contains(err.Error(), "scan") {
		t.Fatalf("acervo vazio: %v", err)
	}
}
