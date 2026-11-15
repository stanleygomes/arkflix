package admin_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

type meT struct {
	ID    string
	RID   int64
	Name  string
	Admin bool
}

func TestInfoAndAuth(t *testing.T) {
	e := setup(t)
	var info struct {
		Product, Name, Version string
		Writable               bool
		Features               []string
	}
	e.must(200, "GET", "/arkflix/api/info", "", nil, &info)
	if info.Product != "arkflix-server" || info.Version != "0.0.0-teste" || info.Writable || !slices.Contains(info.Features, "lists") {
		t.Fatalf("info: %+v", info)
	}

	// sem login, token errado
	code, raw := e.req("GET", "/arkflix/api/me", "", nil, "")
	if code != 401 || errMsg(t, raw) == "" {
		t.Fatalf("sem token: %d %s", code, raw)
	}
	if code := e.call("GET", "/arkflix/api/me", "Bearer nada", nil, nil); code != 401 {
		t.Fatalf("token errado: %d", code)
	}
	// refresh do RomM não vale
	refresh := e.token(e.k.AdminUser.ID, auth.KindRomMRefresh, "")
	if code := e.call("GET", "/arkflix/api/me", "Bearer "+refresh, nil, nil); code != 401 {
		t.Fatalf("refresh: %d", code)
	}

	// MediaBrowser Token=, X-Emby-Token, Bearer (access e client do RomM), ?api_key=
	var me meT
	e.must(200, "GET", "/arkflix/api/me", e.admin, nil, &me)
	if me.ID != e.k.AdminUser.ID || me.RID != e.k.AdminUser.RID || me.Name != testkit.Admin || !me.Admin {
		t.Fatalf("me: %+v", me)
	}
	r, _ := http.NewRequest("GET", e.srv.URL+"/arkflix/api/me", nil)
	r.Header.Set("X-Emby-Token", e.nina)
	res, err := http.DefaultClient.Do(r)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("X-Emby-Token: %v %v", err, res.StatusCode)
	}
	res.Body.Close()
	access := e.token(e.k.PlainUser.ID, auth.KindRomM, "")
	e.must(200, "GET", "/arkflix/api/me", "Bearer "+access, nil, &me)
	if me.Name != testkit.User || me.Admin {
		t.Fatalf("me nina: %+v", me)
	}
	client := e.token(e.k.AdminUser.ID, auth.KindRomMClient, "rmm_")
	e.must(200, "GET", "/arkflix/api/me", "Bearer "+client, nil, &me)
	e.must(200, "GET", "/arkflix/api/me?api_key="+url.QueryEscape(e.admin), "", nil, &me)
	// Basic não é aceito aqui
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(testkit.Admin+":"+testkit.AdminPass))
	if code := e.call("GET", "/arkflix/api/me", basic, nil, nil); code != 401 {
		t.Fatalf("basic: %d", code)
	}

	// rota de admin com usuário comum: 403
	for _, p := range []string{"/arkflix/api/status", "/arkflix/api/users"} {
		code, raw := e.req("GET", p, e.nina, nil, "")
		if code != 403 || errMsg(t, raw) == "" {
			t.Fatalf("%s nina: %d %s", p, code, raw)
		}
	}
	for _, p := range []string{"/arkflix/api/scan", "/arkflix/api/upload", "/arkflix/api/items/" + e.ids.MovieBig + "/hide",
		"/arkflix/api/items/" + e.ids.MovieBig + "/refresh"} {
		if code := e.call("POST", p, e.nina, nil, nil); code != 403 {
			t.Fatalf("%s nina: %d", p, code)
		}
	}
	// rota que não existe
	if code, raw := e.req("GET", "/arkflix/api/nada", e.admin, nil, ""); code != 404 || errMsg(t, raw) == "" {
		t.Fatalf("404: %d %s", code, raw)
	}
	// CORS
	r, _ = http.NewRequest("OPTIONS", e.srv.URL+"/arkflix/api/lists", nil)
	res, err = http.DefaultClient.Do(r)
	if err != nil || res.StatusCode != 204 || res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("options: %v %v", err, res)
	}
	res.Body.Close()
}

func TestStatus(t *testing.T) {
	e := setup(t)
	games := e.k.Config.Games
	e.k.Config.Games = filepath.Join(e.k.Root, "disco-desmontado")
	var st struct {
		Version, StartedAt string
		Writable           bool
		Libraries          []struct {
			ID, Name, Path                       string
			Count                                int
			Episodes, Albums, Artists, Platforms *int
			Online                               bool
		}
		Scan struct {
			Running          bool
			StartedAt, Error string
			FinishedAt       string
			Count            int
		}
		Metadata struct {
			Enabled, Tmdb, Running bool
			Pending                int
			Error                  string
		}
		Sessions []struct{ User, Client, Device, LastSeen string }
	}
	// o login da nina por outro aparelho conta como sessão
	e.must(200, "GET", "/arkflix/api/me", e.nina, nil, nil)
	e.must(200, "GET", "/arkflix/api/status", e.admin, nil, &st)
	if st.Version != "0.0.0-teste" || st.StartedAt == "" || st.Writable || len(st.Libraries) != 4 {
		t.Fatalf("status: %+v", st)
	}
	libs := map[string]int{}
	for _, l := range st.Libraries {
		libs[l.ID] = l.Count
		switch l.ID {
		case "shows":
			if l.Episodes == nil || *l.Episodes != 3 || !l.Online {
				t.Fatalf("séries: %+v", l)
			}
		case "music":
			if *l.Albums != 2 || *l.Artists != 2 {
				t.Fatalf("música: %+v", l)
			}
		case "games":
			if l.Online || *l.Platforms != 2 {
				t.Fatalf("jogos fora do ar: %+v", l)
			}
		}
	}
	if libs["movies"] != 3 || libs["shows"] != 1 || libs["music"] != 3 || libs["games"] != 3 {
		t.Fatalf("contagens: %v", libs)
	}
	users := map[string]bool{}
	for _, s := range st.Sessions {
		users[s.User] = true
		if s.Client != "teste" || s.LastSeen == "" {
			t.Fatalf("sessão: %+v", s)
		}
	}
	if !users[testkit.Admin] || !users[testkit.User] {
		t.Fatalf("sessões: %+v", st.Sessions)
	}
	if st.Scan.Running || st.Scan.StartedAt != "" || st.Metadata.Running {
		t.Fatalf("scan/metadados: %+v %+v", st.Scan, st.Metadata)
	}

	e.k.Config.Games = games
	var empty map[string]any
	e.must(202, "POST", "/arkflix/api/scan", e.admin, nil, &empty)
	e.waitScan()
	e.must(200, "GET", "/arkflix/api/status", e.admin, nil, &st)
	if st.Scan.FinishedAt == "" || st.Scan.Count == 0 {
		t.Fatalf("depois do scan: %+v", st.Scan)
	}
	e.must(202, "POST", "/arkflix/api/items/"+e.ids.MovieBig+"/refresh", e.admin, nil, nil)
	e.must(404, "POST", "/arkflix/api/items/nada/refresh", e.admin, nil, nil)
}

type userT struct {
	ID, Name, CreatedAt, LastLogin string
	RID                            int64
	Admin                          bool
}

func TestUsers(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	var list []userT
	e.must(200, "GET", "/arkflix/api/users", e.admin, nil, &list)
	if len(list) != 2 || list[0].Name != testkit.User || list[0].CreatedAt == "" {
		t.Fatalf("lista: %+v", list)
	}

	var u userT
	e.must(201, "POST", "/arkflix/api/users", e.admin, map[string]any{"name": " ana ", "password": "x1", "admin": false}, &u)
	if u.Name != "ana" || u.Admin || len(u.ID) != 32 || u.RID == 0 || u.LastLogin != "" {
		t.Fatalf("criado: %+v", u)
	}
	e.must(409, "POST", "/arkflix/api/users", e.admin, map[string]any{"name": "ANA", "password": "x"}, nil)
	e.must(400, "POST", "/arkflix/api/users", e.admin, map[string]any{"name": "", "password": "x"}, nil)
	e.must(400, "POST", "/arkflix/api/users", e.admin, map[string]any{"name": "b", "password": ""}, nil)
	e.must(400, "POST", "/arkflix/api/users", e.admin, "não é objeto", nil)

	// troca senha e vira admin
	e.must(200, "PATCH", "/arkflix/api/users/"+u.ID, e.admin, map[string]any{"password": "nova", "admin": true}, &u)
	if !u.Admin {
		t.Fatalf("patch: %+v", u)
	}
	if _, err := e.k.Auth.Login(ctx, "ana", "nova"); err != nil {
		t.Fatal("senha nova não vale:", err)
	}
	e.must(400, "PATCH", "/arkflix/api/users/"+u.ID, e.admin, map[string]any{"password": ""}, nil)
	e.must(404, "PATCH", "/arkflix/api/users/nada", e.admin, map[string]any{"admin": true}, nil)

	// dois admins: dá pra rebaixar um; o último não
	e.must(200, "PATCH", "/arkflix/api/users/"+u.ID, e.admin, map[string]any{"admin": false}, &u)
	code, raw := e.req("PATCH", "/arkflix/api/users/"+e.k.AdminUser.ID, e.admin, strings.NewReader(`{"admin":false}`), "application/json")
	if code != 409 || errMsg(t, raw) == "" {
		t.Fatalf("último admin: %d %s", code, raw)
	}

	// não apaga a si mesmo
	e.must(409, "DELETE", "/arkflix/api/users/"+e.k.AdminUser.ID, e.admin, nil, nil)
	e.must(204, "DELETE", "/arkflix/api/users/"+u.ID, e.admin, nil, nil)
	e.must(404, "DELETE", "/arkflix/api/users/"+u.ID, e.admin, nil, nil)
	e.must(200, "GET", "/arkflix/api/users", e.admin, nil, &list)
	if len(list) != 2 {
		t.Fatalf("depois de apagar: %+v", list)
	}
	// nina não mexe em usuário
	e.must(403, "POST", "/arkflix/api/users", e.nina, map[string]any{"name": "x", "password": "y"}, nil)
	e.must(403, "DELETE", "/arkflix/api/users/"+e.k.AdminUser.ID, e.nina, nil, nil)
}

func TestItemImage(t *testing.T) {
	e := setup(t)
	code, raw := e.req("GET", "/arkflix/api/items/"+e.ids.MovieBig+"/image?maxWidth=20", e.admin, nil, "")
	if code != 200 || !strings.HasPrefix(string(raw), "\x89PNG") {
		t.Fatalf("pôster: %d", code)
	}
	// episódio sem imagem: a da série
	if code, _ := e.req("GET", "/arkflix/api/items/"+e.ids.Ep1+"/image", e.nina, nil, ""); code != 200 {
		t.Fatalf("episódio: %d", code)
	}
	// faixa: capa embutida
	if code, _ := e.req("GET", "/arkflix/api/items/"+e.ids.TrackBip+"/image", e.nina, nil, ""); code != 200 {
		t.Fatalf("faixa: %d", code)
	}
	// ROM sem capa; id que não existe; sem login
	if code, _ := e.req("GET", "/arkflix/api/items/"+e.ids.RomSuper+"/image", e.admin, nil, ""); code != 404 {
		t.Fatalf("rom: %d", code)
	}
	if code, _ := e.req("GET", "/arkflix/api/items/rom:999/image", e.admin, nil, ""); code != 404 {
		t.Fatalf("rom inexistente: %d", code)
	}
	if code, _ := e.req("GET", "/arkflix/api/items/"+e.ids.MovieBig+"/image", "", nil, ""); code != 401 {
		t.Fatalf("sem login: %d", code)
	}
}
