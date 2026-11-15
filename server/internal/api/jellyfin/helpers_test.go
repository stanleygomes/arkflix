package jellyfin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// env é o servidor de teste com o admin logado.
type env struct {
	t      *testing.T
	k      *testkit.Kit
	ids    testkit.SeedIDs
	srv    *httptest.Server
	s      *Server
	token  string
	userID string
}

func setup(t *testing.T) *env {
	t.Helper()
	t.Parallel()
	k := testkit.New(t)
	ids := k.Seed(t)
	s := New(k.Deps).(*Server)
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { s.hls.stopWhere(func(*hlsJob) bool { return true }) })
	e := &env{t: t, k: k, ids: ids, srv: srv, s: s}
	e.token, e.userID = e.login(testkit.Admin, testkit.AdminPass, "aparelho-1")
	return e
}

func authHdr(device, token string) string {
	h := fmt.Sprintf(`MediaBrowser Client="teste", Device="PC", DeviceId=%q, Version="1.0"`, device)
	if token != "" {
		h += fmt.Sprintf(`, Token=%q`, token)
	}
	return h
}

func (e *env) login(user, pass, device string) (string, string) {
	e.t.Helper()
	var res struct {
		AccessToken string
		User        struct{ Id string }
	}
	code := e.call("POST", "/Users/AuthenticateByName", map[string]string{"Username": user, "Pw": pass},
		&res, "Authorization", authHdr(device, ""))
	if code != 200 || res.AccessToken == "" {
		e.t.Fatalf("login %s: %d", user, code)
	}
	return res.AccessToken, res.User.Id
}

// req faz a chamada com o token do admin (hdr sobrescreve: pares chave, valor).
func (e *env) req(method, path string, body any, hdr ...string) (*http.Response, []byte) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	r, err := http.NewRequest(method, e.srv.URL+path, rd)
	if err != nil {
		e.t.Fatal(err)
	}
	r.Header.Set("Authorization", authHdr("aparelho-1", e.token))
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		if hdr[i+1] == "" {
			r.Header.Del(hdr[i])
		} else {
			r.Header.Set(hdr[i], hdr[i+1])
		}
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, raw
}

// call decodifica o JSON em out e devolve o status.
func (e *env) call(method, path string, body, out any, hdr ...string) int {
	e.t.Helper()
	resp, raw := e.req(method, path, body, hdr...)
	if out != nil && resp.StatusCode < 300 && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			e.t.Fatalf("%s %s: %v\n%s", method, path, err, raw)
		}
	}
	return resp.StatusCode
}

// get exige 200 e decodifica.
func (e *env) get(path string, out any) {
	e.t.Helper()
	if code := e.call("GET", path, nil, out); code != 200 {
		_, raw := e.req("GET", path, nil)
		e.t.Fatalf("GET %s: %d %s", path, code, raw)
	}
}

type dtoT struct {
	Id, Name, Type, MediaType, ServerId, SeriesId, SeriesName, SeasonId, SeasonName, ParentId string
	Album, AlbumId, AlbumArtist, CollectionType, Container, Path, SeriesPrimaryImageTag       string
	AlbumPrimaryImageTag, PremiereDate, DateCreated                                           string
	IndexNumber, ParentIndexNumber, ChildCount, RecursiveItemCount                            *int
	ProductionYear                                                                            int
	RunTimeTicks                                                                              int64
	IsFolder                                                                                  bool
	Genres                                                                                    []string
	ImageTags                                                                                 map[string]string
	BackdropImageTags                                                                         []string
	AlbumArtists, ArtistItems                                                                 []nameID
	MediaSources                                                                              []sourceDto
	UserData                                                                                  struct {
		PlaybackPositionTicks int64
		PlayCount             int
		IsFavorite, Played    bool
		UnplayedItemCount     *int
		Key, ItemId           string
	}
}

type listT struct {
	Items            []dtoT
	TotalRecordCount int
	StartIndex       int
}

func names(items []dtoT) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func (e *env) list(path string) listT {
	e.t.Helper()
	var l listT
	e.get(path, &l)
	return l
}

func eq[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", what, got, want)
	}
}

func joined(v []string) string { return strings.Join(v, "|") }
