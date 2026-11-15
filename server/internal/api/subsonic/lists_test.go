package subsonic

import (
	"context"
	"net/http"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func TestAlbumList(t *testing.T) {
	e := setup(t)
	list := func(kv ...string) []string {
		t.Helper()
		return names(e.ok("getAlbumList2", q(append([]string{"size", "50"}, kv...)...)).AlbumList2.Album)
	}
	eq(t, "nome", list("type", "alphabeticalByName"), []string{"Avulsas", "Disco Teste"})
	eq(t, "artista", list("type", "alphabeticalByArtist"), []string{"Avulsas", "Disco Teste"})
	eq(t, "ano", list("type", "byYear", "fromYear", "2019", "toYear", "2021"), []string{"Disco Teste"})
	eq(t, "ano invertido", list("type", "byYear", "fromYear", "2030", "toYear", "0"), []string{"Disco Teste", "Avulsas"})
	eq(t, "gênero", list("type", "byGenre", "genre", "rock"), []string{"Disco Teste"})
	if n := len(list("type", "random")); n != 2 {
		t.Fatalf("random: %d", n)
	}
	if n := len(list("type", "newest")); n != 2 {
		t.Fatalf("newest: %d", n)
	}
	eq(t, "starred vazio", list("type", "starred"), []string{})
	eq(t, "frequent vazio", list("type", "frequent"), []string{})

	e.ok("star", q("albumId", e.ids.AlbumSolo))
	e.ok("scrobble", q("id", e.ids.TrackBip))
	e.ok("scrobble", q("id", e.ids.TrackBop, "time", "1700000000000"))
	e.ok("setRating", q("id", e.ids.Album, "rating", "4"))
	eq(t, "starred", list("type", "starred"), []string{"Avulsas"})
	eq(t, "frequent", list("type", "frequent"), []string{"Disco Teste"})
	eq(t, "recent", list("type", "recent"), []string{"Disco Teste"})
	eq(t, "highest", list("type", "highest"), []string{"Disco Teste"})

	r := e.ok("getAlbumList2", q("type", "frequent"))
	if al := r.AlbumList2.Album[0]; al.PlayCount != 2 || al.Played == nil || al.UserRating != 4 {
		t.Fatalf("álbum tocado: %+v", al)
	}
	// paginação + x-total-count
	w := e.raw(http.MethodGet, "/rest/getAlbumList2", login(q("f", "json", "type", "alphabeticalByName", "size", "1", "offset", "1")))
	if w.Header().Get("x-total-count") != "2" {
		t.Fatalf("total: %q", w.Header().Get("x-total-count"))
	}
	eq(t, "página", names(decode(t, w).AlbumList2.Album), []string{"Disco Teste"})

	// getAlbumList (pastas)
	r = e.ok("getAlbumList", q("type", "alphabeticalByName"))
	if a := r.AlbumList.Album; len(a) != 2 || !a[0].IsDir || a[0].Title != "Avulsas" || a[1].SongCount != 2 {
		t.Fatalf("albumList: %+v", r.AlbumList)
	}
	e.failed("getAlbumList2", q("type", "nada"), 0)
	e.failed("getAlbumList2", q(), 10)
	e.failed("getAlbumList2", q("type", "byYear", "fromYear", "1"), 10)
	e.failed("getAlbumList2", q("type", "byGenre"), 10)

	// estado é por usuário
	r = e.ok("getAlbumList2", q("type", "starred", "u", testkit.User, "p", testkit.UserPass))
	if len(r.AlbumList2.Album) != 0 {
		t.Fatal("estrela vazou pra outro usuário")
	}
}

func TestSongLists(t *testing.T) {
	e := setup(t)
	r := e.ok("getRandomSongs", q("size", "10"))
	if len(r.RandomSongs.Songs) != 3 {
		t.Fatalf("random: %d", len(r.RandomSongs.Songs))
	}
	r = e.ok("getRandomSongs", q("genre", "Rock", "fromYear", "2020", "toYear", "2020"))
	if len(r.RandomSongs.Songs) != 2 {
		t.Fatalf("random rock: %d", len(r.RandomSongs.Songs))
	}
	r = e.ok("getSongsByGenre", q("genre", "Rock", "count", "1", "offset", "1"))
	eq(t, "byGenre", titles(r.SongsByGenre.Songs), []string{"Bop"})
}

func TestSearch(t *testing.T) {
	e := setup(t)
	search := func(kv ...string) *searchResult3 {
		t.Helper()
		return e.ok("search3", q(kv...)).SearchResult3
	}
	// vazio traz tudo, paginado (sincronização do Symfonium)
	for _, query := range []string{"", `""`} {
		s := search("query", query, "songCount", "2", "artistCount", "500", "albumCount", "500")
		if len(s.Artist) != 2 || len(s.Album) != 2 || len(s.Song) != 2 {
			t.Fatalf("q=%q: %d %d %d", query, len(s.Artist), len(s.Album), len(s.Song))
		}
		first := s.Song
		s = search("query", query, "songCount", "2", "songOffset", "2", "artistCount", "0", "albumCount", "0")
		if len(s.Song) != 1 || len(s.Artist) != 0 || len(s.Album) != 0 || s.Song[0].ID == first[0].ID || s.Song[0].ID == first[1].ID {
			t.Fatalf("página 2: %+v", s)
		}
	}
	s := search("query", "bip")
	if len(s.Song) != 1 || s.Song[0].Title != "Bip" || len(s.Album) != 0 || len(s.Artist) != 0 {
		t.Fatalf("bip: %+v", s)
	}
	// várias palavras, acento e caixa
	s = search("query", "BÁNDA disco*")
	if len(s.Album) != 1 || len(s.Song) != 2 {
		t.Fatalf("banda disco: %+v", s)
	}
	s = search("query", "artista")
	if len(s.Artist) != 1 || s.Artist[0].Name != "Artista X" || len(s.Album) != 1 || len(s.Song) != 1 {
		t.Fatalf("artista: %+v", s)
	}
	r := e.ok("search2", q("query", "solta"))
	if len(r.SearchResult2.Song) != 1 || r.SearchResult2.Song[0].Title != "Solta" {
		t.Fatalf("search2: %+v", r.SearchResult2)
	}
}

func TestStar(t *testing.T) {
	e := setup(t)
	e.ok("star", q("id", e.ids.TrackBip, "artistId", e.ids.Artist, "albumId", e.ids.Album, "id", "desconhecido"))
	r := e.ok("getStarred2", q())
	s := r.Starred2
	if len(s.Artist) != 1 || len(s.Album) != 1 || len(s.Song) != 1 || s.Song[0].Starred == nil || s.Album[0].Starred == nil {
		t.Fatalf("starred2: %+v", s)
	}
	r = e.ok("getStarred", q())
	if len(r.Starred.Artist) != 1 || len(r.Starred.Album) != 1 || !r.Starred.Album[0].IsDir {
		t.Fatalf("starred: %+v", r.Starred)
	}
	// estrela = favorito do Jellyfin
	ud, err := e.k.Store.UserData(context.Background(), e.k.AdminUser.ID, e.ids.TrackBip)
	if err != nil || !ud.Favorite || ud.FavoriteAt.IsZero() {
		t.Fatalf("userdata: %+v %v", ud, err)
	}
	// favorito marcado pelo Jellyfin aparece aqui
	e.k.Store.SetFavorite(context.Background(), e.k.AdminUser.ID, e.ids.TrackSolta, true)
	if r := e.ok("getSong", q("id", e.ids.TrackSolta)); r.Song.Starred == nil {
		t.Fatal("favorito do Jellyfin não veio")
	}
	e.ok("unstar", q("id", e.ids.TrackBip, "id", e.ids.TrackSolta, "artistId", e.ids.Artist, "albumId", e.ids.Album))
	if r := e.ok("getStarred2", q()); len(r.Starred2.Song)+len(r.Starred2.Album)+len(r.Starred2.Artist) != 0 {
		t.Fatalf("unstar: %+v", r.Starred2)
	}
	e.failed("star", q(), 10)

	e.ok("setRating", q("id", e.ids.TrackBip, "rating", "5"))
	if r := e.ok("getSong", q("id", e.ids.TrackBip)); r.Song.UserRating != 5 {
		t.Fatalf("rating: %d", r.Song.UserRating)
	}
	e.failed("setRating", q("id", e.ids.TrackBip, "rating", "6"), 0)
	e.failed("setRating", q("id", "nada", "rating", "1"), 70)
	e.failed("setRating", q("id", e.ids.TrackBip), 10)
}

func TestScrobble(t *testing.T) {
	e := setup(t)
	e.ok("scrobble", q("id", e.ids.TrackBip))
	e.ok("scrobble", q("id", e.ids.TrackBip, "submission", "true", "time", "1700000000000"))
	r := e.ok("getSong", q("id", e.ids.TrackBip))
	if r.Song.PlayCount != 2 || r.Song.Played == nil {
		t.Fatalf("scrobble: %+v", r.Song)
	}
	ud, _ := e.k.Store.UserData(context.Background(), e.k.AdminUser.ID, e.ids.TrackBip)
	if ud.PlayCount != 2 || !ud.Played || ud.LastPlayed.IsZero() {
		t.Fatalf("userdata: %+v", ud)
	}
	e.failed("scrobble", q(), 10)
	e.failed("scrobble", q("id", e.ids.TrackBip, "id", e.ids.TrackBop, "time", "1"), 0)

	// tocando agora
	e.ok("scrobble", q("id", e.ids.TrackBop, "submission", "false", "u", testkit.User, "p", testkit.UserPass, "c", "Symfonium"))
	r = e.ok("getNowPlaying", q())
	if n := r.NowPlaying; len(n.Entry) != 1 || n.Entry[0].Title != "Bop" || n.Entry[0].Username != testkit.User || n.Entry[0].PlayerName != "Symfonium" {
		t.Fatalf("nowPlaying: %+v", r.NowPlaying)
	}
	if r := e.ok("getSong", q("id", e.ids.TrackBop)); r.Song.PlayCount != 0 {
		t.Fatal("submission=false contou execução")
	}
}
