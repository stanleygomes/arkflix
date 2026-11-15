package subsonic

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

type env struct {
	t   *testing.T
	k   *testkit.Kit
	ids testkit.SeedIDs
	h   http.Handler
}

func setup(t *testing.T) *env {
	t.Helper()
	k := testkit.New(t)
	ids := k.Seed(t)
	return &env{t: t, k: k, ids: ids, h: New(k.Deps)}
}

// q monta parâmetros: pares nome, valor.
func q(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Add(kv[i], kv[i+1])
	}
	return v
}

// login acrescenta u/t/s/v/c do admin (se não vier credencial). Token em
// vez de senha: bcrypt a cada chamada deixa os testes lentos.
func login(v url.Values) url.Values {
	if !v.Has("u") && !v.Has("apiKey") {
		sum := md5.Sum([]byte(testkit.AdminPass + "sal"))
		v.Set("u", testkit.Admin)
		v.Set("t", hex.EncodeToString(sum[:]))
		v.Set("s", "sal")
	}
	if !v.Has("v") {
		v.Set("v", "1.16.1")
	}
	if !v.Has("c") {
		v.Set("c", "teste")
	}
	return v
}

// raw faz a requisição sem mexer nos parâmetros.
func (e *env) raw(method, path string, v url.Values, hdr ...string) *httptest.ResponseRecorder {
	e.t.Helper()
	var r *http.Request
	if method == http.MethodPost {
		r = httptest.NewRequest(method, path, strings.NewReader(v.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, path+"?"+v.Encode(), nil)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		r.Header.Set(hdr[i], hdr[i+1])
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(w, r)
	return w
}

// get chama /rest/<m> logado como admin e devolve o JSON.
func (e *env) get(m string, v url.Values) *resp {
	e.t.Helper()
	v = login(v)
	v.Set("f", "json")
	return decode(e.t, e.raw(http.MethodGet, "/rest/"+m, v))
}

// ok exige status ok.
func (e *env) ok(m string, v url.Values) *resp {
	e.t.Helper()
	r := e.get(m, v)
	if r.Status != "ok" {
		e.t.Fatalf("%s: %+v", m, r.Error)
	}
	return r
}

// failed exige erro com o código.
func (e *env) failed(m string, v url.Values, code int) *resp {
	e.t.Helper()
	r := e.get(m, v)
	if r.Status != "failed" || r.Error == nil || r.Error.Code != code {
		e.t.Fatalf("%s: esperava erro %d, veio %s %+v", m, code, r.Status, r.Error)
	}
	return r
}

func decode(t *testing.T, w *httptest.ResponseRecorder) *resp {
	t.Helper()
	var body map[string]*resp
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v: %s", err, w.Body.String())
	}
	r := body["subsonic-response"]
	if r == nil {
		t.Fatalf("sem subsonic-response: %s", w.Body.String())
	}
	return r
}

// xmlBody chama em XML e devolve o corpo.
func (e *env) xmlBody(m string, v url.Values) string {
	e.t.Helper()
	w := e.raw(http.MethodGet, "/rest/"+m, login(v))
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		e.t.Fatalf("content-type %q", ct)
	}
	return w.Body.String()
}

func titles(cs []child) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Title
	}
	return out
}

func names(as []albumID3) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Name
	}
	return out
}

func eq[T comparable](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %v, esperava %v", what, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: %v, esperava %v", what, got, want)
		}
	}
}
