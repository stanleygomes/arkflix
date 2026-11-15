package admin_test

import (
	"context"
	"strconv"
	"testing"
)

func TestPlatforms(t *testing.T) {
	e := setup(t)
	var got []struct {
		Slug  string `json:"slug"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	e.must(200, "GET", "/arkflix/api/platforms", e.admin, nil, &got)
	if len(got) != 2 || got[0].Slug != "psx" || got[0].Count != 1 || got[1].Slug != "snes" || got[1].Count != 2 {
		t.Fatalf("consoles: %+v", got)
	}
	if code, _ := e.req("GET", "/arkflix/api/platforms", e.nina, nil, ""); code != 403 {
		t.Fatalf("usuário comum: %d", code)
	}
	// refresh de um item (sem chave do TMDB o Fetcher só ignora): 202
	e.must(202, "POST", "/arkflix/api/items/"+e.ids.MovieBig+"/refresh", e.admin, nil, nil)
	rom, _ := e.k.Store.Item(context.Background(), e.ids.RomSuper)
	e.must(202, "POST", "/arkflix/api/items/rom:"+strconv.FormatInt(rom.RID, 10)+"/refresh", e.admin, nil, nil)
}
