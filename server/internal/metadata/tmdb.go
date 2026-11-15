package metadata

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Respostas do TMDB (API v3), só o que usamos.
type tmdbResult struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	Name          string `json:"name"`
	OriginalName  string `json:"original_name"`
	ReleaseDate   string `json:"release_date"`
	FirstAirDate  string `json:"first_air_date"`
}

type tmdbImage struct {
	FilePath string  `json:"file_path"`
	Lang     *string `json:"iso_639_1"`
	Vote     float64 `json:"vote_average"`
}

type tmdbDetails struct {
	ID           int     `json:"id"`
	Overview     string  `json:"overview"`
	Tagline      string  `json:"tagline"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	Vote         float64 `json:"vote_average"`
	IMDBID       string  `json:"imdb_id"`
	Genres       []struct {
		Name string `json:"name"`
	} `json:"genres"`
	ExternalIDs struct {
		IMDBID string `json:"imdb_id"`
	} `json:"external_ids"`
	Images struct {
		Logos []tmdbImage `json:"logos"`
	} `json:"images"`
}

type tmdbSeason struct {
	Overview   string        `json:"overview"`
	PosterPath string        `json:"poster_path"`
	AirDate    string        `json:"air_date"`
	Episodes   []tmdbEpisode `json:"episodes"`
}

type tmdbEpisode struct {
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	StillPath     string  `json:"still_path"`
	AirDate       string  `json:"air_date"`
	Vote          float64 `json:"vote_average"`
}

const english = "en-US"

// tmdbGet chama a API. Chave v3 vai na query; token v4 (JWT) no Bearer.
func (f *Fetcher) tmdbGet(ctx context.Context, path string, q url.Values, v any) error {
	if q == nil {
		q = url.Values{}
	}
	var hdr http.Header
	if key := f.cfg.TMDBKey; strings.HasPrefix(key, "eyJ") || len(key) > 64 {
		hdr = http.Header{"Authorization": {"Bearer " + key}, "Accept": {"application/json"}}
	} else {
		q.Set("api_key", key)
	}
	return f.http.getJSON(ctx, f.lim.tmdb, f.urls.tmdb+path+"?"+q.Encode(), hdr, v)
}

func (f *Fetcher) lang() string {
	if f.cfg.Language == "" {
		return "pt-BR"
	}
	return f.cfg.Language
}

// imageLangs é o include_image_language: o idioma, inglês e sem texto.
func (f *Fetcher) imageLangs() string {
	l := strings.ToLower(strings.SplitN(f.lang(), "-", 2)[0])
	if l == "en" {
		return "en,null"
	}
	return l + ",en,null"
}

func (f *Fetcher) tmdbImage(size, path string) []string {
	if path == "" {
		return nil
	}
	return []string{f.urls.tmdbImg + "/" + size + path}
}

// tmdbMovie: busca por título+ano, detalhes no idioma (com imagens e ids).
func (f *Fetcher) tmdbMovie(ctx context.Context, it *store.Item, force bool) (*patch, error) {
	return f.tmdbTitle(ctx, it, force, "movie")
}

func (f *Fetcher) tmdbSeries(ctx context.Context, it *store.Item, force bool) (*patch, error) {
	return f.tmdbTitle(ctx, it, force, "tv")
}

func (f *Fetcher) tmdbTitle(ctx context.Context, it *store.Item, force bool, kind string) (*patch, error) {
	id := 0
	if !force {
		id = intExtra(it, "tmdb_id")
	}
	if id == 0 {
		var err error
		if id, err = f.tmdbSearch(ctx, kind, it.Name, it.Year); err != nil {
			return nil, err
		}
	}
	var d tmdbDetails
	q := url.Values{"language": {f.lang()}, "append_to_response": {"images,external_ids"},
		"include_image_language": {f.imageLangs()}}
	if err := f.tmdbGet(ctx, fmt.Sprintf("/%s/%d", kind, id), q, &d); err != nil {
		return nil, err
	}
	if d.Overview == "" && !isEnglish(f.lang()) {
		var en tmdbDetails
		if err := f.tmdbGet(ctx, fmt.Sprintf("/%s/%d", kind, id), url.Values{"language": {english}}, &en); err == nil {
			d.Overview = en.Overview
			if d.Tagline == "" {
				d.Tagline = en.Tagline
			}
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	p := &patch{source: "tmdb", overview: strings.TrimSpace(d.Overview), images: map[string][]string{},
		extra: map[string]any{"tmdb_id": id}}
	p.premiere = firstNonEmpty(d.ReleaseDate, d.FirstAirDate)
	if len(p.premiere) >= 10 {
		p.premiere = p.premiere[:10]
		p.year, _ = strconv.Atoi(p.premiere[:4])
	}
	for _, g := range d.Genres {
		if g.Name != "" {
			p.genres = append(p.genres, g.Name)
		}
	}
	p.images["image"] = f.tmdbImage("w780", d.PosterPath)
	p.images["backdrop"] = f.tmdbImage("w1280", d.BackdropPath)
	p.images["logo"] = f.tmdbImage("w500", f.pickLogo(d.Images.Logos))
	if imdb := firstNonEmpty(d.IMDBID, d.ExternalIDs.IMDBID); imdb != "" {
		p.extra["imdb_id"] = imdb
	}
	if d.Vote > 0 {
		p.extra["rating"] = math.Round(d.Vote*10) / 10
	}
	if t := strings.TrimSpace(d.Tagline); t != "" {
		p.extra["tagline"] = t
	}
	return p, nil
}

// pickLogo: logo no idioma, senão inglês, senão sem idioma (a lista já
// vem pela nota).
func (f *Fetcher) pickLogo(logos []tmdbImage) string {
	want := strings.ToLower(strings.SplitN(f.lang(), "-", 2)[0])
	for _, l := range []string{want, "en", ""} {
		for _, img := range logos {
			lang := ""
			if img.Lang != nil {
				lang = *img.Lang
			}
			if lang == l && img.FilePath != "" {
				return img.FilePath
			}
		}
	}
	return ""
}

// tmdbSearch acha o id: no idioma com ano, sem ano, e em inglês.
func (f *Fetcher) tmdbSearch(ctx context.Context, kind, title string, year int) (int, error) {
	yearParam := map[string]string{"movie": "year", "tv": "first_air_date_year"}[kind]
	type try struct {
		lang string
		year bool
	}
	tries := []try{{f.lang(), true}, {f.lang(), false}, {english, false}}
	if year == 0 {
		tries = []try{{f.lang(), false}, {english, false}}
	}
	if isEnglish(f.lang()) {
		tries = tries[:len(tries)-1]
	}
	for _, t := range tries {
		q := url.Values{"query": {title}, "language": {t.lang}, "include_adult": {"false"}}
		if t.year {
			q.Set(yearParam, strconv.Itoa(year))
		}
		var res struct {
			Results []tmdbResult `json:"results"`
		}
		if err := f.tmdbGet(ctx, "/search/"+kind, q, &res); err != nil {
			return 0, err
		}
		if id := pickResult(res.Results, title, year); id != 0 {
			return id, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", errNotFound, title)
}

// pickResult escolhe o resultado: título igual (normalizado, contra o
// traduzido e o original) com ano a até 1 de distância; sem título igual,
// só vale o ano exato com um título contendo o outro.
func pickResult(rs []tmdbResult, title string, year int) int {
	want := norm(title)
	if want == "" {
		return 0
	}
	fallback := 0
	for _, r := range rs {
		y := 0
		if d := firstNonEmpty(r.ReleaseDate, r.FirstAirDate); len(d) >= 4 {
			y, _ = strconv.Atoi(d[:4])
		}
		if year > 0 && y > 0 && abs(y-year) > 1 {
			continue
		}
		names := []string{norm(r.Title), norm(r.OriginalTitle), norm(r.Name), norm(r.OriginalName)}
		for _, n := range names {
			if n == want {
				return r.ID
			}
		}
		if fallback == 0 && year > 0 && y == year {
			for _, n := range names {
				if n != "" && (strings.Contains(n, want) || strings.Contains(want, n)) {
					fallback = r.ID
					break
				}
			}
		}
	}
	return fallback
}

// tmdbSeason: a temporada inteira num pedido só (cache por ciclo).
func (f *Fetcher) season(ctx context.Context, c *cache, tv, n int, lang string) (*tmdbSeason, error) {
	key := seasonKey(tv, n, lang)
	if r, ok := c.seasons[key]; ok {
		return r.s, r.err
	}
	var s tmdbSeason
	err := f.tmdbGet(ctx, fmt.Sprintf("/tv/%d/season/%d", tv, n), url.Values{"language": {lang}}, &s)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	r := seasonResult{&s, err}
	if err != nil {
		r.s = nil
	}
	c.seasons[key] = r
	return r.s, r.err
}

// seriesTMDB é o id da série no TMDB (precisa ter sido buscada antes).
func (f *Fetcher) seriesTMDB(ctx context.Context, it *store.Item) (int, error) {
	ser, err := f.st.Item(ctx, it.SeriesID)
	if err != nil {
		return 0, err
	}
	if id := intExtra(ser, "tmdb_id"); id != 0 {
		return id, nil
	}
	return 0, fmt.Errorf("%w: série %q sem id do TMDB", errNotFound, ser.Name)
}

func (f *Fetcher) tmdbSeason(ctx context.Context, it *store.Item, c *cache) (*patch, error) {
	tv, err := f.seriesTMDB(ctx, it)
	if err != nil {
		return nil, err
	}
	s, err := f.season(ctx, c, tv, it.Index, f.lang())
	if err != nil {
		return nil, err
	}
	p := &patch{source: "tmdb", overview: s.Overview, images: map[string][]string{"image": f.tmdbImage("w780", s.PosterPath)}}
	if p.overview == "" && !isEnglish(f.lang()) {
		if en, err := f.season(ctx, c, tv, it.Index, english); err == nil {
			p.overview = en.Overview
		}
	}
	p.premiere, p.year = date(s.AirDate)
	return p, nil
}

func (f *Fetcher) tmdbEpisode(ctx context.Context, it *store.Item, c *cache) (*patch, error) {
	tv, err := f.seriesTMDB(ctx, it)
	if err != nil {
		return nil, err
	}
	s, err := f.season(ctx, c, tv, it.ParentIndex, f.lang())
	if err != nil {
		return nil, err
	}
	ep := findEpisode(s, it.Index)
	if ep == nil {
		return nil, fmt.Errorf("%w: episódio S%02dE%02d", errNotFound, it.ParentIndex, it.Index)
	}
	p := &patch{source: "tmdb", name: ep.Name, overview: ep.Overview,
		images: map[string][]string{"image": f.tmdbImage("w780", ep.StillPath)}, extra: map[string]any{}}
	if genericEpisodeTitle(p.name) {
		p.name = "" // "Episódio 3" do TMDB não acrescenta nada
	}
	if p.overview == "" && !isEnglish(f.lang()) {
		if en, err := f.season(ctx, c, tv, it.ParentIndex, english); err == nil {
			if e := findEpisode(en, it.Index); e != nil {
				p.overview = e.Overview
				if p.name == "" {
					p.name = e.Name // título no idioma vazio: o inglês serve
				}
			}
		}
	}
	if genericEpisodeTitle(p.name) {
		p.name = ""
	}
	p.premiere, p.year = date(ep.AirDate)
	if ep.Vote > 0 {
		p.extra["rating"] = math.Round(ep.Vote*10) / 10
	}
	return p, nil
}

func findEpisode(s *tmdbSeason, n int) *tmdbEpisode {
	for i := range s.Episodes {
		if s.Episodes[i].EpisodeNumber == n {
			return &s.Episodes[i]
		}
	}
	return nil
}

func genericEpisodeTitle(name string) bool {
	n := norm(name)
	for _, p := range []string{"episodio ", "episode "} {
		if rest, ok := strings.CutPrefix(n, p); ok {
			if _, err := strconv.Atoi(rest); err == nil {
				return true
			}
		}
	}
	return false
}

func date(s string) (string, int) {
	if len(s) < 10 {
		return "", 0
	}
	y, _ := strconv.Atoi(s[:4])
	return s[:10], y
}

func isEnglish(lang string) bool { return strings.HasPrefix(strings.ToLower(lang), "en") }

// norm deixa o título comparável: sem acento, minúsculo, só letras e
// números ("Star Wars: Episódio IV" → "star wars episodio iv").
func norm(s string) string {
	s = strings.ReplaceAll(store.Fold(s), "&", " and ")
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
		} else {
			space = true
		}
	}
	return b.String()
}

func intExtra(it *store.Item, key string) int {
	switch v := it.Extra[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
