package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func TestAuth(t *testing.T) {
	e := setup(t)
	sum := md5.Sum([]byte(testkit.UserPass + "sal123"))
	tok := hex.EncodeToString(sum[:])
	key, err := e.k.Auth.NewToken(context.Background(), auth.Token{UserID: e.k.PlainUser.ID, Kind: auth.KindSubsonic, Name: "chave"}, "", 0)
	if err != nil {
		t.Fatal(err)
	}

	// as três formas
	e.ok("ping", q("u", testkit.User, "p", testkit.UserPass))
	e.ok("ping", q("u", testkit.User, "p", "enc:"+hex.EncodeToString([]byte(testkit.UserPass))))
	e.ok("ping", q("u", testkit.User, "t", tok, "s", "sal123"))
	e.ok("ping", q("u", "NINA", "t", strings.ToUpper(tok), "s", "sal123")) // nome sem caixa
	r := e.ok("tokenInfo", q("apiKey", key.Token))
	if r.TokenInfo == nil || r.TokenInfo.Username != testkit.User {
		t.Fatalf("tokenInfo: %+v", r.TokenInfo)
	}

	// erros: senha errada e usuário inexistente dão a mesma resposta
	a := e.failed("ping", q("u", testkit.User, "p", "errada"), 40)
	b := e.failed("ping", q("u", "ninguem", "p", "errada"), 40)
	if a.Error.Message != b.Error.Message || a.Error.Message != "Wrong username or password" {
		t.Fatalf("mensagens: %q %q", a.Error.Message, b.Error.Message)
	}
	e.failed("ping", q("u", testkit.User, "t", tok, "s", "outro-sal"), 40)
	e.failed("ping", q("u", testkit.User), 40) // sem senha nem token
	e.failed("ping", q("u", testkit.User, "p", "enc:zz"), 40)
	e.failed("ping", q("apiKey", "nao-existe"), 44)
	e.failed("ping", q("apiKey", key.Token, "u", testkit.User), 43)
	// apiKey de outro tipo não vale
	jf, _ := e.k.Auth.NewToken(context.Background(), auth.Token{UserID: e.k.PlainUser.ID, Kind: auth.KindJellyfin}, "", 0)
	e.failed("ping", q("apiKey", jf.Token), 44)

	// parâmetros obrigatórios: u, v, c
	w := e.raw(http.MethodGet, "/rest/ping", q("p", "x", "v", "1.16.1", "c", "x", "f", "json"))
	if r := decode(t, w); r.Error == nil || r.Error.Code != 10 || r.Error.Message != "missing parameter: 'u'" {
		t.Fatalf("sem u: %+v", r.Error)
	}
	w = e.raw(http.MethodGet, "/rest/ping", q("u", testkit.User, "p", testkit.UserPass, "c", "x", "f", "json"))
	if r := decode(t, w); r.Error == nil || r.Error.Code != 10 {
		t.Fatalf("sem v: %+v", r.Error)
	}

	// extensões não pedem login
	w = e.raw(http.MethodGet, "/rest/getOpenSubsonicExtensions", q("f", "json"))
	if r := decode(t, w); r.Status != "ok" || r.Extensions == nil || len(*r.Extensions) == 0 {
		t.Fatalf("extensões: %s", w.Body.String())
	}

	// admin só pra admin
	e.failed("startScan", q("u", testkit.User, "p", testkit.UserPass), 50)
	e.failed("getUsers", q("u", testkit.User, "p", testkit.UserPass), 50)
}

func TestRouting(t *testing.T) {
	e := setup(t)
	// .view, maiúsculas e POST com formulário
	for _, p := range []string{"/rest/ping.view", "/rest/PING", "/rest/Ping.view"} {
		if r := decode(t, e.raw(http.MethodGet, p, login(q("f", "json")))); r.Status != "ok" {
			t.Fatalf("%s: %+v", p, r)
		}
	}
	if r := decode(t, e.raw(http.MethodPost, "/rest/getSong", login(q("f", "json", "id", e.ids.TrackBip)))); r.Song == nil || r.Song.Title != "Bip" {
		t.Fatalf("POST: %+v", r)
	}
	// método desconhecido: 404 como o roteador do Navidrome
	if w := e.raw(http.MethodGet, "/rest/naoExiste", login(q())); w.Code != http.StatusNotFound {
		t.Fatalf("desconhecido: %d", w.Code)
	}
	// desligados
	if w := e.raw(http.MethodGet, "/rest/jukeboxControl", login(q())); w.Code != http.StatusNotImplemented {
		t.Fatalf("jukebox: %d", w.Code)
	}
	if w := e.raw(http.MethodGet, "/rest/getVideos", login(q())); w.Code != http.StatusGone {
		t.Fatalf("videos: %d", w.Code)
	}
	r := e.ok("getPodcasts", q())
	if r.Podcasts == nil {
		t.Fatal("podcasts")
	}
	e.ok("getInternetRadioStations", q())
	e.ok("getShares", q())
	e.ok("getNewestPodcasts", q())
}

func TestEnvelope(t *testing.T) {
	e := setup(t)
	const head = `<subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="1.16.1" type="arkflix" serverVersion="0.0.0-teste" openSubsonic="true">`
	if got := e.xmlBody("ping", q()); got != head+`</subsonic-response>` {
		t.Fatalf("ping xml: %s", got)
	}
	if got := e.xmlBody("getLicense", q()); got != head+`<license valid="true"></license></subsonic-response>` {
		t.Fatalf("license xml: %s", got)
	}
	w := e.raw(http.MethodGet, "/rest/ping", login(q("f", "json")))
	if got := w.Body.String(); got != `{"subsonic-response":{"status":"ok","version":"1.16.1","type":"arkflix","serverVersion":"0.0.0-teste","openSubsonic":true}}` {
		t.Fatalf("ping json: %s", got)
	}
	w = e.raw(http.MethodGet, "/rest/ping", login(q("f", "jsonp", "callback", "cb")))
	if got := w.Body.String(); !strings.HasPrefix(got, `cb({"subsonic-response":`) || w.Header().Get("Content-Type") != "application/javascript" {
		t.Fatalf("jsonp: %s", got)
	}
	w = e.raw(http.MethodGet, "/rest/ping", login(q("f", "jsonp", "callback", "alert(1)")))
	if r := decode(t, w); r.Status != "failed" {
		t.Fatalf("callback ruim: %s", w.Body.String())
	}
	errHead := strings.Replace(head, `status="ok"`, `status="failed"`, 1)
	if got := e.xmlBody("ping", q("u", "x", "p", "y")); got != errHead+`<error code="40" message="Wrong username or password"></error></subsonic-response>` {
		t.Fatalf("erro xml: %s", got)
	}
	if got := e.xmlBody("getMusicFolders", q()); got != head+`<musicFolders><musicFolder id="1" name="Músicas"></musicFolder></musicFolders></subsonic-response>` {
		t.Fatalf("pastas: %s", got)
	}
	exts := e.xmlBody("getOpenSubsonicExtensions", q())
	if !strings.Contains(exts, `<openSubsonicExtensions name="formPost"><versions>1</versions></openSubsonicExtensions>`) {
		t.Fatalf("extensões xml: %s", exts)
	}
	w = e.raw(http.MethodGet, "/rest/getOpenSubsonicExtensions", q("f", "json"))
	if !strings.Contains(w.Body.String(), `"openSubsonicExtensions":[{"name":"formPost","versions":[1]}`) {
		t.Fatalf("extensões json: %s", w.Body.String())
	}
}

func TestUsersAndScan(t *testing.T) {
	e := setup(t)
	r := e.ok("getUser", q("username", testkit.Admin))
	if u := r.User; u == nil || !u.AdminRole || !u.StreamRole || !u.DownloadRole || !u.ScrobblingEnabled || u.SettingsRole || len(u.Folder) != 1 {
		t.Fatalf("user: %+v", r.User)
	}
	e.failed("getUser", q("u", testkit.User, "p", testkit.UserPass, "username", testkit.Admin), 50)
	e.failed("getUser", q(), 10)
	if r := e.ok("getUsers", q()); len(r.Users.User) != 2 {
		t.Fatalf("users: %+v", r.Users)
	}
	xml := e.xmlBody("getUser", q("username", testkit.User, "u", testkit.User, "p", testkit.UserPass))
	if !strings.Contains(xml, `<user username="nina" scrobblingEnabled="true" adminRole="false" settingsRole="false" downloadRole="true"`) ||
		!strings.Contains(xml, `<folder>1</folder>`) {
		t.Fatalf("user xml: %s", xml)
	}
	r = e.ok("getScanStatus", q())
	if r.ScanStatus == nil || r.ScanStatus.Count != 3 || r.ScanStatus.Scanning {
		t.Fatalf("scan: %+v", r.ScanStatus)
	}
	e.ok("startScan", q())
}
