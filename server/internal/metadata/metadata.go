// Package metadata completa o acervo com dados da internet (TMDB pra
// filmes e séries, libretro pra capas de jogos, MusicBrainz/Cover Art
// Archive pra música). Roda depois de cada scan, só pro que falta, e grava
// direto nas colunas do item: toda API enxerga sem mudar nada. O que veio
// do disco ganha sempre; o que veio da internet fica anotado em
// Extra["meta"] (library.MetaKey) e sobrevive aos rescans.
package metadata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Erros de Refresh.
var (
	ErrDisabled    = errors.New("metadados desligados (--metadata=false)")
	ErrUnsupported = errors.New("tipo de item sem provedor de metadados")
)

const (
	retryMiss  = 24 * time.Hour      // falhou: tenta de novo no outro dia
	retryStale = 30 * 24 * time.Hour // achou mas ainda falta algo: revê no mês
)

// Status é o andamento (o painel de admin mostra).
type Status struct {
	Running bool
	Pending int    // itens ainda sem metadados
	Done    int    // completados no último ciclo
	Error   string // último erro
}

// endpoints são as bases dos provedores (os testes trocam por httptest).
type endpoints struct {
	tmdb, tmdbImg, libretro, musicbrainz, coverart string
}

// Fetcher busca e grava os metadados.
type Fetcher struct {
	cfg *config.Config
	st  *store.Store

	// Version vai no User-Agent (o MusicBrainz pede um que identifique).
	Version string

	urls endpoints
	http *getter
	lim  struct{ tmdb, libretro, mb, caa *limiter }
	now  func() time.Time

	mu     sync.Mutex
	status Status
	again  bool // pediram Run durante um ciclo: roda outro no fim

	work sync.Mutex // um item por vez (Run e Refresh)
}

// New cria o Fetcher.
func New(cfg *config.Config, st *store.Store) *Fetcher {
	f := &Fetcher{cfg: cfg, st: st, Version: buildVersion(), now: time.Now,
		urls: endpoints{
			tmdb:        "https://api.themoviedb.org/3",
			tmdbImg:     "https://image.tmdb.org/t/p",
			libretro:    "https://thumbnails.libretro.com",
			musicbrainz: "https://musicbrainz.org",
			coverart:    "https://coverartarchive.org",
		},
		http: &getter{client: &http.Client{Timeout: 60 * time.Second}, tries: 4,
			backoff: time.Second, maxWait: time.Minute},
	}
	f.http.ua = f.userAgent
	f.lim.tmdb = &limiter{every: 250 * time.Millisecond} // ~40 a cada 10 s
	f.lim.libretro = &limiter{every: 100 * time.Millisecond}
	f.lim.mb = &limiter{every: time.Second} // regra do MusicBrainz: 1/s
	f.lim.caa = &limiter{every: 100 * time.Millisecond}
	return f
}

func buildVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

func (f *Fetcher) userAgent() string {
	return "arkflix-server/" + f.Version + " ( https://github.com/stanleygomes/arkflix )"
}

// Status devolve o andamento.
func (f *Fetcher) Status() Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

// Run completa o que falta e volta quando acabar (ou ctx cancelar). Se já
// houver um rodando, só pede mais um ciclo no fim e volta na hora.
func (f *Fetcher) Run(ctx context.Context) error {
	if !f.cfg.Metadata {
		return nil
	}
	f.mu.Lock()
	if f.status.Running {
		f.again = true
		f.mu.Unlock()
		return nil
	}
	f.status = Status{Running: true}
	f.mu.Unlock()
	for {
		err := f.pass(ctx)
		f.mu.Lock()
		if err == nil && f.again {
			f.again = false
			f.mu.Unlock()
			continue
		}
		f.again, f.status.Running = false, false
		if err != nil {
			f.status.Error = err.Error()
		}
		f.mu.Unlock()
		return err
	}
}

// Refresh busca de novo um item agora, ignorando a janela de nova
// tentativa e o id do provedor já guardado (o admin chama).
func (f *Fetcher) Refresh(ctx context.Context, id string) error {
	if !f.cfg.Metadata {
		return ErrDisabled
	}
	it, err := f.st.Item(ctx, id)
	if err != nil {
		return err
	}
	if !f.supported(it.Kind) {
		return ErrUnsupported
	}
	return f.process(ctx, it, newCache(), true)
}

func (f *Fetcher) supported(kind string) bool {
	switch kind {
	case store.KindMovie, store.KindSeries, store.KindSeason, store.KindEpisode:
		return f.cfg.TMDBKey != ""
	case store.KindAlbum, store.KindRom:
		return true
	}
	return false
}

// pass é um ciclo: limpa imagens órfãs e busca cada candidato.
func (f *Fetcher) pass(ctx context.Context) error {
	f.cleanOrphans(ctx)
	todo, err := f.candidates(ctx)
	if err != nil {
		return err
	}
	f.setStatus(func(s *Status) { s.Pending = len(todo) })
	c := newCache()
	for _, it := range todo {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := f.process(ctx, it, c, false)
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		f.setStatus(func(s *Status) {
			s.Pending--
			if err == nil {
				s.Done++
			} else if !errors.Is(err, errNotFound) {
				s.Error = err.Error()
			}
		})
	}
	return nil
}

func (f *Fetcher) setStatus(fn func(*Status)) {
	f.mu.Lock()
	fn(&f.status)
	f.mu.Unlock()
}

// candidates são os itens dos tipos com provedor, faltando algo e fora da
// janela de nova tentativa. Séries antes de temporadas e episódios (eles
// precisam do id da série no TMDB).
func (f *Fetcher) candidates(ctx context.Context) ([]*store.Item, error) {
	items, err := f.st.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i
WHERE i.kind IN ('movie','series','season','episode','album','rom')
ORDER BY CASE i.kind WHEN 'movie' THEN 0 WHEN 'series' THEN 1 WHEN 'season' THEN 2 WHEN 'episode' THEN 3
  WHEN 'album' THEN 4 ELSE 5 END, i.path`)
	if err != nil {
		return nil, err
	}
	now := f.now()
	out := items[:0]
	for _, it := range items {
		if f.supported(it.Kind) && due(it, now) {
			out = append(out, it)
		}
	}
	return out, nil
}

// due: falta dado e não tentou há pouco.
func due(it *store.Item, now time.Time) bool {
	m := readMeta(it)
	if t := parse(m.Missed); !t.IsZero() && now.Sub(t) < retryMiss {
		return false
	}
	if t := parse(m.Fetched); !t.IsZero() && now.Sub(t) < retryStale {
		return false
	}
	return needs(it)
}

func parse(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// needs diz se falta algo que o provedor do tipo preenche.
func needs(it *store.Item) bool {
	switch it.Kind {
	case store.KindMovie, store.KindSeries:
		return it.Overview == "" || it.Image == "" || it.Backdrop == "" || it.Logo == "" ||
			len(it.Genres) == 0 || it.Premiere == ""
	case store.KindSeason:
		return it.Overview == "" || it.Image == ""
	case store.KindEpisode:
		return it.Overview == "" || it.Image == "" || genericName(it)
	case store.KindAlbum, store.KindRom:
		return it.Image == ""
	}
	return false
}

// process busca e grava um item; falha vira "missed" (nova tentativa só
// no outro dia).
func (f *Fetcher) process(ctx context.Context, it *store.Item, c *cache, force bool) error {
	f.work.Lock()
	defer f.work.Unlock()
	p, err := f.fetch(ctx, it, c, force)
	if err == nil {
		err = f.apply(ctx, it.ID, p)
	}
	if err != nil && ctx.Err() == nil {
		slog.Debug("metadados: falhou", "item", it.Path, "err", err)
		if merr := f.markMissed(ctx, it.ID, sourceOf(it.Kind), err); merr != nil {
			return merr
		}
	}
	return err
}

func (f *Fetcher) fetch(ctx context.Context, it *store.Item, c *cache, force bool) (*patch, error) {
	switch it.Kind {
	case store.KindMovie:
		return f.tmdbMovie(ctx, it, force)
	case store.KindSeries:
		return f.tmdbSeries(ctx, it, force)
	case store.KindSeason:
		return f.tmdbSeason(ctx, it, c)
	case store.KindEpisode:
		return f.tmdbEpisode(ctx, it, c)
	case store.KindAlbum:
		return f.musicAlbum(ctx, it)
	case store.KindRom:
		return f.libretroRom(ctx, it)
	}
	return nil, ErrUnsupported
}

func sourceOf(kind string) string {
	switch kind {
	case store.KindAlbum:
		return "musicbrainz"
	case store.KindRom:
		return "libretro"
	}
	return "tmdb"
}

var reHexID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// imageDir é onde ficam as imagens baixadas do item.
func (f *Fetcher) imageDir(id string) string {
	return filepath.Join(f.cfg.Data, "metadata", id)
}

// cleanOrphans apaga as imagens de itens que saíram do acervo.
func (f *Fetcher) cleanOrphans(ctx context.Context) {
	root := filepath.Join(f.cfg.Data, "metadata")
	ents, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range ents {
		if !e.IsDir() || !reHexID.MatchString(e.Name()) {
			continue
		}
		if _, err := f.st.Item(ctx, e.Name()); errors.Is(err, store.ErrNotFound) {
			os.RemoveAll(filepath.Join(root, e.Name()))
		}
	}
}

// cache guarda, num ciclo, as temporadas já pedidas ao TMDB.
type cache struct {
	seasons map[string]seasonResult
}

type seasonResult struct {
	s   *tmdbSeason
	err error
}

func newCache() *cache { return &cache{seasons: map[string]seasonResult{}} }

func seasonKey(tv, n int, lang string) string { return fmt.Sprintf("%d/%d/%s", tv, n, lang) }
