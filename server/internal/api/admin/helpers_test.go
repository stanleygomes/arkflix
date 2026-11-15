package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
	"github.com/stanleygomes/arkflix/server/internal/web"
)

// env é o servidor inteiro (as quatro APIs) com o acervo semeado e um
// token do Jellyfin pra cada usuário.
type env struct {
	t           *testing.T
	k           *testkit.Kit
	ids         testkit.SeedIDs
	srv         *httptest.Server
	admin, nina string
}

func setup(t *testing.T) *env {
	t.Helper()
	t.Parallel()
	k := testkit.New(t)
	ids := k.Seed(t)
	srv := httptest.NewServer(web.New(k.Deps))
	t.Cleanup(srv.Close)
	e := &env{t: t, k: k, ids: ids, srv: srv}
	e.admin = e.token(k.AdminUser.ID, auth.KindJellyfin, "")
	e.nina = e.token(k.PlainUser.ID, auth.KindJellyfin, "")
	return e
}

// token cria um token direto no banco.
func (e *env) token(userID, kind, prefix string) string {
	e.t.Helper()
	tk, err := e.k.Auth.NewToken(context.Background(), auth.Token{UserID: userID, Kind: kind, Client: "teste",
		Device: "PC", DeviceID: "pc-" + kind + userID[:4], Name: "me.read roms.read collections.read collections.write"}, prefix, 0)
	if err != nil {
		e.t.Fatal(err)
	}
	return tk.Token
}

// mb é o cabeçalho de login do Jellyfin.
func mb(tok string) string {
	return `MediaBrowser Client="teste", Device="PC", DeviceId="pc1", Version="1.0", Token="` + tok + `"`
}

// req faz a chamada. auth: token do Jellyfin (vira MediaBrowser), "Bearer x",
// "Basic x" ou "" (sem login).
func (e *env) req(method, path, auth string, body io.Reader, ctype string) (int, []byte) {
	e.t.Helper()
	r, err := http.NewRequest(method, e.srv.URL+path, body)
	if err != nil {
		e.t.Fatal(err)
	}
	switch {
	case auth == "":
	case strings.HasPrefix(auth, "Bearer ") || strings.HasPrefix(auth, "Basic "):
		r.Header.Set("Authorization", auth)
	default:
		r.Header.Set("Authorization", mb(auth))
	}
	if ctype != "" {
		r.Header.Set("Content-Type", ctype)
	}
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

// call manda body como JSON e decodifica a resposta em out (se 2xx).
func (e *env) call(method, path, auth string, body, out any) int {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	code, raw := e.req(method, path, auth, rd, "application/json")
	if out != nil && code < 300 && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			e.t.Fatalf("%s %s: %v\n%s", method, path, err, raw)
		}
	}
	return code
}

// must exige o status.
func (e *env) must(want int, method, path, auth string, body, out any) {
	e.t.Helper()
	if code := e.call(method, path, auth, body, out); code != want {
		_, raw := e.req(method, path, auth, nil, "")
		e.t.Fatalf("%s %s: %d, queria %d (%s)", method, path, code, want, raw)
	}
}

// errMsg lê {"error": ...}.
func errMsg(t *testing.T, raw []byte) string {
	t.Helper()
	var v struct{ Error string }
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("erro sem JSON: %s", raw)
	}
	return v.Error
}

type field struct{ name, value, file string } // file != "": é arquivo

// multipartBody monta o formulário na ordem dada.
func multipartBody(t *testing.T, fs ...field) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range fs {
		if f.file != "" {
			fw, err := w.CreateFormFile(f.name, f.file)
			if err != nil {
				t.Fatal(err)
			}
			fw.Write([]byte(f.value))
			continue
		}
		w.WriteField(f.name, f.value)
	}
	w.Close()
	return &buf, w.FormDataContentType()
}

// waitScan espera o scan disparado em segundo plano terminar (pra não
// sobrar goroutine mexendo nas pastas depois do teste).
func (e *env) waitScan() {
	e.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	quiet := 0
	for time.Now().Before(deadline) {
		st := e.k.Library.Status()
		if !st.Running && !st.Finished.IsZero() {
			if quiet++; quiet >= 10 {
				return
			}
		} else {
			quiet = 0
		}
		time.Sleep(20 * time.Millisecond)
	}
	e.t.Fatal("scan não terminou")
}
