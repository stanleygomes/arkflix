package jellyfin

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func TestLogin(t *testing.T) {
	e := setup(t)
	var res struct {
		AccessToken, ServerId string
		User                  struct {
			Id, Name string
			Policy   struct{ IsAdministrator bool }
		}
		SessionInfo struct{ Id, UserId, DeviceId string }
	}
	code := e.call("POST", "/Users/AuthenticateByName", map[string]string{"Username": "NINA", "Pw": testkit.UserPass},
		&res, "Authorization", authHdr("celular", ""))
	eq(t, "status", code, 200)
	eq(t, "user", res.User.Id, e.k.PlainUser.ID)
	eq(t, "admin", res.User.Policy.IsAdministrator, false)
	eq(t, "server id", res.ServerId, e.s.serverID)
	eq(t, "device", res.SessionInfo.DeviceId, "celular")
	eq(t, "session user", res.SessionInfo.UserId, e.k.PlainUser.ID)
	if len(res.ServerId) != 32 || len(res.AccessToken) < 32 {
		t.Fatalf("ids: %+v", res)
	}

	// senha errada / usuário inexistente
	for _, body := range []map[string]string{{"Username": testkit.User, "Pw": "x"}, {"Username": "ninguem", "Pw": "x"}} {
		eq(t, "login ruim", e.call("POST", "/Users/AuthenticateByName", body, nil, "Authorization", authHdr("x", "")), 401)
	}

	// mesmo aparelho de novo: token antigo cai; outro aparelho continua
	old := e.token
	other, _ := e.login(testkit.Admin, testkit.AdminPass, "aparelho-2")
	again, _ := e.login(testkit.Admin, testkit.AdminPass, "aparelho-1")
	eq(t, "token velho", e.call("GET", "/Users/Me", nil, nil, "Authorization", authHdr("aparelho-1", old)), 401)
	eq(t, "token novo", e.call("GET", "/Users/Me", nil, nil, "Authorization", authHdr("aparelho-1", again)), 200)
	eq(t, "outro aparelho", e.call("GET", "/Users/Me", nil, nil, "Authorization", authHdr("aparelho-2", other)), 200)

	// logout derruba
	eq(t, "logout", e.call("POST", "/Sessions/Logout", nil, nil, "Authorization", authHdr("aparelho-2", other)), 204)
	eq(t, "pós logout", e.call("GET", "/Users/Me", nil, nil, "Authorization", authHdr("aparelho-2", other)), 401)
}

func TestAuthVariants(t *testing.T) {
	e := setup(t)
	tok := e.token
	cases := []struct {
		name, path string
		hdr        []string
		want       int
	}{
		{"Authorization", "/Users/Me", nil, 200},
		{"X-Emby-Authorization", "/Users/Me", []string{"Authorization", "", "X-Emby-Authorization", `Emby Client="x", Device="y", DeviceId="z", Version="1", Token="` + tok + `"`}, 200},
		{"X-Emby-Token", "/Users/Me", []string{"Authorization", "", "X-Emby-Token", tok}, 200},
		{"X-MediaBrowser-Token", "/Users/Me", []string{"Authorization", "", "X-MediaBrowser-Token", tok}, 200},
		{"api_key", "/Users/Me?api_key=" + tok, []string{"Authorization", ""}, 200},
		{"ApiKey", "/Users/Me?ApiKey=" + tok, []string{"Authorization", ""}, 200},
		{"sem token", "/Users/Me", []string{"Authorization", ""}, 401},
		{"token errado", "/Users/Me", []string{"Authorization", authHdr("a", "nada")}, 401},
		{"sem token no header", "/Users/Me", []string{"Authorization", authHdr("a", "")}, 401},
	}
	for _, c := range cases {
		eq(t, c.name, e.call("GET", c.path, nil, nil, c.hdr...), c.want)
	}
	// valor url-encoded (jellyfin-web)
	enc := `MediaBrowser Client="Jellyfin%20Web", Device="Firefox", DeviceId="TW96aWxsYQ%3D%3D", Version="10.10.7", Token="` + tok + `"`
	eq(t, "encoded", e.call("GET", "/Users/Me", nil, nil, "Authorization", enc), 200)
	if a := parseAuthHeader(enc); a.Client != "Jellyfin Web" || a.DeviceID != "TW96aWxsYQ==" {
		t.Fatalf("%+v", a)
	}
}

func TestRoutingAndCORS(t *testing.T) {
	e := setup(t)
	for _, p := range []string{"/users/me", "/USERS/ME", "/emby/Users/Me", "/jellyfin/users/Me", "/Users/" + e.userID} {
		eq(t, p, e.call("GET", p, nil, nil), 200)
	}
	// GUID com hífen e maiúsculo
	id := e.ids.MoviePixel
	dashed := strings.ToUpper(id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:])
	var it dtoT
	e.get("/Users/"+e.userID+"/Items/"+dashed, &it)
	eq(t, "id normalizado", it.Id, id)

	resp, _ := e.req("OPTIONS", "/Users/Me", nil, "Access-Control-Request-Headers", "authorization", "Origin", "http://x")
	eq(t, "options", resp.StatusCode, 204)
	eq(t, "cors", resp.Header.Get("Access-Control-Allow-Origin"), "*")
	resp, _ = e.req("GET", "/Users/Me", nil)
	eq(t, "cors get", resp.Header.Get("Access-Control-Allow-Origin"), "*")
	eq(t, "404", e.call("GET", "/NaoExiste/Mesmo", nil, nil), 404)

	// usuário comum não lê os dados de outro
	tok, _ := e.login(testkit.User, testkit.UserPass, "x")
	eq(t, "outro user", e.call("GET", "/Users/"+e.userID+"/Items", nil, nil, "Authorization", authHdr("x", tok)), 403)
	var users []map[string]any
	e.call("GET", "/Users", nil, &users, "Authorization", authHdr("x", tok))
	eq(t, "users comum", len(users), 1)
	e.get("/Users", &users)
	eq(t, "users admin", len(users), 2)
}

func TestPublicEndpoints(t *testing.T) {
	e := setup(t)
	var info map[string]any
	eq(t, "info", e.call("GET", "/System/Info/Public", nil, &info, "Authorization", ""), 200)
	eq(t, "product", info["ProductName"], any("Jellyfin Server"))
	eq(t, "id", info["Id"], any(e.s.serverID))
	eq(t, "wizard", info["StartupWizardCompleted"], any(true))
	eq(t, "name", info["ServerName"], any("arkflix de teste"))
	if !strings.HasPrefix(info["LocalAddress"].(string), "http://127.0.0.1:") || !strings.HasPrefix(info["Version"].(string), "10.") {
		t.Fatal(info)
	}
	for _, m := range []string{"GET", "POST"} {
		resp, raw := e.req(m, "/System/Ping", nil, "Authorization", "")
		eq(t, "ping", resp.StatusCode, 200)
		eq(t, "ping body", string(raw), `"Jellyfin Server"`)
	}
	_, raw := e.req("GET", "/Users/Public", nil, "Authorization", "")
	eq(t, "users public", strings.TrimSpace(string(raw)), "[]")
	_, raw = e.req("GET", "/QuickConnect/Enabled", nil, "Authorization", "")
	eq(t, "quick connect", strings.TrimSpace(string(raw)), "false")
	eq(t, "branding", e.call("GET", "/Branding/Configuration", nil, nil, "Authorization", ""), 200)
	eq(t, "branding css", e.call("GET", "/Branding/Css", nil, nil, "Authorization", ""), 200)
	eq(t, "system info sem login", e.call("GET", "/System/Info", nil, nil, "Authorization", ""), 401)
	var full map[string]any
	e.get("/System/Info", &full)
	eq(t, "full", full["HasPendingRestart"], any(false))

	// o resto do pacote de inicialização dos clientes
	for _, p := range []string{"/Plugins", "/ScheduledTasks", "/Packages", "/Localization/Cultures", "/Localization/Countries",
		"/Localization/ParentalRatings", "/Localization/Options", "/Sessions", "/Devices", "/Library/VirtualFolders",
		"/Library/MediaFolders", "/DisplayPreferences/usersettings?client=emby", "/System/Endpoint", "/UserViews/GroupingOptions"} {
		eq(t, p, e.call("GET", p, nil, nil), 200)
	}
	eq(t, "capabilities", e.call("POST", "/Sessions/Capabilities/Full", map[string]any{"PlayableMediaTypes": []string{"Video"}}, nil), 204)
	eq(t, "capabilities", e.call("POST", "/Sessions/Capabilities?PlayableMediaTypes=Video", nil, nil), 204)
	eq(t, "config", e.call("POST", "/Users/"+e.userID+"/Configuration", map[string]any{}, nil), 204)
	eq(t, "web", e.call("GET", "/web/index.html", nil, nil), http.StatusNotFound)
}

func TestDisplayPreferences(t *testing.T) {
	e := setup(t)
	var p map[string]any
	e.get("/DisplayPreferences/usersettings?userId="+e.userID+"&client=emby", &p)
	eq(t, "padrão", p["SortBy"], any("SortName"))
	p["SortBy"] = "DateCreated"
	p["CustomPrefs"] = map[string]string{"homesection0": "resume"}
	eq(t, "post", e.call("POST", "/DisplayPreferences/usersettings?userId="+e.userID+"&client=emby", p, nil), 204)
	e.get("/DisplayPreferences/usersettings?client=emby", &p)
	eq(t, "salvo", p["SortBy"], any("DateCreated"))
	eq(t, "custom", p["CustomPrefs"].(map[string]any)["homesection0"], any("resume"))
	e.get("/DisplayPreferences/usersettings?client=outro", &p)
	eq(t, "por cliente", p["SortBy"], any("SortName"))
}
