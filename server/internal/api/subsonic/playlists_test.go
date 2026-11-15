package subsonic

import (
	"context"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func TestPlaylists(t *testing.T) {
	e := setup(t)
	r := e.ok("createPlaylist", q("name", "Minha", "songId", e.ids.TrackBip, "songId", e.ids.TrackBop))
	pl := r.Playlist
	if pl == nil || pl.Name != "Minha" || pl.SongCount != 2 || pl.Duration != 4 || pl.Owner != testkit.Admin || pl.Public || pl.Readonly ||
		pl.CoverArt != e.ids.TrackBip || pl.Created.IsZero() {
		t.Fatalf("create: %+v", pl)
	}
	eq(t, "entradas", titles(pl.Entry), []string{"Bip", "Bop"})
	id := pl.ID

	e.ok("updatePlaylist", q("playlistId", id, "name", "Outra", "comment", "oi", "public", "true",
		"songIndexToRemove", "0", "songIdToAdd", e.ids.TrackSolta, "songIdToAdd", e.ids.TrackBip))
	r = e.ok("getPlaylist", q("id", id))
	if p := r.Playlist; p.Name != "Outra" || p.Comment != "oi" || !p.Public {
		t.Fatalf("update: %+v", p)
	}
	eq(t, "depois do update", titles(r.Playlist.Entry), []string{"Bop", "Solta", "Bip"})

	// createPlaylist com playlistId troca as faixas
	r = e.ok("createPlaylist", q("playlistId", id, "songId", e.ids.TrackSolta))
	eq(t, "troca", titles(r.Playlist.Entry), []string{"Solta"})

	// outro usuário vê a pública, mas não mexe
	nina := []string{"u", testkit.User, "p", testkit.UserPass}
	r = e.ok("getPlaylists", q(nina...))
	if len(r.Playlists.Playlist) != 1 || !r.Playlists.Playlist[0].Readonly {
		t.Fatalf("pública: %+v", r.Playlists)
	}
	e.failed("deletePlaylist", q(append(nina, "id", id)...), 50)
	e.failed("updatePlaylist", q(append(nina, "playlistId", id, "name", "x")...), 50)

	// privada some pros outros
	e.ok("updatePlaylist", q("playlistId", id, "public", "false"))
	e.failed("getPlaylist", q(append(nina, "id", id)...), 70)
	if r := e.ok("getPlaylists", q(nina...)); len(r.Playlists.Playlist) != 0 {
		t.Fatal("privada apareceu")
	}

	// a da nina é dela
	r = e.ok("createPlaylist", q(append(nina, "name", "Da Nina")...))
	if r.Playlist.SongCount != 0 || r.Playlist.Owner != testkit.User {
		t.Fatalf("nina: %+v", r.Playlist)
	}
	if r := e.ok("getPlaylists", q()); len(r.Playlists.Playlist) != 1 {
		t.Fatalf("admin vê só a dele: %+v", r.Playlists)
	}

	e.ok("deletePlaylist", q("id", id))
	e.failed("getPlaylist", q("id", id), 70)
	e.failed("createPlaylist", q(), 10)
	e.failed("deletePlaylist", q("id", "nada"), 70)
}

func TestPlayQueue(t *testing.T) {
	e := setup(t)
	r := e.ok("getPlayQueue", q())
	if pq := r.PlayQueue; pq == nil || pq.Username != testkit.Admin || len(pq.Entry) != 0 {
		t.Fatalf("vazia: %+v", r.PlayQueue)
	}
	e.ok("savePlayQueue", q("id", e.ids.TrackBip, "id", e.ids.TrackBop, "id", e.ids.TrackBip, "current", e.ids.TrackBop, "position", "1234", "c", "Feishin"))
	r = e.ok("getPlayQueue", q())
	pq := r.PlayQueue
	if pq.Current != e.ids.TrackBop || pq.Position != 1234 || pq.ChangedBy != "Feishin" || pq.Changed.IsZero() {
		t.Fatalf("fila: %+v", pq)
	}
	eq(t, "fila", titles(pq.Entry), []string{"Bip", "Bop", "Bip"})

	r = e.ok("getPlayQueueByIndex", q())
	if p := r.PlayQueueByIndex; p.CurrentIndex == nil || *p.CurrentIndex != 1 || len(p.Entry) != 3 {
		t.Fatalf("byIndex: %+v", p)
	}
	e.ok("savePlayQueueByIndex", q("id", e.ids.TrackSolta, "id", e.ids.TrackBip, "currentIndex", "1", "position", "5"))
	r = e.ok("getPlayQueueByIndex", q())
	if p := r.PlayQueueByIndex; *p.CurrentIndex != 1 || p.Position != 5 || p.Entry[1].Title != "Bip" {
		t.Fatalf("byIndex 2: %+v", p)
	}
	if r := e.ok("getPlayQueue", q()); r.PlayQueue.Current != e.ids.TrackBip {
		t.Fatalf("current: %+v", r.PlayQueue)
	}
	e.failed("savePlayQueueByIndex", q("id", e.ids.TrackBip, "currentIndex", "3"), 10)
	// fila é por usuário
	if r := e.ok("getPlayQueue", q("u", testkit.User, "p", testkit.UserPass)); len(r.PlayQueue.Entry) != 0 {
		t.Fatal("fila vazou")
	}
	// salvar vazio limpa
	e.ok("savePlayQueue", q())
	if r := e.ok("getPlayQueue", q()); len(r.PlayQueue.Entry) != 0 {
		t.Fatal("não limpou")
	}
}

func TestBookmarks(t *testing.T) {
	e := setup(t)
	e.ok("createBookmark", q("id", e.ids.TrackBop, "position", "1500", "comment", "parei aqui"))
	r := e.ok("getBookmarks", q())
	b := r.Bookmarks.Bookmark
	if len(b) != 1 || b[0].Position != 1500 || b[0].Comment != "parei aqui" || b[0].Username != testkit.Admin || b[0].Entry.Title != "Bop" {
		t.Fatalf("bookmarks: %+v", r.Bookmarks)
	}
	if r := e.ok("getSong", q("id", e.ids.TrackBop)); r.Song.BookmarkPosition != 1500 {
		t.Fatalf("bookmarkPosition: %d", r.Song.BookmarkPosition)
	}
	ud, _ := e.k.Store.UserData(context.Background(), e.k.AdminUser.ID, e.ids.TrackBop)
	if ud.Position != 1.5 {
		t.Fatalf("posição no userdata: %v", ud.Position)
	}
	// scrobble não apaga o marcador
	e.ok("scrobble", q("id", e.ids.TrackBop))
	if r := e.ok("getBookmarks", q()); len(r.Bookmarks.Bookmark) != 1 {
		t.Fatal("scrobble apagou")
	}
	e.ok("deleteBookmark", q("id", e.ids.TrackBop))
	if r := e.ok("getBookmarks", q()); len(r.Bookmarks.Bookmark) != 0 {
		t.Fatal("não apagou")
	}
	ud, _ = e.k.Store.UserData(context.Background(), e.k.AdminUser.ID, e.ids.TrackBop)
	if ud.Position != 0 {
		t.Fatalf("posição ficou: %v", ud.Position)
	}
	e.failed("createBookmark", q("id", "nada"), 70)
	e.failed("createBookmark", q(), 10)
}
