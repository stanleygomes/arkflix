package admin_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

type itemT struct {
	ID, Kind, Name, Platform, SeriesName, Image string
	RID                                         int64
	Year                                        int
}

type listT struct {
	ID, Name, UpdatedAt, CreatedAt, Cover string
	RID                                   int64
	Count                                 int
	Kinds                                 []string
	Items                                 []itemT
}

func names(its []itemT) string {
	var out []string
	for _, it := range its {
		out = append(out, it.Name)
	}
	return strings.Join(out, "|")
}

func TestLists(t *testing.T) {
	e := setup(t)
	var l listT
	e.must(201, "POST", "/arkflix/api/lists", e.admin, map[string]string{"name": " Fim de semana "}, &l)
	if l.Name != "Fim de semana" || len(l.ID) != 32 || l.RID == 0 || l.Count != 0 || l.Kinds == nil || l.Cover != "" || l.UpdatedAt == "" {
		t.Fatalf("criada: %+v", l)
	}
	e.must(400, "POST", "/arkflix/api/lists", e.admin, map[string]string{"name": " "}, nil)
	base := "/arkflix/api/lists/" + l.ID

	// itens de qualquer tipo; ROM pelo id do RomM; repetido é ignorado
	super, _ := e.k.Store.Item(context.Background(), e.ids.RomSuper)
	e.must(200, "POST", base+"/items", e.admin, map[string]any{"ids": []string{e.ids.RomOutro, e.ids.MovieBig,
		"rom:" + itoa(super.RID), e.ids.Ep2, e.ids.TrackBip, e.ids.Album, e.ids.Series, e.ids.MovieBig}}, &l)
	if l.Count != 7 || strings.Join(l.Kinds, ",") != "rom,movie,episode,track,album,series" || l.Cover != e.ids.MovieBig {
		t.Fatalf("depois de pôr: %+v", l)
	}
	for _, bad := range []string{"nada", e.ids.Season1, e.ids.Artist, e.ids.BiosPSX, e.ids.SNES, "rom:abc"} {
		e.must(400, "POST", base+"/items", e.admin, map[string]any{"ids": []string{bad}}, nil)
	}

	e.must(200, "GET", base, e.admin, nil, &l)
	if names(l.Items) != "Outro Jogo|O Grande Teste|Super Teste|O Segundo|Bip|Disco Teste|Série de Teste" {
		t.Fatalf("itens: %s", names(l.Items))
	}
	ep := l.Items[3]
	if ep.Kind != "episode" || ep.SeriesName != "Série de Teste" || ep.Image != "/arkflix/api/items/"+e.ids.Ep2+"/image" {
		t.Fatalf("episódio: %+v", ep)
	}
	rom := l.Items[0]
	if rom.Kind != "rom" || rom.Platform != "snes" || rom.Image != "" || rom.RID == 0 {
		t.Fatalf("rom: %+v", rom)
	}
	if mv := l.Items[1]; mv.Year != 2021 || mv.Image == "" {
		t.Fatalf("filme: %+v", mv)
	}

	// tira (por id e por rom:<rid>) e reordena
	e.must(204, "DELETE", base+"/items/"+e.ids.Series, e.admin, nil, nil)
	e.must(204, "DELETE", base+"/items/rom:"+itoa(super.RID), e.admin, nil, nil)
	e.must(200, "PUT", base+"/order", e.admin, map[string]any{"ids": []string{e.ids.TrackBip, e.ids.MovieBig}}, &l)
	if names(l.Items) != "Bip|O Grande Teste|Outro Jogo|O Segundo|Disco Teste" {
		t.Fatalf("ordem: %s", names(l.Items))
	}
	e.must(200, "PATCH", base, e.admin, map[string]string{"name": "Outro nome"}, &l)
	if l.Name != "Outro nome" || l.Count != 5 {
		t.Fatalf("patch: %+v", l)
	}
	e.must(400, "PATCH", base, e.admin, map[string]string{"name": ""}, nil)

	// a lista é do admin: a nina não vê nem mexe
	var ls []listT
	e.must(200, "GET", "/arkflix/api/lists", e.nina, nil, &ls)
	if len(ls) != 0 {
		t.Fatalf("nina vê: %+v", ls)
	}
	for _, c := range []struct{ m, p string }{{"GET", base}, {"PATCH", base}, {"DELETE", base},
		{"POST", base + "/items"}, {"DELETE", base + "/items/" + e.ids.MovieBig}, {"PUT", base + "/order"}} {
		if code := e.call(c.m, c.p, e.nina, map[string]any{"name": "x", "ids": []string{}}, nil); code != 404 {
			t.Fatalf("nina %s %s: %d", c.m, c.p, code)
		}
	}
	if code := e.call("GET", "/arkflix/api/lists", "", nil, nil); code != 401 {
		t.Fatalf("sem login: %d", code)
	}

	e.must(201, "POST", "/arkflix/api/lists", e.admin, map[string]string{"name": "A primeira"}, nil)
	e.must(200, "GET", "/arkflix/api/lists", e.admin, nil, &ls)
	if len(ls) != 2 || ls[0].Name != "A primeira" || ls[1].Count != 5 || ls[1].Items != nil {
		t.Fatalf("lista de listas: %+v", ls)
	}
	e.must(204, "DELETE", base, e.admin, nil, nil)
	e.must(404, "GET", base, e.admin, nil, nil)
}

// subsonic chama /rest/<m> como o usuário e devolve o subsonic-response.
func (e *env) subsonic(user, pass, m string, kv ...string) map[string]any {
	e.t.Helper()
	sum := md5.Sum([]byte(pass + "sal"))
	v := url.Values{"u": {user}, "t": {hex.EncodeToString(sum[:])}, "s": {"sal"}, "v": {"1.16.1"}, "c": {"teste"}, "f": {"json"}}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Add(kv[i], kv[i+1])
	}
	code, raw := e.req("GET", "/rest/"+m+"?"+v.Encode(), "", nil, "")
	var out struct {
		R map[string]any `json:"subsonic-response"`
	}
	if code != 200 || json.Unmarshal(raw, &out) != nil || out.R["status"] != "ok" {
		e.t.Fatalf("subsonic %s: %d %s", m, code, raw)
	}
	return out.R
}

// TestListsEverywhere: a mesma lista é Playlist no Jellyfin, coleção no
// RomM e playlist no Subsonic, cada um com a sua parte.
func TestListsEverywhere(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	var l listT
	e.must(201, "POST", "/arkflix/api/lists", e.admin, map[string]string{"name": "Tudo"}, &l)
	e.must(200, "POST", "/arkflix/api/lists/"+l.ID+"/items", e.admin,
		map[string]any{"ids": []string{e.ids.MovieBig, e.ids.RomSuper, e.ids.TrackBop}}, nil)
	super, _ := e.k.Store.Item(ctx, e.ids.RomSuper)

	// Jellyfin: pasta Playlists nas bibliotecas e a playlist com filme e faixa
	type jfItem struct {
		Id, Name, Type, MediaType, CollectionType, PlaylistItemId string
		ChildCount                                                *int
	}
	var views struct{ Items []jfItem }
	e.must(200, "GET", "/Users/"+e.k.AdminUser.ID+"/Views", e.admin, nil, &views)
	if v := views.Items[len(views.Items)-1]; v.CollectionType != "playlists" || *v.ChildCount != 1 {
		t.Fatalf("views: %+v", views.Items)
	}
	var pls struct {
		Items            []jfItem
		TotalRecordCount int
	}
	e.must(200, "GET", "/Items?IncludeItemTypes=Playlist&Recursive=true", e.admin, nil, &pls)
	if pls.TotalRecordCount != 1 || pls.Items[0].Id != l.ID || pls.Items[0].Type != "Playlist" || *pls.Items[0].ChildCount != 2 {
		t.Fatalf("playlists: %+v", pls)
	}
	e.must(200, "GET", "/Playlists/"+l.ID+"/Items", e.admin, nil, &pls)
	if len(pls.Items) != 2 || pls.Items[0].Id != e.ids.MovieBig || pls.Items[1].Id != e.ids.TrackBop || pls.Items[0].PlaylistItemId == "" {
		t.Fatalf("itens da playlist: %+v", pls)
	}

	// RomM: coleção só com a ROM; filtro de ROMs pela coleção
	var colls []struct {
		ID       int64   `json:"id"`
		Name     string  `json:"name"`
		RomIDs   []int64 `json:"rom_ids"`
		RomCount int     `json:"rom_count"`
		IsPublic bool    `json:"is_public"`
		Cover    *string `json:"path_cover_small"`
	}
	romm := "Bearer " + e.token(e.k.AdminUser.ID, auth.KindRomM, "")
	e.must(200, "GET", "/api/collections", romm, nil, &colls)
	if len(colls) != 1 || colls[0].ID != l.RID || colls[0].RomCount != 1 || colls[0].RomIDs[0] != super.RID || colls[0].Cover != nil {
		t.Fatalf("coleções: %+v", colls)
	}
	var roms struct {
		Items []struct{ ID int64 } `json:"items"`
		Total int                  `json:"total"`
	}
	e.must(200, "GET", "/api/roms?collection_id="+itoa(l.RID), romm, nil, &roms)
	if roms.Total != 1 || roms.Items[0].ID != super.RID {
		t.Fatalf("roms da coleção: %+v", roms)
	}

	// Subsonic: playlist só com a faixa
	r := e.subsonic(testkit.Admin, testkit.AdminPass, "getPlaylist", "id", l.ID)
	pl := r["playlist"].(map[string]any)
	entries := pl["entry"].([]any)
	if pl["songCount"].(float64) != 1 || len(entries) != 1 || entries[0].(map[string]any)["id"] != e.ids.TrackBop {
		t.Fatalf("subsonic: %+v", pl)
	}

	// criada pelo Subsonic aparece no app
	r = e.subsonic(testkit.User, testkit.UserPass, "createPlaylist", "name", "Da Nina", "songId", e.ids.TrackBip)
	ninaID := r["playlist"].(map[string]any)["id"].(string)
	var ls []listT
	e.must(200, "GET", "/arkflix/api/lists", e.nina, nil, &ls)
	if len(ls) != 1 || ls[0].ID != ninaID || ls[0].Name != "Da Nina" || ls[0].Count != 1 || !slices.Equal(ls[0].Kinds, []string{"track"}) {
		t.Fatalf("lista da nina: %+v", ls)
	}
	// e uma lista só de filmes não vira playlist de música nem coleção
	e.must(200, "POST", "/arkflix/api/lists/"+ninaID+"/items", e.nina, map[string]any{"ids": []string{e.ids.MovieNoYear}}, nil)
	var filmes listT
	e.must(201, "POST", "/arkflix/api/lists", e.nina, map[string]string{"name": "Filmes"}, &filmes)
	e.must(200, "POST", "/arkflix/api/lists/"+filmes.ID+"/items", e.nina, map[string]any{"ids": []string{e.ids.MovieNoYear}}, nil)
	r = e.subsonic(testkit.User, testkit.UserPass, "getPlaylists")
	if got := r["playlists"].(map[string]any)["playlist"].([]any); len(got) != 1 {
		t.Fatalf("playlists da nina: %+v", got)
	}
	ninaRomm := "Bearer " + e.token(e.k.PlainUser.ID, auth.KindRomM, "")
	e.must(200, "GET", "/api/collections", ninaRomm, nil, &colls)
	if len(colls) != 0 {
		t.Fatalf("coleções da nina: %+v", colls)
	}
	// trocar as faixas pelo Subsonic não tira o filme da lista
	e.subsonic(testkit.User, testkit.UserPass, "updatePlaylist", "playlistId", ninaID, "songIndexToRemove", "0")
	var nl listT
	e.must(200, "GET", "/arkflix/api/lists/"+ninaID, e.nina, nil, &nl)
	if names(nl.Items) != "Filme Sem Ano" {
		t.Fatalf("depois do update: %s", names(nl.Items))
	}
}
