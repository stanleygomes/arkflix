package jellyfin_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/app/internal/fixture"
	"github.com/stanleygomes/arkflix/app/internal/jellyfin"
)

func TestNormalizeBase(t *testing.T) {
	ok := map[string]string{
		"jellyfin.exemplo.org":        "https://jellyfin.exemplo.org",
		"https://filmes.casa.org/jf/": "https://filmes.casa.org/jf",
		"http://192.168.0.10:8096":    "http://192.168.0.10:8096",
		"http://nas.local:8096/":      "http://nas.local:8096",
		"http://localhost:8096":       "http://localhost:8096",
	}
	for in, want := range ok {
		if got, err := jellyfin.NormalizeBase(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "http://jellyfin.exemplo.org", "http://8.8.8.8", "ftp://x", "https://"} {
		if _, err := jellyfin.NormalizeBase(bad); err == nil {
			t.Errorf("devia recusar %q", bad)
		}
	}
}

func login(t *testing.T, srv *httptest.Server) *jellyfin.API {
	t.Helper()
	acc, err := jellyfin.Login(context.Background(), srv.URL, "stanley", "senha123", jellyfin.Account{DeviceID: "dev1", Device: "pc", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	return &jellyfin.API{Account: acc}
}

func TestLoginAndCatalog(t *testing.T) {
	fake := fixture.NewFakeJellyfin()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()

	if _, err := jellyfin.Login(ctx, srv.URL, "stanley", "errada", jellyfin.Account{DeviceID: "d"}); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	c := login(t, srv)
	if c.Token != "tok-stanley" || c.UserID != "u-stanley" || c.Server != "Jellyfin de Teste" || c.User != "stanley" {
		t.Fatalf("conta: %+v", c.Account)
	}

	resume, err := c.Resume(ctx, 10)
	if err != nil || len(resume) != 1 || resume[0].ID != "m1" || resume[0].Position != 1800 || resume[0].Runtime != 7200 {
		t.Fatalf("continuar: %+v %v", resume, err)
	}
	if resume[0].Backdrop == "" || resume[0].BackdropItem != "m1" || resume[0].Logo == "" {
		t.Fatalf("imagens: %+v", resume[0])
	}
	next, err := c.NextUp(ctx, 10)
	if err != nil || len(next) != 1 || next[0].SeriesName != "Série de Teste" || next[0].Season != 1 || next[0].Episode != 2 {
		t.Fatalf("próximo: %+v %v", next, err)
	}
	if next[0].BackdropItem != "s1" || next[0].LogoItem != "s1" {
		t.Fatalf("episódio usa as imagens da série: %+v", next[0])
	}
	latest, err := c.Latest(ctx, "Movie", 10)
	if err != nil || len(latest) != 3 || latest[0].ID != "m1" {
		t.Fatalf("recentes: %+v %v", latest, err)
	}
	movies, total, err := c.Items(ctx, jellyfin.Query{Types: "Movie", Sort: "name", Limit: 2})
	if err != nil || total != 3 || len(movies) != 2 || movies[0].Name != "Curta Pixel" {
		t.Fatalf("filmes: %+v %d %v", movies, total, err)
	}
	found, _, err := c.Items(ctx, jellyfin.Query{Types: "Movie,Series", Search: "série"})
	if err != nil || len(found) != 1 || found[0].Type != "Series" || found[0].Unplayed != 3 {
		t.Fatalf("busca: %+v %v", found, err)
	}
	seasons, err := c.Seasons(ctx, "s1")
	if err != nil || len(seasons) != 2 {
		t.Fatalf("temporadas: %+v %v", seasons, err)
	}
	eps, err := c.Episodes(ctx, "s1", seasons[0].ID)
	if err != nil || len(eps) != 3 || !eps[0].Played {
		t.Fatalf("episódios: %+v %v", eps, err)
	}

	if err := c.SetFavorite(ctx, "m3", true); err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Item(ctx, "m3"); !it.Favorite {
		t.Fatalf("favorito: %+v", it)
	}
	favs, _, _ := c.Items(ctx, jellyfin.Query{Types: "Movie,Series", Favorite: true})
	if len(favs) != 2 {
		t.Fatalf("favoritos: %+v", favs)
	}
	if err := c.SetPlayed(ctx, "e2", true); err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Item(ctx, "e2"); !it.Played {
		t.Fatalf("assistido: %+v", it)
	}

	if _, err := (&jellyfin.API{Account: jellyfin.Account{Base: srv.URL, UserID: "u-stanley", Token: "velho"}}).Resume(ctx, 1); err != jellyfin.ErrAuth {
		t.Fatalf("token revogado: %v", err)
	}
	if err := c.Logout(ctx); err != nil || fake.Logouts() != 1 {
		t.Fatalf("logout: %v %d", err, fake.Logouts())
	}
}

func TestPlayAndReport(t *testing.T) {
	fake := fixture.NewFakeJellyfin()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()
	c := login(t, srv)

	st, err := c.Play(ctx, "m1", -1, 1800)
	if err != nil {
		t.Fatal(err)
	}
	if st.HLS || !strings.HasPrefix(st.Path, "/Videos/m1/stream?") || !strings.Contains(st.Path, "static=true") || st.Session != "sess-m1" {
		t.Fatalf("direto: %+v", st)
	}
	if len(st.Audio) != 2 || st.AudioIdx != 1 || len(st.Subtitles) != 1 || !strings.HasSuffix(st.Subtitles[0].URL, ".vtt") {
		t.Fatalf("faixas: %+v", st)
	}
	other, err := c.Play(ctx, "m1", 2, 0)
	if err != nil || other.AudioIdx != 2 || !strings.Contains(other.Path, "AudioStreamIndex=2") {
		t.Fatalf("outro áudio vira transcodificação: %+v %v", other, err)
	}

	for _, ev := range []string{"start", "progress", "stop"} {
		if err := c.Report(ctx, ev, "m1", st.Session, st.Source, 2000, false); err != nil {
			t.Fatal(err)
		}
	}
	if got := fake.Reports(); !slices.Equal(got, []string{"start m1 2000.0", "progress m1 2000.0", "stop m1 2000.0"}) {
		t.Fatalf("relatórios: %v", got)
	}
	if it, _ := fake.Item("m1"); it.Position != 2000 {
		t.Fatalf("posição salva: %+v", it)
	}
	if err := c.Report(ctx, "pause", "m1", "", "", 0, true); err == nil {
		t.Fatal("evento desconhecido devia falhar")
	}
}

func TestProxy(t *testing.T) {
	fake := fixture.NewFakeJellyfin()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := login(t, srv)
	linked := true
	p := &jellyfin.Proxy{Account: func() (jellyfin.Account, bool) { return c.Account, linked }}
	defer p.Close()
	base, err := p.URL()
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := p.URL(); again != base {
		t.Fatal("o proxy liga uma vez só")
	}

	get := func(method, url string, hdr map[string]string) *http.Response {
		req, _ := http.NewRequest(method, url, nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	resp := get("GET", base+"/Items/m1/Images/Primary?maxHeight=300", nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" || len(body) == 0 || len(resp.Header.Values("Access-Control-Allow-Origin")) != 1 || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("imagem: %d %v", resp.StatusCode, resp.Header)
	}
	if auths := fake.Auths(); len(auths) != 1 || !strings.Contains(auths[0], `Token="tok-stanley"`) {
		t.Fatalf("o proxy põe o token: %v", auths)
	}

	resp = get("GET", base+"/Videos/m1/stream?static=true", map[string]string{"Range": "bytes=0-99"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent || resp.ContentLength != 100 {
		t.Fatalf("vídeo com Range: %d %d", resp.StatusCode, resp.ContentLength)
	}

	root := base[:strings.LastIndex(base, "/")]
	for _, bad := range []struct{ method, url string }{
		{"GET", root + "/outro-segredo/Items/m1/Images/Primary"},
		{"GET", base + "/Users/u-stanley/Items"},
		{"GET", base + "/Sessions/Logout"},
		{"POST", base + "/Items/m1/PlaybackInfo"},
		{"GET", base + "/Items/../Users/u-stanley/Items"},
	} {
		resp := get(bad.method, bad.url, nil)
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s: devia recusar, veio %d", bad.method, bad.url, resp.StatusCode)
		}
	}

	linked = false
	resp = get("GET", base+"/Items/m1/Images/Primary", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("sem conta: %d", resp.StatusCode)
	}
}
