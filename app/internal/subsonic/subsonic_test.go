package subsonic_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/fixture"
	"github.com/stanleygomes/arkflix/app/internal/subsonic"
)

func login(t *testing.T) (*subsonic.API, *fixture.FakeSubsonic) {
	t.Helper()
	fake := fixture.NewFakeSubsonic()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	acc, err := subsonic.Login(context.Background(), srv.URL, "stanley", "senha123", "")
	if err != nil {
		t.Fatal(err)
	}
	return &subsonic.API{Account: acc}, fake
}

func TestLoginKeepsOnlyToken(t *testing.T) {
	fake := fixture.NewFakeSubsonic()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()
	if _, err := subsonic.Login(ctx, srv.URL, "stanley", "errada", ""); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	acc, err := subsonic.Login(ctx, srv.URL, "stanley", "senha123", "")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Token != subsonic.TokenFor("senha123", acc.Salt) || len(acc.Salt) < 6 || acc.Server != "navidrome" || !acc.Has("apiKeyAuthentication") {
		t.Fatalf("conta: %+v", acc)
	}
	for _, q := range fake.Queries() {
		if q.Has("p") {
			t.Fatalf("a senha foi pro servidor: %v", q)
		}
	}
	if _, err := subsonic.Login(ctx, "http://8.8.8.8", "a", "b", ""); err == nil {
		t.Fatal("http fora da rede de casa")
	}
	fake.NoAPIKey = true
	if _, err := subsonic.Login(ctx, srv.URL, "", "", "chave"); err == nil || !strings.Contains(err.Error(), "chave") {
		t.Fatalf("chave sem a extensão: %v", err)
	}
}

func TestBrowse(t *testing.T) {
	c, fake := login(t)
	ctx := context.Background()
	idx, err := c.Artists(ctx)
	if err != nil || len(idx) != 2 || idx[0].Name != "B" || idx[0].Artists[0].Name != "Banda Teste" {
		t.Fatalf("artistas: %+v %v", idx, err)
	}
	ar, albums, err := c.Artist(ctx, "ar1")
	if err != nil || ar.Name != "Banda Teste" || len(albums) != 2 {
		t.Fatalf("artista: %+v %+v %v", ar, albums, err)
	}
	if top, err := c.TopSongs(ctx, "Orquestra Pixel", 5); err != nil || len(top) != 2 || top[0].ID != "t4" {
		t.Fatalf("mais tocadas: %+v %v", top, err)
	}
	al, songs, err := c.Album(ctx, "al1")
	if err != nil || al.Name != "Disco Teste" || len(songs) != 2 || songs[1].Starred == "" || songs[0].Duration != 2 {
		t.Fatalf("álbum: %+v %+v %v", al, songs, err)
	}
	if _, _, err := c.Album(ctx, "nada"); err == nil {
		t.Fatal("álbum que não existe")
	}
	newest, err := c.AlbumList2(ctx, "newest", 2, 0)
	if err != nil || len(newest) != 2 || newest[0].ID != "al2" {
		t.Fatalf("novos: %+v %v", newest, err)
	}
	if found, err := c.Search(ctx, "bip", 10); err != nil || len(found.Songs) != 2 {
		t.Fatalf("busca: %+v %v", found, err)
	}
	if err := c.Star(ctx, "album", "al1", true); err != nil || !fake.Starred("al1") {
		t.Fatalf("estrela: %v", err)
	}
	st, err := c.Starred(ctx)
	if err != nil || len(st.Albums) != 2 || len(st.Songs) != 1 {
		t.Fatalf("favoritas: %+v %v", st, err)
	}
	if err := c.Scrobble(ctx, "t1", false, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if got := fake.Scrobbles(); len(got) != 1 || got[0] != "now t1" {
		t.Fatalf("scrobble: %v", got)
	}
	var buf bytes.Buffer
	n, err := c.Download(ctx, "t1", 100, &buf, nil)
	if err != nil || n != int64(len(fixture.ToneMP3)) || !bytes.Equal(buf.Bytes(), fixture.ToneMP3[100:]) {
		t.Fatalf("download do meio: %d %d %v", n, buf.Len(), err)
	}
}

func TestPlaylists(t *testing.T) {
	c, fake := login(t)
	ctx := context.Background()
	p, err := c.CreatePlaylist(ctx, "Nova", []string{"t1", "t2"})
	if err != nil || p.ID == "" || len(p.Entry) != 2 {
		t.Fatalf("criar: %+v %v", p, err)
	}
	name := "Renomeada"
	if err := c.UpdatePlaylist(ctx, p.ID, subsonic.PlaylistChange{Name: &name, Add: []string{"t3"}, Remove: []int{0}}); err != nil {
		t.Fatal(err)
	}
	if got := fake.PlaylistSongs(p.ID); strings.Join(got, ",") != "t2,t3" {
		t.Fatalf("depois de mexer: %v", got)
	}
	if err := c.SetPlaylistSongs(ctx, p.ID, []string{"t3", "t2"}); err != nil || strings.Join(fake.PlaylistSongs(p.ID), ",") != "t3,t2" {
		t.Fatalf("reordenar: %v", err)
	}
	lists, err := c.Playlists(ctx)
	if err != nil || len(lists) != 2 || lists[1].Name != "Renomeada" {
		t.Fatalf("playlists: %+v %v", lists, err)
	}
	if err := c.DeletePlaylist(ctx, p.ID); err != nil || fake.PlaylistSongs(p.ID) != nil {
		t.Fatalf("apagar: %v", err)
	}
	var e *subsonic.Error
	if err := c.DeletePlaylist(ctx, p.ID); !errors.As(err, &e) || e.Code != 70 {
		t.Fatalf("apagar de novo: %v", err)
	}
}

func TestAdmin(t *testing.T) {
	c, _ := login(t)
	ctx := context.Background()
	if u, err := c.GetUser(ctx, "stanley"); err != nil || !u.AdminRole {
		t.Fatalf("usuário: %+v %v", u, err)
	}
	if users, err := c.Users(ctx); err != nil || len(users) != 2 {
		t.Fatalf("usuários: %+v %v", users, err)
	}
	st, err := c.StartScan(ctx)
	if err != nil || !st.Scanning {
		t.Fatalf("scan: %+v %v", st, err)
	}
	if err := c.SaveQueue(ctx, []string{"t1", "t2"}, "t2", 1500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	q, err := c.Queue(ctx)
	if err != nil || q.Current != "t2" || q.Position != 1500 || len(q.Entry) != 2 {
		t.Fatalf("fila: %+v %v", q, err)
	}
	c.Token = "errado"
	if err := c.Ping(ctx); !errors.Is(err, subsonic.ErrAuth) {
		t.Fatalf("token errado: %v", err)
	}
}

func TestProxy(t *testing.T) {
	c, fake := login(t)
	dir := t.TempDir()
	local := filepath.Join(dir, "x.flac")
	os.WriteFile(local, fixture.ToneFLAC, 0o644)
	p := &subsonic.Proxy{
		Account: func() (subsonic.Account, bool) { return c.Account, true },
		Local: func(id string) (string, bool) {
			return local, id == "t3"
		},
	}
	defer p.Close()
	base, err := p.URL()
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string, hdr ...string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, base+path, nil)
		for i := 0; i+1 < len(hdr); i += 2 {
			req.Header.Set(hdr[i], hdr[i+1])
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	resp := get("/rest/stream?id=t1&format=raw&u=intruso&p=senha", "Range", "bytes=0-9")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent || len(body) != 10 || resp.Header.Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("stream: %d %d %s", resp.StatusCode, len(body), resp.Header.Get("Content-Type"))
	}
	last := fake.Queries()[len(fake.Queries())-1]
	if last.Get("u") != "stanley" || last.Has("p") || last.Get("t") == "" || last.Get("id") != "t1" {
		t.Fatalf("o proxy põe a conta (e só ela): %v", last)
	}
	for _, path := range []string{"/rest/ping", "/rest/getUsers.view", "/rest/../rest/stream?id=t1", "/local/t1", "/local/..%2Fx"} {
		r := get(path)
		r.Body.Close()
		if r.StatusCode != http.StatusNotFound {
			t.Fatalf("%s devia ser 404: %d", path, r.StatusCode)
		}
	}
	r := get("/local/t3")
	data, _ := io.ReadAll(r.Body)
	r.Body.Close()
	if r.StatusCode != 200 || !bytes.Equal(data, fixture.ToneFLAC) || r.Header.Get("Content-Type") != "audio/flac" {
		t.Fatalf("cópia local: %d %s", r.StatusCode, r.Header.Get("Content-Type"))
	}
	resp, _ = http.Get(strings.Replace(base, "/"+base[strings.LastIndex(base, "/")+1:], "/outro", 1) + "/rest/stream?id=t1")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("sem o segredo: %d", resp.StatusCode)
	}
	resp.Body.Close()
}
