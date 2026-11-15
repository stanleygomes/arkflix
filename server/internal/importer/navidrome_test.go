package importer

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"slices"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func navidromeFixture(t *testing.T, key string) string {
	return fixtureDB(t, t.TempDir(), "navidrome.db", navidromeDDL,
		`INSERT INTO user (id, user_name, name, email, password, is_admin, created_at, updated_at) VALUES
  ('u-nina', 'Nina', 'Nina', 'nina@x', `+q(navidromeEnc(t, key, "outra"))+`, 0, '2024-01-01 00:00:00', '2024-01-01 00:00:00'),
  ('u-rui', 'rui', 'Rui', 'rui@x', `+q(navidromeEnc(t, key, "rui123"))+`, 1, '2024-01-01 00:00:00', '2024-01-01 00:00:00')`,
		`INSERT INTO library (id, name, path) VALUES (1, 'Music Library', '/music')`,
		`INSERT INTO artist (id, name) VALUES ('ar1', 'Banda Teste'), ('ar2', 'Artista X'), ('ar9', 'Ninguém')`,
		`INSERT INTO album (id, name, album_artist_id, album_artist) VALUES
  ('al1', 'Disco Teste', 'ar1', 'Banda Teste'), ('al2', 'Avulsas', 'ar2', 'Artista X')`,
		`INSERT INTO media_file (id, path, title, album_id, album_artist_id) VALUES
  ('mf1', 'Banda Teste/Disco Teste/01 - Bip.mp3', 'Bip', 'al1', 'ar1'),
  ('mf2', 'Banda Teste/Disco Teste/02 - Bop.flac', 'Bop', 'al1', 'ar1'),
  ('mf3', 'Avulsas/Artista X - Solta.mp3', 'Solta', 'al2', 'ar2'),
  ('mfX', 'Fantasma/Não Existe.mp3', 'Fantasma', 'al9', 'ar9')`,
		`INSERT INTO annotation (user_id, item_id, item_type, play_count, play_date, rating, starred, starred_at) VALUES
  ('u-rui', 'mf1', 'media_file', 3, '2025-05-01 12:00:00+00:00', 0, 1, '2025-05-02T08:00:00Z'),
  ('u-rui', 'mfX', 'media_file', 9, '2025-05-01 12:00:00+00:00', 0, 1, NULL),
  ('u-rui', 'al1', 'album', 0, NULL, 5, 1, '2025-05-03 08:00:00+00:00'),
  ('u-rui', 'ar2', 'artist', 0, NULL, 4, 0, NULL),
  ('u-nina', 'mf3', 'media_file', 1, '2025-06-01 09:00:00+00:00', 0, 0, NULL)`,
		`INSERT INTO playqueue (id, user_id, current, position, changed_by, items, created_at, updated_at) VALUES
  ('pq1', 'u-rui', 2, 1234, 'Symfonium', 'mf1,mfX,mf2', '2025-07-01 10:00:00+00:00', '2025-07-01 10:00:00+00:00')`,
		`INSERT INTO bookmark (user_id, item_id, item_type, comment, position, changed_by, created_at, updated_at) VALUES
  ('u-rui', 'mf2', 'media_file', 'parei aqui', 1500, 'x', '2025-07-02 10:00:00+00:00', '2025-07-02 10:00:00+00:00')`,
		`INSERT INTO playlist (id, name, owner_id, rules) VALUES
  ('pl1', 'Viagem', 'u-rui', NULL), ('pl2', 'Mais tocadas', 'u-rui', '{"all":[{"gt":{"playCount":1}}]}')`,
		`INSERT INTO playlist_tracks (id, playlist_id, media_file_id) VALUES (1, 'pl1', 'mf3'), (2, 'pl1', 'mfX'), (3, 'pl1', 'mf1')`)
}

func TestNavidrome(t *testing.T) {
	ctx := context.Background()
	k, ids := kit(t)
	db := navidromeFixture(t, "minha-chave")
	im, lists := newImporter(k, "/music="+k.Config.Music)
	if err := im.Navidrome(ctx, db, "minha-chave"); err != nil {
		t.Fatal(err)
	}
	r := im.Report()
	if r.Count("usuários criados") != 1 || r.Count("usuários juntados") != 1 || r.Misses("faixa") != 1 {
		t.Fatalf("%+v %+v", r.n, r.misses)
	}

	// rui nasce com a senha de verdade: token do Subsonic vale na hora
	rui, err := k.Auth.UserByName(ctx, "rui")
	if err != nil || !rui.Admin {
		t.Fatalf("%+v %v", rui, err)
	}
	sum := md5.Sum([]byte("rui123" + "s"))
	if _, err := k.Auth.SubsonicToken(ctx, "rui", hex.EncodeToString(sum[:]), "s"); err != nil {
		t.Fatal(err)
	}
	// nina já existia: fica a senha dela
	if _, err := k.Auth.Login(ctx, testkit.User, testkit.UserPass); err != nil {
		t.Fatal(err)
	}

	d, _ := k.Store.UserData(ctx, rui.ID, ids.TrackBip)
	if !d.Favorite || d.PlayCount != 3 || !d.Played || d.FavoriteAt.IsZero() || d.LastPlayed.IsZero() {
		t.Fatalf("faixa: %+v", d)
	}
	if d, _ = k.Store.UserData(ctx, rui.ID, ids.Album); !d.Favorite || d.Rating != 5 {
		t.Fatalf("álbum: %+v", d)
	}
	if d, _ = k.Store.UserData(ctx, rui.ID, ids.ArtistX); d.Favorite || d.Rating != 4 {
		t.Fatalf("artista: %+v", d)
	}
	if d, _ = k.Store.UserData(ctx, k.PlainUser.ID, ids.TrackSolta); d.PlayCount != 1 {
		t.Fatalf("nina: %+v", d)
	}
	// bookmark: tabela do Subsonic e onde parou
	if d, _ = k.Store.UserData(ctx, rui.ID, ids.TrackBop); d.Position != 1.5 {
		t.Fatalf("posição do marcador: %+v", d)
	}
	var pos int64
	var comment string
	if err := k.Store.DB.QueryRow(`SELECT position, comment FROM subsonic_bookmarks WHERE user_id = ? AND item_id = ?`, rui.ID, ids.TrackBop).
		Scan(&pos, &comment); err != nil || pos != 1500 || comment != "parei aqui" {
		t.Fatalf("marcador: %d %q %v", pos, comment, err)
	}
	// fila: sem a faixa que sumiu, o índice atual acompanha
	var items, by string
	var cur int
	if err := k.Store.DB.QueryRow(`SELECT items, current, position, changed_by FROM subsonic_playqueue WHERE user_id = ?`, rui.ID).
		Scan(&items, &cur, &pos, &by); err != nil {
		t.Fatal(err)
	}
	if items != `["`+ids.TrackBip+`","`+ids.TrackBop+`"]` || cur != 1 || pos != 1234 || by != "Symfonium" {
		t.Fatalf("fila: %s %d %d %s", items, cur, pos, by)
	}
	// playlist normal vai pro ListSink; a inteligente fica de fora
	if got := lists.lists[rui.ID+"/Viagem"]; !slices.Equal(got, []string{ids.TrackSolta, ids.TrackBip}) || len(lists.lists) != 1 {
		t.Fatalf("listas: %+v", lists.lists)
	}
	if r.Count("playlists inteligentes (puladas)") != 1 || r.Count("faixas de playlist sem par") != 1 {
		t.Fatalf("%+v", r.n)
	}

	// de novo: nada muda
	im, _ = newImporter(k, "/music="+k.Config.Music)
	if err := im.Navidrome(ctx, db, "minha-chave"); err != nil {
		t.Fatal(err)
	}
	r = im.Report()
	if r.Count("usuários criados") != 0 || r.Count("filas gravadas") != 0 || r.Count("marcadores gravados") != 0 ||
		r.Count("estado gravado (track)") != 0 || r.Count("estado sem mudança") == 0 {
		t.Fatalf("idempotente: %+v", r.n)
	}
	var n int
	k.Store.DB.QueryRow(`SELECT COUNT(*) FROM subsonic_bookmarks`).Scan(&n)
	if n != 1 {
		t.Fatalf("%d marcadores", n)
	}

	// --passwords=overwrite: a senha do Navidrome troca a de quem já existia
	im, _ = newImporter(k, "/music="+k.Config.Music)
	im.Overwrite = true
	if err := im.Navidrome(ctx, db, "minha-chave"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Auth.Login(ctx, "nina", "outra"); err != nil {
		t.Fatal("overwrite:", err)
	}
}

func TestNavidromeWrongKeyAndDryRun(t *testing.T) {
	ctx := context.Background()
	k, _ := kit(t)
	db := navidromeFixture(t, navidromeDefaultKey)
	im, lists := newImporter(k, "/music="+k.Config.Music)
	im.Dry = true
	if err := im.Navidrome(ctx, db, ""); err != nil { // chave vazia = a padrão
		t.Fatal(err)
	}
	r := im.Report()
	if r.Count("usuários criados") != 1 || r.Count("filas gravadas") != 1 || r.Count("listas") != 1 || len(lists.lists) != 0 {
		t.Fatalf("dry-run: %+v", r.n)
	}
	var n int
	k.Store.DB.QueryRow(`SELECT (SELECT COUNT(*) FROM users) + (SELECT COUNT(*) FROM userdata) + (SELECT COUNT(*) FROM subsonic_playqueue)`).Scan(&n)
	if n != 2 {
		t.Fatalf("dry-run gravou (%d linhas)", n)
	}

	im, _ = newImporter(k, "/music="+k.Config.Music)
	if err := im.Navidrome(ctx, db, "chave-errada"); err != nil {
		t.Fatal(err)
	}
	rui, err := k.Auth.UserByName(ctx, "rui")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := k.Auth.HasPassword(ctx, rui.ID); ok {
		t.Fatal("chave errada: sem senha")
	}
	if len(im.Report().Notes()) == 0 {
		t.Fatal("devia avisar da chave")
	}
}

func TestNavidromePassword(t *testing.T) {
	enc := navidromeEnc(t, navidromeDefaultKey, "abc")
	if p, err := navidromePassword("", enc); err != nil || p != "abc" {
		t.Fatal(p, err)
	}
	if _, err := navidromePassword("x", enc); err == nil {
		t.Fatal("chave errada abriu")
	}
	if _, err := navidromePassword("", "curta"); err == nil {
		t.Fatal("lixo abriu")
	}
}
