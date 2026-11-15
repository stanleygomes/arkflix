package romm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/api/romm"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// env é o servidor de teste com o acervo semeado.
type env struct {
	t   *testing.T
	k   *testkit.Kit
	ids testkit.SeedIDs
	srv *httptest.Server
}

func newEnv(t *testing.T) *env {
	t.Helper()
	k := testkit.New(t)
	ids := k.Seed(t)
	srv := httptest.NewServer(romm.New(k.Deps))
	t.Cleanup(srv.Close)
	return &env{t: t, k: k, ids: ids, srv: srv}
}

// rid devolve o id numérico de um item semeado.
func (e *env) rid(id string) int64 {
	e.t.Helper()
	it, err := e.k.Store.Item(context.Background(), id)
	if err != nil {
		e.t.Fatal(err)
	}
	return it.RID
}

type resp struct {
	status int
	header http.Header
	body   []byte
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("json: %v\n%s", err, r.body)
	}
}

// do faz a requisição; auth é "Bearer x", "Basic x" ou "" (sem login).
func (e *env) do(method, path, auth string, body io.Reader, ctype string, hdr ...string) resp {
	e.t.Helper()
	req, err := http.NewRequest(method, e.srv.URL+path, body)
	if err != nil {
		e.t.Fatal(err)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return resp{status: res.StatusCode, header: res.Header, body: b}
}

func (e *env) get(path, auth string) resp { return e.do(http.MethodGet, path, auth, nil, "") }

func (e *env) postJSON(path, auth string, v any) resp {
	b, _ := json.Marshal(v)
	return e.do(http.MethodPost, path, auth, bytes.NewReader(b), "application/json")
}

// tokenResp é a resposta do /api/token.
type tokenResp struct {
	Access    string `json:"access_token"`
	Refresh   string `json:"refresh_token"`
	Type      string `json:"token_type"`
	Expires   int    `json:"expires"`
	ExpiresIn int    `json:"expires_in"`
}

func (e *env) passwordToken(user, pass, scope string) (resp, tokenResp) {
	e.t.Helper()
	form := url.Values{"grant_type": {"password"}, "username": {user}, "password": {pass}}
	if scope != "" {
		form.Set("scope", scope)
	}
	r := e.do(http.MethodPost, "/api/token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	var tr tokenResp
	if r.status == http.StatusOK {
		r.json(e.t, &tr)
	}
	return r, tr
}

// bearer devolve um "Bearer <access>" com todos os escopos do usuário.
func (e *env) bearer(user, pass string) string {
	e.t.Helper()
	r, tr := e.passwordToken(user, pass, "")
	if r.status != http.StatusOK {
		e.t.Fatalf("token: %d %s", r.status, r.body)
	}
	return "Bearer " + tr.Access
}

func (e *env) admin() string { return e.bearer(testkit.Admin, testkit.AdminPass) }
func (e *env) nina() string  { return e.bearer(testkit.User, testkit.UserPass) }

// multipartBody monta um multipart com os arquivos (campo → nome, conteúdo).
type part struct{ field, name, data string }

func multipartBody(t *testing.T, parts ...part) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, p := range parts {
		fw, err := w.CreateFormFile(p.field, p.name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(p.data))
	}
	w.Close()
	return &buf, w.FormDataContentType()
}

var isoRe = regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d{6})?\+00:00$`)

func detail(t *testing.T, r resp) string {
	t.Helper()
	var d struct {
		Detail string `json:"detail"`
	}
	r.json(t, &d)
	return d.Detail
}

func TestTokenPasswordAndRefresh(t *testing.T) {
	e := newEnv(t)
	r, _ := e.passwordToken(testkit.Admin, "errada", "")
	if r.status != http.StatusUnauthorized || detail(t, r) != "Invalid username or password" {
		t.Fatalf("senha errada: %d %s", r.status, r.body)
	}
	r, tr := e.passwordToken(testkit.Admin, testkit.AdminPass, "me.read roms.read")
	if r.status != http.StatusOK || tr.Access == "" || tr.Refresh == "" || tr.Type != "bearer" || tr.Expires != 1800 || tr.ExpiresIn != 1800 {
		t.Fatalf("token: %d %s", r.status, r.body)
	}
	if r := e.get("/api/users/me", "Bearer "+tr.Access); r.status != http.StatusOK {
		t.Fatalf("me: %d %s", r.status, r.body)
	}
	// escopo que não pediu
	if r := e.get("/api/platforms", "Bearer "+tr.Access); r.status != http.StatusForbidden || detail(t, r) != "Forbidden" {
		t.Fatalf("sem platforms.read: %d %s", r.status, r.body)
	}
	// refresh não serve de access
	if r := e.get("/api/users/me", "Bearer "+tr.Refresh); r.status != http.StatusUnauthorized {
		t.Fatalf("refresh como access: %d", r.status)
	}
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tr.Refresh}}
	r = e.do(http.MethodPost, "/api/token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	var tr2 tokenResp
	r.json(t, &tr2)
	if r.status != http.StatusOK || tr2.Access == "" || tr2.Access == tr.Access || tr2.Refresh == tr.Refresh {
		t.Fatalf("refresh: %d %s", r.status, r.body)
	}
	if r := e.get("/api/roms", "Bearer "+tr2.Access); r.status != http.StatusOK {
		t.Fatalf("escopo herdado: %d", r.status)
	}
	if r := e.get("/api/platforms", "Bearer "+tr2.Access); r.status != http.StatusForbidden {
		t.Fatalf("refresh aumentou escopo: %d", r.status)
	}
	// refresh é de uso único
	r = e.do(http.MethodPost, "/api/token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if r.status != http.StatusUnauthorized || detail(t, r) != "Invalid refresh token" {
		t.Fatalf("refresh reusado: %d %s", r.status, r.body)
	}
	// usuário comum não pede escopo de admin
	if r, _ := e.passwordToken(testkit.User, testkit.UserPass, "users.read"); r.status != http.StatusForbidden {
		t.Fatalf("escopo de admin: %d", r.status)
	}
	for _, gt := range []string{"client_credentials", "banana"} {
		form := url.Values{"grant_type": {gt}}
		if r := e.do(http.MethodPost, "/api/token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded"); r.status != http.StatusBadRequest {
			t.Fatalf("%s: %d", gt, r.status)
		}
	}
	// sem login
	if r := e.get("/api/users/me", ""); r.status != http.StatusUnauthorized || r.header.Get("WWW-Authenticate") != "Bearer" || detail(t, r) != "Not authenticated" {
		t.Fatalf("anônimo: %d %v %s", r.status, r.header, r.body)
	}
	if r := e.get("/api/users/me", "Bearer lixo"); r.status != http.StatusUnauthorized {
		t.Fatalf("token lixo: %d", r.status)
	}
}

func TestBasicAuthAndSession(t *testing.T) {
	e := newEnv(t)
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth(testkit.User, testkit.UserPass)
	basic := req.Header.Get("Authorization")
	for range 2 { // a segunda vem do cache
		r := e.get("/api/users/me", basic)
		var me struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		}
		r.json(t, &me)
		if r.status != http.StatusOK || me.Username != testkit.User || me.Role != "user" {
			t.Fatalf("basic: %d %s", r.status, r.body)
		}
	}
	req.SetBasicAuth(testkit.User, "errada")
	if r := e.get("/api/users/me", req.Header.Get("Authorization")); r.status != http.StatusUnauthorized {
		t.Fatalf("basic errado: %d", r.status)
	}
	// nina não é admin
	if r := e.get("/api/users", basic); r.status != http.StatusForbidden {
		t.Fatalf("users sem admin: %d", r.status)
	}

	// sessão por cookie
	r := e.do(http.MethodPost, "/api/login", basic, nil, "")
	if r.status != http.StatusOK {
		t.Fatalf("login: %d %s", r.status, r.body)
	}
	var cookie string
	for _, c := range (&http.Response{Header: r.header}).Cookies() {
		if c.Name == "romm_session" {
			cookie = c.Name + "=" + c.Value
		}
	}
	if cookie == "" {
		t.Fatal("sem cookie de sessão")
	}
	if r := e.do(http.MethodGet, "/api/users/me", "", nil, "", "Cookie", cookie); r.status != http.StatusOK {
		t.Fatalf("sessão: %d", r.status)
	}
	e.do(http.MethodPost, "/api/logout", "", nil, "", "Cookie", cookie)
	if r := e.do(http.MethodGet, "/api/users/me", "", nil, "", "Cookie", cookie); r.status != http.StatusUnauthorized {
		t.Fatalf("depois do logout: %d", r.status)
	}
}

func TestClientTokens(t *testing.T) {
	e := newEnv(t)
	nina := e.nina()
	r := e.postJSON("/api/client-tokens", nina, map[string]any{"name": "muOS", "scopes": []string{"roms.read", "assets.read", "me.read"}})
	var ct struct {
		ID        int64    `json:"id"`
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		Raw       string   `json:"raw_token"`
		UserID    int64    `json:"user_id"`
		ExpiresAt *string  `json:"expires_at"`
		CreatedAt string   `json:"created_at"`
	}
	r.json(t, &ct)
	if r.status != http.StatusCreated || !strings.HasPrefix(ct.Raw, "rmm_") || ct.Name != "muOS" || len(ct.Scopes) != 3 ||
		ct.UserID != e.k.PlainUser.RID || ct.ExpiresAt != nil || !isoRe.MatchString(ct.CreatedAt) {
		t.Fatalf("criar: %d %s", r.status, r.body)
	}
	tok := "Bearer " + ct.Raw
	if r := e.get("/api/users/me", tok); r.status != http.StatusOK {
		t.Fatalf("usar: %d", r.status)
	}
	if r := e.get("/api/platforms", tok); r.status != http.StatusForbidden {
		t.Fatalf("fora do escopo do token: %d", r.status)
	}
	// client token não cria outro (sem me.write)
	if r := e.postJSON("/api/client-tokens", tok, map[string]any{"name": "x", "scopes": []string{"roms.read"}}); r.status != http.StatusForbidden {
		t.Fatalf("token criando token: %d", r.status)
	}
	// escopo que a nina não tem
	if r := e.postJSON("/api/client-tokens", nina, map[string]any{"name": "x", "scopes": []string{"users.read"}}); r.status != http.StatusForbidden {
		t.Fatalf("escopo demais: %d %s", r.status, r.body)
	}
	if r := e.postJSON("/api/client-tokens", nina, map[string]any{"name": "x", "scopes": []string{"roms.read"}, "expires_in": "2d"}); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("expires_in ruim: %d", r.status)
	}
	r = e.postJSON("/api/client-tokens", nina, map[string]any{"name": "outro", "scopes": []string{"roms.read"}, "expires_in": "30d"})
	var ct2 struct {
		ID        int64   `json:"id"`
		ExpiresAt *string `json:"expires_at"`
	}
	r.json(t, &ct2)
	if r.status != http.StatusCreated || ct2.ExpiresAt == nil {
		t.Fatalf("com validade: %d %s", r.status, r.body)
	}
	var list []map[string]any
	r = e.get("/api/client-tokens", nina)
	r.json(t, &list)
	if r.status != http.StatusOK || len(list) != 2 || list[0]["raw_token"] != nil || list[0]["name"] != "muOS" {
		t.Fatalf("listar: %s", r.body)
	}
	// stanley não vê nem apaga token da nina
	admin := e.admin()
	r = e.get("/api/client-tokens", admin)
	r.json(t, &list)
	if len(list) != 0 {
		t.Fatalf("admin listou tokens da nina: %s", r.body)
	}
	if r := e.do(http.MethodDelete, "/api/client-tokens/"+itoa(ct.ID), admin, nil, ""); r.status != http.StatusNotFound {
		t.Fatalf("apagar de outro: %d", r.status)
	}
	// o próprio token se revoga
	if r := e.do(http.MethodDelete, "/api/client-tokens/"+itoa(ct.ID), tok, nil, ""); r.status != http.StatusOK {
		t.Fatalf("auto-revogar: %d %s", r.status, r.body)
	}
	if r := e.get("/api/users/me", tok); r.status != http.StatusUnauthorized {
		t.Fatalf("revogado ainda entra: %d", r.status)
	}
	r = e.do(http.MethodPut, "/api/client-tokens/"+itoa(ct2.ID)+"/regenerate", nina, nil, "")
	var re struct {
		ID  int64  `json:"id"`
		Raw string `json:"raw_token"`
	}
	r.json(t, &re)
	if r.status != http.StatusOK || re.ID != ct2.ID || !strings.HasPrefix(re.Raw, "rmm_") {
		t.Fatalf("regenerar: %d %s", r.status, r.body)
	}
	if r := e.get("/api/roms", "Bearer "+re.Raw); r.status != http.StatusOK {
		t.Fatalf("token novo: %d", r.status)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestHeartbeatConfigUsers(t *testing.T) {
	e := newEnv(t)
	r := e.get("/api/heartbeat", "")
	var hb struct {
		System struct {
			Version string `json:"VERSION"`
		} `json:"SYSTEM"`
		Meta map[string]bool `json:"METADATA_SOURCES"`
		FS   struct {
			Platforms []string `json:"FS_PLATFORMS"`
		} `json:"FILESYSTEM"`
		OIDC struct {
			Enabled bool `json:"ENABLED"`
		} `json:"OIDC"`
	}
	r.json(t, &hb)
	if r.status != http.StatusOK || hb.System.Version != romm.Version || hb.Meta["IGDB_API_ENABLED"] || len(hb.FS.Platforms) != 2 || hb.OIDC.Enabled {
		t.Fatalf("heartbeat: %s", r.body)
	}
	if r := e.get("/api/config", ""); r.status != http.StatusOK || !strings.Contains(string(r.body), "EXCLUDED_PLATFORMS") {
		t.Fatalf("config: %d %s", r.status, r.body)
	}
	admin := e.admin()
	var me map[string]any
	r = e.get("/api/users/me", admin)
	r.json(t, &me)
	for _, k := range []string{"id", "username", "email", "enabled", "role", "oauth_scopes", "has_password", "avatar_path",
		"last_login", "last_active", "created_at", "updated_at", "ui_settings", "current_device_id"} {
		if _, ok := me[k]; !ok {
			t.Errorf("users/me sem %s", k)
		}
	}
	if me["role"] != "admin" || me["id"] != float64(e.k.AdminUser.RID) || !isoRe.MatchString(me["created_at"].(string)) {
		t.Fatalf("me: %s", r.body)
	}
	var users []map[string]any
	r = e.get("/api/users", admin)
	r.json(t, &users)
	if len(users) != 2 {
		t.Fatalf("users: %s", r.body)
	}
	if r := e.get("/api/users/"+itoa(e.k.PlainUser.RID), admin); r.status != http.StatusOK || !strings.Contains(string(r.body), `"nina"`) {
		t.Fatalf("user: %d %s", r.status, r.body)
	}
	if r := e.get("/api/users/999", admin); r.status != http.StatusNotFound {
		t.Fatalf("user 404: %d", r.status)
	}
	if r := e.get("/api/nada", admin); r.status != http.StatusNotFound || detail(t, r) != "Not Found" {
		t.Fatalf("404: %d %s", r.status, r.body)
	}
}

func TestStatsCollectionsTasks(t *testing.T) {
	e := newEnv(t)
	admin, nina := e.admin(), e.nina()
	var st map[string]int64
	r := e.get("/api/stats", nina)
	r.json(t, &st)
	if st["PLATFORMS"] != 2 || st["ROMS"] != 3 || st["TOTAL_FILESIZE_BYTES"] == 0 {
		t.Fatalf("stats: %s", r.body)
	}
	for _, p := range []string{"/api/collections", "/api/collections/virtual?type=collection", "/api/collections/smart"} {
		if r := e.get(p, nina); r.status != http.StatusOK || strings.TrimSpace(string(r.body)) != "[]" {
			t.Fatalf("%s: %d %s", p, r.status, r.body)
		}
	}
	if r := e.get("/api/collections/1", nina); r.status != http.StatusNotFound {
		t.Fatalf("coleção: %d", r.status)
	}
	if r := e.do(http.MethodPost, "/api/tasks/run/scan_library", nina, nil, ""); r.status != http.StatusForbidden {
		t.Fatalf("nina rodando tarefa: %d", r.status)
	}
	if r := e.get("/api/tasks", admin); r.status != http.StatusOK || !strings.Contains(string(r.body), "scan_library") {
		t.Fatalf("tasks: %s", r.body)
	}
	if r := e.do(http.MethodPost, "/api/tasks/scan", admin, nil, ""); r.status != http.StatusAccepted && r.status != http.StatusConflict {
		t.Fatalf("scan: %d %s", r.status, r.body)
	}
	if r := e.get("/api/tasks/status", admin); r.status != http.StatusOK {
		t.Fatalf("status: %d", r.status)
	}
	if r := e.do(http.MethodPost, "/api/tasks/run/cleanup", admin, nil, ""); r.status != http.StatusNotFound {
		t.Fatalf("tarefa inexistente: %d", r.status)
	}
}
