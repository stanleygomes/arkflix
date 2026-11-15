package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// fake é a internet de mentira: TMDB, libretro, MusicBrainz e Cover Art
// Archive num servidor só, com respostas no formato real.
type fake struct {
	*httptest.Server
	mu     sync.Mutex
	hits   []string // caminhos pedidos
	enPlot string   // sinopse em inglês do filme 11 (Refresh troca)
	apiKey string   // api_key recebida
	bearer string   // Authorization recebido
}

func (fk *fake) count(prefix string) int {
	fk.mu.Lock()
	defer fk.mu.Unlock()
	n := 0
	for _, h := range fk.hits {
		if strings.HasPrefix(h, prefix) {
			n++
		}
	}
	return n
}

func pngBytes() []byte {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	return b.Bytes()
}

func jpgBytes() []byte {
	var b bytes.Buffer
	jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil)
	return b.Bytes()
}

func newFake(t *testing.T) *fake {
	fk := &fake{enPlot: "English overview"}
	js := func(w http.ResponseWriter, v string) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(v))
	}
	tmdb := map[string]func(lang string) string{
		"/search/movie": nil, // tratado à parte
		"/movie/11": func(lang string) string {
			if lang == "en-US" {
				return `{"id":11,"title":"Test Movie","overview":` + strconvQuote(fk.enPlot) + `,"tagline":"Tag","genres":[{"id":28,"name":"Action"}]}`
			}
			return `{"id":11,"title":"Filme Teste","overview":"","tagline":"","release_date":"2021-03-04",
"genres":[{"id":28,"name":"Ação"},{"id":18,"name":"Drama"}],"poster_path":"/p11.jpg","backdrop_path":"/b11.jpg",
"vote_average":7.25,"imdb_id":"tt0000011","external_ids":{"imdb_id":"tt0000011"},
"images":{"backdrops":[],"posters":[],"logos":[{"file_path":"/l-en.png","iso_639_1":"en","vote_average":9},
{"file_path":"/l-pt.png","iso_639_1":"pt","vote_average":5},{"file_path":"/l-null.png","iso_639_1":null}]}}`
		},
		"/movie/603": func(string) string {
			return `{"id":603,"title":"Matrix","overview":"Sinopse do TMDB","release_date":"1999-03-31",
"genres":[{"name":"Ficção científica"}],"poster_path":"/p603.jpg","backdrop_path":"/b603.jpg","vote_average":8.2,
"imdb_id":"tt0133093","images":{"logos":[]}}`
		},
		"/tv/77": func(string) string {
			return `{"id":77,"name":"Série X","overview":"Sinopse da série","first_air_date":"2010-05-01",
"genres":[{"name":"Drama"}],"poster_path":"/p77.jpg","backdrop_path":"/b77.jpg","vote_average":8,"tagline":"",
"external_ids":{"imdb_id":"tt77"},"images":{"logos":[{"file_path":"/l77.png","iso_639_1":"en"}]}}`
		},
		"/tv/77/season/1": func(lang string) string {
			if lang == "en-US" {
				return `{"overview":"Season one","episodes":[{"episode_number":1,"name":"Pilot","overview":"Ep one"},
{"episode_number":2,"name":"Second","overview":"Ep two in English"}]}`
			}
			return `{"_id":"x","air_date":"2010-05-01","name":"Temporada 1","overview":"Temporada um","poster_path":"/s1.jpg",
"episodes":[{"episode_number":1,"name":"Piloto","overview":"Ep um","still_path":"/e1.jpg","air_date":"2010-05-01","vote_average":8},
{"episode_number":2,"name":"Episódio 2","overview":"","still_path":null,"air_date":"2010-05-08"}]}`
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/tmdb/", func(w http.ResponseWriter, r *http.Request) {
		fk.mu.Lock()
		fk.apiKey, fk.bearer = r.URL.Query().Get("api_key"), r.Header.Get("Authorization")
		fk.mu.Unlock()
		p := strings.TrimPrefix(r.URL.Path, "/tmdb")
		lang := r.URL.Query().Get("language")
		switch p {
		case "/search/movie":
			switch r.URL.Query().Get("query") {
			case "Filme Teste":
				js(w, `{"page":1,"results":[{"id":10,"title":"Filme Teste Errado","release_date":"2020-01-01"},
{"id":12,"title":"Filme Teste","original_title":"Filme Teste","release_date":"1990-01-01"},
{"id":11,"title":"Filme Teste","original_title":"Test Movie","release_date":"2021-03-04"}],"total_results":3}`)
			case "Matrix":
				js(w, `{"page":1,"results":[{"id":603,"title":"Matrix","original_title":"The Matrix","release_date":"1999-03-31"}]}`)
			default:
				js(w, `{"page":1,"results":[],"total_results":0}`)
			}
			return
		case "/search/tv":
			if r.URL.Query().Get("query") == "Série X" {
				js(w, `{"results":[{"id":77,"name":"Série X","original_name":"Series X","first_air_date":"2010-05-01"}]}`)
				return
			}
			js(w, `{"results":[]}`)
			return
		}
		if h := tmdb[p]; h != nil {
			js(w, h(lang))
			return
		}
		http.Error(w, `{"status_code":34}`, http.StatusNotFound)
	})
	mux.HandleFunc("/img/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".png") {
			w.Write(pngBytes())
			return
		}
		w.Write(jpgBytes())
	})
	retro := map[string]bool{
		"/libretro/Nintendo - Super Nintendo Entertainment System/Named_Boxarts/Super Teste (USA).png":   true,
		"/libretro/Nintendo - Super Nintendo Entertainment System/Named_Snaps/Super Teste (USA).png":     true,
		"/libretro/Nintendo - Super Nintendo Entertainment System/Named_Boxarts/Outro Jogo (Europe).png": true,
		"/libretro/Nintendo - Super Nintendo Entertainment System/Named_Titles/Outro Jogo (Europe).png":  true,
		"/libretro/Sony - PlayStation/Named_Boxarts/Jogo_ Disco (USA).png":                               true,
	}
	mux.HandleFunc("/libretro/", func(w http.ResponseWriter, r *http.Request) {
		if retro[r.URL.Path] {
			w.Write(pngBytes())
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/mb/ws/2/release/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "arkflix-server/0.0.0-teste ( https://github.com/") {
			http.Error(w, "sem user-agent", http.StatusForbidden)
			return
		}
		if q := r.URL.Query().Get("query"); q == `release:"Disco" AND artist:"Banda"` {
			js(w, `{"created":"2026-01-01T00:00:00.000Z","count":2,"offset":0,"releases":[
{"id":"mb-other","score":100,"title":"Disco (Deluxe)","artist-credit":[{"name":"Banda","artist":{"id":"a1","name":"Banda"}}],"release-group":{"id":"rg-x"}},
{"id":"mb-1","score":98,"title":"Disco","status":"Official","artist-credit":[{"name":"Banda","artist":{"id":"a1","name":"Banda"}}],
"release-group":{"id":"rg-1","primary-type":"Album"},"date":"2001"}]}`)
			return
		}
		js(w, `{"count":0,"releases":[]}`)
	})
	mux.HandleFunc("/caa/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/caa/release-group/rg-1/front-500":
			http.Redirect(w, r, "/img/caa/front-500.jpg", http.StatusTemporaryRedirect)
		default:
			http.NotFound(w, r)
		}
	})
	fk.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fk.mu.Lock()
		fk.hits = append(fk.hits, r.URL.Path)
		fk.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(fk.Close)
	return fk
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// env monta pastas de mídia de mentira, escaneia e devolve o Fetcher
// apontado pro fake.
type env struct {
	t    *testing.T
	root string
	cfg  *config.Config
	st   *store.Store
	lib  *library.Library
	f    *Fetcher
	fk   *fake
}

var files = map[string]string{
	"Movies/Filme Teste (2020)/Filme Teste (2020).mkv":                "video",
	"Movies/Matrix (1999)/Matrix (1999).mkv":                          "video",
	"Movies/Matrix (1999)/poster.jpg":                                 "jpg",
	"Movies/Matrix (1999)/movie.nfo":                                  `<movie><title>Matrix</title><plot>Plot local</plot></movie>`,
	"Movies/Inexistente (2001).mkv":                                   "video",
	"Shows/Série X/Season 01/Série X - S01E01.mkv":                    "video",
	"Shows/Série X/Season 01/Série X - S01E02.mkv":                    "video",
	"musicas/Banda/Disco/01 - Faixa.mp3":                              "audio",
	"jogos/roms/snes/Super Teste (USA).sfc":                           "rom",
	"jogos/roms/snes/Outro Jogo (Europe) (En,Fr).sfc":                 "rom",
	"jogos/roms/snes/Sumido (USA).sfc":                                "rom",
	"jogos/roms/psx/Jogo: Disco (USA)/Jogo: Disco (USA) (Disc 1).bin": "rom",
}

func newEnv(t *testing.T, key string) *env {
	t.Helper()
	root := t.TempDir()
	cfg := &config.Config{Data: filepath.Join(root, "data"), Movies: filepath.Join(root, "Movies"),
		Shows: filepath.Join(root, "Shows"), Music: filepath.Join(root, "musicas"), Games: filepath.Join(root, "jogos"),
		TMDBKey: key, Metadata: true, Language: "pt-BR"}
	for rel, content := range files {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		data := []byte(content)
		if content == "jpg" {
			data = jpgBytes()
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	lib := library.New(cfg, st)
	if err := lib.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	fk := newFake(t)
	f := New(cfg, st)
	f.Version = "0.0.0-teste"
	f.urls = endpoints{tmdb: fk.URL + "/tmdb", tmdbImg: fk.URL + "/img", libretro: fk.URL + "/libretro",
		musicbrainz: fk.URL + "/mb", coverart: fk.URL + "/caa"}
	f.http.backoff, f.http.maxWait = time.Millisecond, 5*time.Millisecond
	for _, l := range []*limiter{f.lim.tmdb, f.lim.libretro, f.lim.mb, f.lim.caa} {
		l.every = 0
	}
	return &env{t: t, root: root, cfg: cfg, st: st, lib: lib, f: f, fk: fk}
}

func (e *env) item(kind, rel string) *store.Item {
	e.t.Helper()
	items, err := e.st.Items(context.Background(), `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = ? AND (i.rel_path = ? OR i.path = ?)`,
		kind, rel, rel)
	if err != nil || len(items) != 1 {
		e.t.Fatalf("%s %s: %v (%d)", kind, rel, err, len(items))
	}
	return items[0]
}

func (e *env) run() {
	e.t.Helper()
	if err := e.f.Run(context.Background()); err != nil {
		e.t.Fatal(err)
	}
}

func fields(it *store.Item) []string { return readMeta(it).Fields }

func TestRunFills(t *testing.T) {
	e := newEnv(t, "chave-v3")
	e.run()
	if e.fk.apiKey != "chave-v3" || e.fk.bearer != "" {
		t.Fatalf("auth: %q %q", e.fk.apiKey, e.fk.bearer)
	}

	mv := e.item(store.KindMovie, "Filme Teste (2020)/Filme Teste (2020).mkv")
	if mv.Overview != "English overview" { // pt-BR vazio: cai pro inglês
		t.Errorf("overview %q", mv.Overview)
	}
	if !slices.Equal(mv.Genres, []string{"Ação", "Drama"}) || mv.Premiere != "2021-03-04" || mv.Year != 2020 {
		t.Errorf("genres/premiere/year: %v %q %d", mv.Genres, mv.Premiere, mv.Year)
	}
	dir := filepath.Join(e.cfg.Data, "metadata", mv.ID)
	for field, want := range map[string]string{"image": "poster.jpg", "backdrop": "backdrop.jpg", "logo": "logo.png"} {
		got := map[string]string{"image": mv.Image, "backdrop": mv.Backdrop, "logo": mv.Logo}[field]
		if got != filepath.Join(dir, want) {
			t.Errorf("%s = %q", field, got)
		}
		if _, err := os.Stat(got); err != nil {
			t.Errorf("%s: %v", field, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "poster.jpg.part")); err == nil {
		t.Error("sobrou .part")
	}
	if e.fk.count("/img/w500/l-pt.png") != 1 { // logo no idioma ganha do inglês
		t.Errorf("logo errado: %v", e.fk.hits)
	}
	if mv.Extra["tmdb_id"] != float64(11) || mv.Extra["imdb_id"] != "tt0000011" || mv.Extra["rating"] != 7.3 ||
		mv.Extra["tagline"] != "Tag" {
		t.Errorf("extra %v", mv.Extra)
	}
	m := readMeta(mv)
	if m.Source != "tmdb" || m.Fetched == "" || m.Missed != "" || slices.Contains(m.Fields, "year") {
		t.Errorf("meta %+v", m)
	}

	ser := e.item(store.KindSeries, "Série X")
	if ser.Overview != "Sinopse da série" || ser.Extra["imdb_id"] != "tt77" || ser.Logo == "" || ser.Year != 2010 {
		t.Errorf("série %+v", ser)
	}
	sea := e.item(store.KindSeason, "Série X/Season 01")
	if sea.Overview != "Temporada um" || filepath.Base(sea.Image) != "poster.jpg" {
		t.Errorf("temporada %+v", sea)
	}
	ep1 := e.item(store.KindEpisode, "Série X/Season 01/Série X - S01E01.mkv")
	if ep1.Name != "Piloto" || ep1.Overview != "Ep um" || filepath.Base(ep1.Image) != "still.jpg" || ep1.SortName != "piloto" {
		t.Errorf("ep1 %+v", ep1)
	}
	ep2 := e.item(store.KindEpisode, "Série X/Season 01/Série X - S01E02.mkv")
	if ep2.Name != "Second" || ep2.Overview != "Ep two in English" || ep2.Image != "" {
		t.Errorf("ep2 %q %q %q", ep2.Name, ep2.Overview, ep2.Image)
	}
	if n := e.fk.count("/tmdb/tv/77/season/1"); n != 2 { // pt-BR + en-US, uma vez cada
		t.Errorf("temporada pedida %d vezes", n)
	}

	al := e.item(store.KindAlbum, "album:banda/disco")
	if al.Extra["mbid"] != "mb-1" || filepath.Base(al.Image) != "poster.jpg" {
		t.Errorf("álbum %q %v", al.Image, al.Extra)
	}

	rom := e.item(store.KindRom, "roms/snes/Super Teste (USA).sfc")
	if filepath.Base(rom.Image) != "poster.png" || filepath.Base(rom.Backdrop) != "backdrop.png" {
		t.Errorf("rom %q %q", rom.Image, rom.Backdrop)
	}
	if rom.Extra["fs_name"] != "Super Teste (USA).sfc" {
		t.Errorf("mexeu no extra do disco: %v", rom.Extra)
	}
	outro := e.item(store.KindRom, "roms/snes/Outro Jogo (Europe) (En,Fr).sfc")
	if outro.Image == "" || outro.Backdrop == "" { // achou sem as tags, fundo pelo Named_Titles
		t.Errorf("outro %q %q", outro.Image, outro.Backdrop)
	}
	disco := e.item(store.KindRom, "roms/psx/Jogo: Disco (USA)")
	if disco.Image == "" { // ":" vira "_" no libretro
		t.Errorf("multidisco sem capa")
	}
	sumido := e.item(store.KindRom, "roms/snes/Sumido (USA).sfc")
	if mm := readMeta(sumido); sumido.Image != "" || mm.Missed == "" || mm.Source != "libretro" {
		t.Errorf("sumido %+v", mm)
	}
	if e.fk.count("/libretro/Nintendo - Super Nintendo Entertainment System/Named_Snaps/Sumido") != 0 {
		t.Error("sem capa não devia tentar o fundo")
	}
	miss := e.item(store.KindMovie, "Inexistente (2001).mkv")
	if readMeta(miss).Missed == "" {
		t.Error("filme sem resultado devia ficar marcado")
	}

	s := e.f.Status()
	if s.Running || s.Pending != 0 || s.Done == 0 || s.Error != "" {
		t.Errorf("status %+v", s)
	}
}

// TestLocalWins: nfo e pôster do disco não são trocados, nem no Refresh.
func TestLocalWins(t *testing.T) {
	e := newEnv(t, "k")
	e.run()
	mx := e.item(store.KindMovie, "Matrix (1999)/Matrix (1999).mkv")
	if mx.Overview != "Plot local" || mx.Image != filepath.Join(e.root, "Movies/Matrix (1999)/poster.jpg") {
		t.Fatalf("local perdeu: %q %q", mx.Overview, mx.Image)
	}
	if !strings.HasPrefix(mx.Backdrop, e.cfg.Data) || mx.Genres[0] != "Ficção científica" {
		t.Fatalf("faltou o que não tinha no disco: %q %v", mx.Backdrop, mx.Genres)
	}
	if fs := fields(mx); slices.Contains(fs, "overview") || slices.Contains(fs, "image") {
		t.Fatalf("fields %v", fs)
	}
	if err := e.f.Refresh(context.Background(), mx.ID); err != nil {
		t.Fatal(err)
	}
	if mx = e.item(store.KindMovie, "Matrix (1999)/Matrix (1999).mkv"); mx.Overview != "Plot local" {
		t.Fatalf("refresh trocou o local: %q", mx.Overview)
	}
}

// TestRescanKeeps: arquivo que muda não perde o que veio da internet;
// imagem nova no disco ganha e sai da lista.
func TestRescanKeeps(t *testing.T) {
	e := newEnv(t, "k")
	e.run()
	ctx := context.Background()
	rel := "Filme Teste (2020)/Filme Teste (2020).mkv"
	before := e.item(store.KindMovie, rel)
	ep := e.item(store.KindEpisode, "Série X/Season 01/Série X - S01E01.mkv")

	later := time.Now().Add(time.Hour)
	for _, p := range []string{before.Path, ep.Path} {
		os.WriteFile(p, []byte("mudou de tamanho"), 0o644)
		os.Chtimes(p, later, later)
	}
	if err := e.lib.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	after := e.item(store.KindMovie, rel)
	if after.Size == before.Size {
		t.Fatal("rescan não regravou")
	}
	if after.Overview != before.Overview || after.Backdrop != before.Backdrop || after.Logo != before.Logo ||
		!slices.Equal(after.Genres, before.Genres) || after.Extra["tmdb_id"] != float64(11) || readMeta(after).Fetched == "" {
		t.Fatalf("rescan perdeu metadados:\n%+v\n%+v", before, after)
	}
	if ep2 := e.item(store.KindEpisode, "Série X/Season 01/Série X - S01E01.mkv"); ep2.Name != "Piloto" || ep2.SortName != "piloto" {
		t.Fatalf("episódio voltou pro nome genérico: %q", ep2.Name)
	}
	// rescan sem mudança não regrava nada
	if err := e.lib.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if again := e.item(store.KindMovie, rel); !again.UpdatedAt.Equal(after.UpdatedAt) {
		t.Fatal("rescan sem mudança regravou")
	}

	fan := filepath.Join(e.root, "Movies/Filme Teste (2020)/fanart.jpg")
	os.WriteFile(fan, jpgBytes(), 0o644)
	if err := e.lib.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	local := e.item(store.KindMovie, rel)
	if local.Backdrop != fan || slices.Contains(fields(local), "backdrop") || local.Logo != before.Logo {
		t.Fatalf("disco devia ganhar: %q %v", local.Backdrop, fields(local))
	}
}

// TestRetryWindow: o que achou não é buscado de novo; o que falhou só
// depois de um dia.
func TestRetryWindow(t *testing.T) {
	e := newEnv(t, "k")
	now := time.Now()
	e.f.now = func() time.Time { return now }
	e.run()
	first := len(e.fk.hits)
	sumido := "/libretro/Nintendo - Super Nintendo Entertainment System/Named_Boxarts/Sumido (USA).png"
	if e.fk.count(sumido) != 1 {
		t.Fatalf("sumido pedido %d vezes", e.fk.count(sumido))
	}
	e.run()
	if len(e.fk.hits) != first {
		t.Fatalf("segundo ciclo pediu de novo: %v", e.fk.hits[first:])
	}
	now = now.Add(25 * time.Hour)
	e.run()
	if e.fk.count(sumido) != 2 {
		t.Fatal("falha devia ser tentada de novo depois de um dia")
	}
	if e.fk.count("/tmdb/search/movie") == 0 || e.fk.count("/libretro/Sony") == 0 {
		t.Fatal("?")
	}
	// o que achou fica quieto mesmo depois de um dia
	for _, h := range e.fk.hits[first:] {
		if strings.Contains(h, "Super Teste") || strings.HasPrefix(h, "/mb/") || strings.Contains(h, "/movie/11") {
			al := e.item(store.KindAlbum, "album:banda/disco")
			t.Fatalf("buscou de novo o que já tinha achado: %s %v %q", h, al.Extra, al.Image)
		}
	}
}

func TestRefreshOverwritesInternet(t *testing.T) {
	e := newEnv(t, "k")
	e.run()
	mv := e.item(store.KindMovie, "Filme Teste (2020)/Filme Teste (2020).mkv")
	e.fk.mu.Lock()
	e.fk.enPlot = "Sinopse nova"
	e.fk.mu.Unlock()
	if err := e.f.Refresh(context.Background(), mv.ID); err != nil {
		t.Fatal(err)
	}
	if mv = e.item(store.KindMovie, mv.RelPath); mv.Overview != "Sinopse nova" {
		t.Fatalf("overview %q", mv.Overview)
	}
	if err := e.f.Refresh(context.Background(), "nao-existe"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("err %v", err)
	}
	tr, _ := e.st.Items(context.Background(), `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = 'track'`)
	if err := e.f.Refresh(context.Background(), tr[0].ID); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("faixa: %v", err)
	}
}

func TestDisabled(t *testing.T) {
	e := newEnv(t, "k")
	e.cfg.Metadata = false
	e.run()
	mv := e.item(store.KindMovie, "Filme Teste (2020)/Filme Teste (2020).mkv")
	if len(e.fk.hits) != 0 {
		t.Fatalf("desligado e pediu: %v", e.fk.hits)
	}
	if err := e.f.Refresh(context.Background(), mv.ID); !errors.Is(err, ErrDisabled) {
		t.Fatalf("err %v", err)
	}
}

// TestNoTMDBKey: sem chave, filmes e séries ficam como estão; jogos e
// música continuam.
func TestNoTMDBKey(t *testing.T) {
	e := newEnv(t, "")
	e.run()
	if n := e.fk.count("/tmdb/"); n != 0 {
		t.Fatalf("sem chave e chamou o TMDB %d vezes", n)
	}
	if mv := e.item(store.KindMovie, "Filme Teste (2020)/Filme Teste (2020).mkv"); mv.Extra["meta"] != nil {
		t.Fatalf("filme marcado: %v", mv.Extra)
	}
	if rom := e.item(store.KindRom, "roms/snes/Super Teste (USA).sfc"); rom.Image == "" {
		t.Fatal("rom sem capa")
	}
}

func TestBearerToken(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiJ9." + strings.Repeat("a", 120) + ".assinatura"
	e := newEnv(t, jwt)
	e.run()
	if e.fk.bearer != "Bearer "+jwt || e.fk.apiKey != "" {
		t.Fatalf("auth: %q %q", e.fk.bearer, e.fk.apiKey)
	}
}

func TestCanceled(t *testing.T) {
	e := newEnv(t, "k")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.f.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v", err)
	}
	if s := e.f.Status(); s.Running {
		t.Fatal("ficou rodando")
	}
}

// TestBackoff: 429/5xx tentam de novo com espera crescente; 404 não.
func TestBackoff(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.URL.Path]++
		n := calls[r.URL.Path]
		mu.Unlock()
		switch {
		case r.URL.Path == "/limite" && n <= 2:
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
		case r.URL.Path == "/caiu" && n == 1:
			w.WriteHeader(http.StatusBadGateway)
		case r.URL.Path == "/sempre":
			w.WriteHeader(http.StatusServiceUnavailable)
		case r.URL.Path == "/nada":
			http.NotFound(w, r)
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer srv.Close()
	g := &getter{client: srv.Client(), ua: func() string { return "teste" }, tries: 4, backoff: time.Millisecond, maxWait: 10 * time.Millisecond}
	lim := &limiter{}
	ctx := context.Background()
	for _, p := range []string{"/limite", "/caiu"} {
		var v map[string]bool
		if err := g.getJSON(ctx, lim, srv.URL+p, nil, &v); err != nil || !v["ok"] {
			t.Fatalf("%s: %v", p, err)
		}
	}
	if _, err := g.get(ctx, lim, srv.URL+"/sempre", nil); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("sempre: %v", err)
	}
	if _, err := g.get(ctx, lim, srv.URL+"/nada", nil); !errors.Is(err, errNotFound) {
		t.Fatalf("nada: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls["/limite"] != 3 || calls["/caiu"] != 2 || calls["/sempre"] != 4 || calls["/nada"] != 1 {
		t.Fatalf("chamadas %v", calls)
	}
}

func TestLimiter(t *testing.T) {
	l := &limiter{every: 20 * time.Millisecond}
	start := time.Now()
	for range 5 {
		if err := l.wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d < 80*time.Millisecond {
		t.Fatalf("5 pedidos em %v", d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	l.next = time.Now().Add(time.Hour)
	if err := l.wait(ctx); err == nil {
		t.Fatal("devia respeitar o cancelamento")
	}
}

func TestPickResult(t *testing.T) {
	rs := []tmdbResult{
		{ID: 1, Title: "O Poderoso Chefão", OriginalTitle: "The Godfather", ReleaseDate: "1972-03-14"},
		{ID: 2, Title: "O Poderoso Chefão: Parte II", OriginalTitle: "The Godfather Part II", ReleaseDate: "1974-12-20"},
	}
	for _, c := range []struct {
		title string
		year  int
		want  int
	}{
		{"The Godfather", 1972, 1},
		{"o poderoso chefao", 1973, 1}, // ano ±1, sem acento
		{"The Godfather", 1980, 0},     // ano longe
		{"The Godfather Part II", 0, 2},
		{"Godfather", 1974, 2}, // sem título igual: ano exato e um contém o outro
		{"Godfather", 0, 0},
	} {
		if got := pickResult(rs, c.title, c.year); got != c.want {
			t.Errorf("%q %d = %d, quer %d", c.title, c.year, got, c.want)
		}
	}
}

func TestRomNames(t *testing.T) {
	it := &store.Item{Name: "Zelda", Path: "/r/snes/Zelda (USA) (Rev 1) [!].sfc",
		Extra: map[string]any{"fs_name": "Zelda (USA) (Rev 1) [!].sfc", "multi": false,
			"files": []any{map[string]any{"name": "Zelda (USA) (Rev 1) [!].sfc", "rel": "Zelda (USA) (Rev 1) [!].sfc"}}}}
	got := romNames(it)
	want := []string{"Zelda (USA) (Rev 1) [!]", "Zelda (USA) (Rev 1)", "Zelda (USA)", "Zelda (World)",
		"Zelda (USA, Europe)", "Zelda (Europe)", "Zelda (Japan)"}
	if !slices.Equal(got, want) {
		t.Fatalf("%q", got)
	}
	if s := libretroSafe.Replace(`A&B: C/D?`); s != "A_B_ C_D_" {
		t.Fatalf("%q", s)
	}
}
