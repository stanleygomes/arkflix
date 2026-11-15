package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// meta é o que fica em Extra["meta"] (formato combinado com o scanner,
// ver library.MetaKey).
type meta struct {
	Source  string   `json:"source,omitempty"`
	Fetched string   `json:"fetched,omitempty"` // última busca que achou
	Missed  string   `json:"missed,omitempty"`  // última busca que falhou
	Error   string   `json:"error,omitempty"`
	Fields  []string `json:"fields"` // colunas preenchidas pela internet
	Keys    []string `json:"keys"`   // chaves de Extra preenchidas pela internet
}

func readMeta(it *store.Item) meta {
	var m meta
	if raw, err := json.Marshal(it.Extra[library.MetaKey]); err == nil {
		json.Unmarshal(raw, &m)
	}
	if m.Fields == nil {
		m.Fields = []string{}
	}
	if m.Keys == nil {
		m.Keys = []string{}
	}
	return m
}

func (m meta) toMap() map[string]any {
	raw, _ := json.Marshal(m)
	var out map[string]any
	json.Unmarshal(raw, &out)
	return out
}

// patch é o que um provedor achou pra um item. images: campo (image,
// backdrop, logo) → URLs candidatas em ordem; a primeira que baixar vale.
type patch struct {
	source   string
	name     string
	overview string
	premiere string // AAAA-MM-DD
	year     int
	genres   []string
	images   map[string][]string
	extra    map[string]any
	// needImage: sem a capa, nem tenta o resto (libretro: nome errado
	// dá 404 em tudo)
	needImage bool
}

// fileNames: nome do arquivo baixado por campo.
var fileNames = map[string]string{"image": "poster", "backdrop": "backdrop", "logo": "logo"}

func genericName(it *store.Item) bool {
	return it.Kind == store.KindEpisode && it.Name == library.EpisodeName(it.Index)
}

// apply grava o patch no item relido agora (o scanner pode ter mexido):
// só preenche campo vazio ou que já era da internet. Nada preenchido =
// errNotFound.
func (f *Fetcher) apply(ctx context.Context, id string, p *patch) error {
	it, err := f.st.Item(ctx, id)
	if err != nil {
		return err
	}
	m := readMeta(it)
	filled := 0
	can := func(field string, empty bool) bool { return empty || slices.Contains(m.Fields, field) }
	mark := func(field string) {
		filled++
		if !slices.Contains(m.Fields, field) {
			m.Fields = append(m.Fields, field)
		}
	}
	str := func(field string, dst *string, v string) {
		if v != "" && can(field, *dst == "") {
			*dst = v
			mark(field)
		}
	}
	if p.name != "" && can("name", genericName(it)) {
		it.Name, it.SortName = p.name, store.SortName(p.name)
		mark("name")
	}
	str("overview", &it.Overview, p.overview)
	str("premiere", &it.Premiere, p.premiere)
	if p.year > 0 && can("year", it.Year == 0) {
		it.Year = p.year
		mark("year")
	}
	if len(p.genres) > 0 && can("genres", len(it.Genres) == 0) {
		it.Genres = p.genres
		mark("genres")
	}
	for _, field := range []string{"image", "backdrop", "logo"} {
		dst := map[string]*string{"image": &it.Image, "backdrop": &it.Backdrop, "logo": &it.Logo}[field]
		if len(p.images[field]) == 0 || !can(field, *dst == "") {
			continue
		}
		name := fileNames[field]
		if field == "image" && it.Kind == store.KindEpisode {
			name = "still"
		}
		path, err := f.firstImage(ctx, p.source, p.images[field], it.ID, name)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			*dst = path
			mark(field)
		} else if field == "image" && p.needImage {
			return err
		}
	}
	for k, v := range p.extra {
		if _, has := it.Extra[k]; has && !slices.Contains(m.Keys, k) {
			continue // chave do disco (ROM etc.)
		}
		it.Extra[k] = v
		if !slices.Contains(m.Keys, k) {
			m.Keys = append(m.Keys, k)
		}
	}
	if filled == 0 && len(p.extra) == 0 {
		return errNotFound
	}
	m.Source, m.Fetched, m.Missed, m.Error = p.source, f.stamp(), "", ""
	it.Extra[library.MetaKey] = m.toMap()
	return f.write(ctx, it)
}

func (f *Fetcher) stamp() string { return f.now().UTC().Format(time.RFC3339) }

// firstImage baixa a primeira URL que existir.
func (f *Fetcher) firstImage(ctx context.Context, source string, urls []string, id, name string) (string, error) {
	lim := map[string]*limiter{"tmdb": f.lim.tmdb, "libretro": f.lim.libretro, "musicbrainz": f.lim.caa}[source]
	err := errNotFound
	for _, u := range urls {
		var path string
		path, err = f.http.download(ctx, lim, u, f.imageDir(id), name)
		if err == nil {
			return path, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}
	return "", err
}

// markMissed anota a falha (a próxima tentativa automática é no outro dia).
func (f *Fetcher) markMissed(ctx context.Context, id, source string, cause error) error {
	it, err := f.st.Item(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	m := readMeta(it)
	if m.Source == "" {
		m.Source = source
	}
	m.Missed, m.Error = f.stamp(), cause.Error()
	it.Extra[library.MetaKey] = m.toMap()
	return f.write(ctx, it)
}

// write grava só as colunas que os metadados tocam.
func (f *Fetcher) write(ctx context.Context, it *store.Item) error {
	genres, _ := json.Marshal(it.Genres)
	extra, _ := json.Marshal(it.Extra)
	_, err := f.st.DB.ExecContext(ctx, `UPDATE items SET name=?, sort_name=?, overview=?, year=?, premiere=?, genres=?,
  image=?, backdrop=?, logo=?, extra=?, updated_at=? WHERE id=?`,
		it.Name, it.SortName, it.Overview, it.Year, it.Premiere, string(genres),
		it.Image, it.Backdrop, it.Logo, string(extra), store.Now(), it.ID)
	return err
}
