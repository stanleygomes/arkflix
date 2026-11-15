package jellyfin

import (
	"context"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

type plItemT struct {
	Id, Name, Type, MediaType, ParentId, PlaylistItemId, CollectionType string
	ChildCount                                                          *int
	RunTimeTicks                                                        int64
	IsFolder, CanDelete                                                 bool
	ImageTags                                                           map[string]string
}

type plListT struct {
	Items            []plItemT
	TotalRecordCount int
}

func plNames(items []plItemT) string {
	var out []string
	for _, it := range items {
		out = append(out, it.Name)
	}
	return strings.Join(out, "|")
}

func TestPlaylists(t *testing.T) {
	e := setup(t)
	// sem lista: nem pasta de playlists
	eq(t, "views sem lista", e.list("/UserViews").TotalRecordCount, 3)
	var pls plListT
	e.get("/Items?IncludeItemTypes=Playlist&Recursive=true", &pls)
	eq(t, "sem playlists", pls.TotalRecordCount, 0)

	// cria pelo corpo (CreatePlaylistDto), com itens; ROM é ignorada
	var created struct{ Id string }
	if code := e.call("POST", "/Playlists", map[string]any{"Name": "Mix", "Ids": []string{e.ids.TrackBop, e.ids.RomSuper, e.ids.TrackBip},
		"MediaType": "Audio", "UserId": e.userID}, &created); code != 200 || len(created.Id) != 32 {
		t.Fatalf("criar: %d %+v", code, created)
	}
	id := created.Id
	// e pela query (clientes antigos)
	var other struct{ Id string }
	if code := e.call("POST", "/Playlists?Name=Filmes&Ids="+e.ids.MovieBig, nil, &other); code != 200 {
		t.Fatalf("criar pela query: %d", code)
	}
	if code := e.call("POST", "/Playlists", map[string]any{"Name": " "}, nil); code != 400 {
		t.Fatalf("sem nome: %d", code)
	}

	views := e.list("/UserViews")
	eq(t, "views com lista", views.TotalRecordCount, 4)
	var view plItemT
	e.get("/Items/"+playlistsViewID, &view)
	eq(t, "pasta", view.CollectionType, "playlists")
	eq(t, "pasta filhos", *view.ChildCount, 2)
	if code := e.call("GET", "/Library/MediaFolders", nil, &pls); code != 200 || pls.TotalRecordCount != 3 {
		t.Fatalf("MediaFolders: %+v", pls)
	}

	e.get("/Items?IncludeItemTypes=Playlist&Recursive=true&SortBy=SortName", &pls)
	eq(t, "playlists", plNames(pls.Items), "Filmes|Mix")
	mix := pls.Items[1]
	if mix.Type != "Playlist" || mix.MediaType != "Audio" || !mix.IsFolder || *mix.ChildCount != 2 || mix.ParentId != playlistsViewID ||
		!mix.CanDelete || mix.RunTimeTicks != 4*ticksPerSecond || mix.ImageTags["Primary"] == "" {
		t.Fatalf("mix: %+v", mix)
	}
	eq(t, "filmes", pls.Items[0].MediaType, "Video")
	e.get("/Items?ParentId="+playlistsViewID, &pls)
	eq(t, "pela pasta", pls.TotalRecordCount, 2)

	// itens na ordem, com PlaylistItemId; pelos dois caminhos
	e.get("/Playlists/"+id+"/Items", &pls)
	eq(t, "itens", plNames(pls.Items), "Bop|Bip")
	if pls.Items[0].PlaylistItemId == "" || pls.Items[0].PlaylistItemId == pls.Items[1].PlaylistItemId {
		t.Fatalf("entradas: %+v", pls.Items)
	}
	e.get("/Users/"+e.userID+"/Items?ParentId="+id+"&Limit=1", &pls)
	eq(t, "página", plNames(pls.Items), "Bop")
	eq(t, "total", pls.TotalRecordCount, 2)

	var one plItemT
	e.get("/Items/"+id, &one)
	eq(t, "item", one.Type, "Playlist")
	var dto struct {
		OpenAccess bool
		ItemIds    []string
	}
	e.get("/Playlists/"+id, &dto)
	eq(t, "ItemIds", strings.Join(dto.ItemIds, ","), e.ids.TrackBop+","+e.ids.TrackBip)
	if code := e.call("GET", "/Playlists/"+id+"/Users", nil, nil); code != 200 {
		t.Fatalf("users: %d", code)
	}
	if code := e.call("GET", "/Playlists/"+id+"/Users/"+e.userID, nil, nil); code != 200 {
		t.Fatalf("user: %d", code)
	}
	// capa da playlist (sem login, como as outras)
	if resp, _ := e.req("GET", "/Items/"+id+"/Images/Primary", nil, "Authorization", ""); resp.StatusCode != 200 {
		t.Fatalf("capa: %d", resp.StatusCode)
	}

	// põe, move, tira por entrada
	if code := e.call("POST", "/Playlists/"+id+"/Items?ids="+e.ids.TrackSolta+","+e.ids.TrackBop, nil, nil); code != 204 {
		t.Fatalf("pôr: %d", code)
	}
	e.get("/Playlists/"+id+"/Items", &pls)
	eq(t, "depois de pôr", plNames(pls.Items), "Bop|Bip|Solta")
	if code := e.call("POST", "/Playlists/"+id+"/Items/"+pls.Items[2].PlaylistItemId+"/Move/0", nil, nil); code != 204 {
		t.Fatalf("mover: %d", code)
	}
	e.get("/Playlists/"+id+"/Items", &pls)
	eq(t, "depois de mover", plNames(pls.Items), "Solta|Bop|Bip")
	if code := e.call("DELETE", "/Playlists/"+id+"/Items?entryIds="+pls.Items[1].PlaylistItemId, nil, nil); code != 204 {
		t.Fatalf("tirar: %d", code)
	}
	e.get("/Playlists/"+id+"/Items", &pls)
	eq(t, "depois de tirar", plNames(pls.Items), "Solta|Bip")

	// atualiza (UpdatePlaylistDto): nome e itens; a ROM que estava fica na lista
	ls := lists.New(e.k.Store)
	ls.Add(context.Background(), id, []string{e.ids.RomSuper})
	if code := e.call("POST", "/Playlists/"+id, map[string]any{"Name": "Mix 2", "Ids": []string{e.ids.TrackBop}}, nil); code != 204 {
		t.Fatalf("update: %d", code)
	}
	e.get("/Items/"+id, &one)
	eq(t, "nome novo", one.Name, "Mix 2")
	es, _ := ls.Entries(context.Background(), id)
	eq(t, "entradas", strings.Join(lists.ItemIDs(es), ","), e.ids.TrackBop+","+e.ids.RomSuper)

	// outro usuário: não vê a privada, não mexe
	nina, _ := e.login(testkit.User, testkit.UserPass, "aparelho-nina")
	if code := e.call("GET", "/Playlists/"+id+"/Items", nil, nil, "Authorization", authHdr("aparelho-nina", nina)); code != 404 {
		t.Fatalf("nina vê: %d", code)
	}
	ls.Update(context.Background(), id, lists.Changes{Public: new(true)})
	if code := e.call("GET", "/Playlists/"+id+"/Items", nil, nil, "Authorization", authHdr("aparelho-nina", nina)); code != 200 {
		t.Fatalf("nina vê a pública: %d", code)
	}
	for _, c := range []struct{ m, p string }{{"POST", "/Playlists/" + id + "/Items?ids=" + e.ids.TrackBip},
		{"DELETE", "/Playlists/" + id + "/Items?entryIds=x"}, {"POST", "/Playlists/" + id}, {"DELETE", "/Items/" + id}} {
		if code := e.call(c.m, c.p, map[string]any{}, nil, "Authorization", authHdr("aparelho-nina", nina)); code != 403 {
			t.Fatalf("nina %s %s: %d", c.m, c.p, code)
		}
	}

	// apaga pelo DELETE /Items/{id}; mídia não se apaga
	if code := e.call("DELETE", "/Items/"+e.ids.MovieBig, nil, nil); code != 403 {
		t.Fatalf("apagar filme: %d", code)
	}
	if code := e.call("DELETE", "/Items/"+id, nil, nil); code != 204 {
		t.Fatalf("apagar: %d", code)
	}
	if code := e.call("GET", "/Playlists/"+id+"/Items", nil, nil); code != 404 {
		t.Fatalf("apagada: %d", code)
	}
}
