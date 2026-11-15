package importer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Mapper troca prefixos de caminho (ANTIGO=NOVO). Vale o prefixo mais
// longo que casar, e só em fronteira de pasta (/music não pega /musicas).
type Mapper []mapRule

type mapRule struct{ from, to string }

func (m *Mapper) String() string {
	var out []string
	for _, r := range *m {
		out = append(out, r.from+"="+r.to)
	}
	return strings.Join(out, ",")
}

// Set lê uma regra ANTIGO=NOVO (flag.Value).
func (m *Mapper) Set(s string) error {
	from, to, ok := strings.Cut(s, "=")
	if !ok || from == "" || to == "" {
		return errors.New("use ANTIGO=NOVO")
	}
	*m = append(*m, mapRule{path.Clean(from), path.Clean(to)})
	sort.SliceStable(*m, func(i, j int) bool { return len((*m)[i].from) > len((*m)[j].from) })
	return nil
}

// Apply troca o prefixo (caminho sem regra que case volta limpo e igual).
func (m Mapper) Apply(p string) string {
	if p == "" {
		return ""
	}
	p = path.Clean(p)
	for _, r := range m {
		if p == r.from {
			return r.to
		}
		if r.from == "/" {
			return path.Join(r.to, p)
		}
		if rest, ok := strings.CutPrefix(p, r.from+"/"); ok {
			return path.Join(r.to, rest)
		}
	}
	return p
}

// Report conta o que o import fez e guarda os primeiros caminhos sem par.
type Report struct {
	Source string
	Dry    bool
	keys   []string
	n      map[string]int
	misses map[string]*miss
	mkeys  []string
	notes  []string
}

type miss struct {
	n     int
	first []string
	seen  map[string]bool
}

func newReport(source string, dry bool) *Report {
	return &Report{Source: source, Dry: dry, n: map[string]int{}, misses: map[string]*miss{}}
}

// Add soma n no contador key (a ordem de criação é a ordem do relatório).
func (r *Report) Add(key string, n int) {
	if _, ok := r.n[key]; !ok {
		r.keys = append(r.keys, key)
	}
	r.n[key] += n
}

// Count lê um contador.
func (r *Report) Count(key string) int { return r.n[key] }

// Miss registra um item sem par no acervo (cat: "jellyfin", "faixa"...).
func (r *Report) Miss(cat, p string) {
	m := r.misses[cat]
	if m == nil {
		m = &miss{seen: map[string]bool{}}
		r.misses[cat] = m
		r.mkeys = append(r.mkeys, cat)
	}
	m.n++
	if !m.seen[p] && len(m.first) < 200 {
		m.seen[p] = true
		m.first = append(m.first, p)
	}
}

// Misses diz quantos itens de cat ficaram sem par.
func (r *Report) Misses(cat string) int {
	if m := r.misses[cat]; m != nil {
		return m.n
	}
	return 0
}

// Note guarda um aviso pro fim do relatório.
func (r *Report) Note(format string, args ...any) {
	r.notes = append(r.notes, fmt.Sprintf(format, args...))
}

// Notes devolve os avisos.
func (r *Report) Notes() []string { return r.notes }

// Print escreve o resumo.
func (r *Report) Print(w io.Writer, show int) {
	head := r.Source
	if r.Dry {
		head += " (dry-run: nada foi gravado)"
	}
	fmt.Fprintln(w, head)
	for _, k := range r.keys {
		fmt.Fprintf(w, "  %-34s %d\n", k+":", r.n[k])
	}
	for _, k := range r.mkeys {
		m := r.misses[k]
		fmt.Fprintf(w, "  sem par no acervo (%s): %d\n", k, m.n)
		for i, p := range m.first {
			if i >= show {
				fmt.Fprintf(w, "    ... (--show N mostra mais)\n")
				break
			}
			fmt.Fprintf(w, "    %s\n", p)
		}
	}
	for _, n := range r.notes {
		fmt.Fprintf(w, "  ! %s\n", n)
	}
}

// item é o pedaço do acervo que o import precisa pra casar.
type item struct {
	ID, Kind, Path    string
	AlbumID, ArtistID string
	Platform          string
	RID               int64
}

// index é o acervo por caminho (e as ROMs por console/nome do arquivo).
type index struct {
	size   int
	byPath map[string]*item
	roms   map[string]*item // "<slug>/<fs_name>"
}

func loadIndex(ctx context.Context, st *store.Store) (*index, error) {
	rows, err := st.DB.QueryContext(ctx, `SELECT rid, id, kind, path, COALESCE(album_id,''), COALESCE(artist_id,''), platform FROM items`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idx := &index{byPath: map[string]*item{}, roms: map[string]*item{}}
	for rows.Next() {
		it := &item{}
		if err := rows.Scan(&it.RID, &it.ID, &it.Kind, &it.Path, &it.AlbumID, &it.ArtistID, &it.Platform); err != nil {
			return nil, err
		}
		idx.size++
		idx.byPath[it.Path] = it
		if it.Kind == store.KindRom {
			idx.roms[it.Platform+"/"+path.Base(it.Path)] = it
		}
	}
	return idx, rows.Err()
}

// artistKey e albumKey são as chaves que o scanner usa como path.
func artistKey(name string) string { return "artist:" + strings.ToLower(name) }

func albumKey(artist, album string) string {
	return "album:" + strings.ToLower(artist) + "/" + strings.ToLower(album)
}

// udKey é um (usuário, item) do estado acumulado.
type udKey struct{ user, item string }

// udIn é o estado vindo da fonte, já juntado entre linhas repetidas.
type udIn struct {
	d    store.UserData
	kind string
}

// addUserData acumula o estado de um item (várias linhas da fonte pro
// mesmo item se juntam com a mesma regra de juntar com o que já existe).
func (im *Importer) addUserData(user, kind string, it *item, d store.UserData) {
	k := udKey{user, it.ID}
	cur, ok := im.ud[k]
	if ok {
		d = mergeUD(cur.d, d)
	}
	im.ud[k] = udIn{d: d, kind: kind}
}

// flushUserData junta com o que o arkflix já tem e grava.
func (im *Importer) flushUserData(ctx context.Context) error {
	keys := make([]udKey, 0, len(im.ud))
	for k := range im.ud {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].user != keys[j].user {
			return keys[i].user < keys[j].user
		}
		return keys[i].item < keys[j].item
	})
	for _, k := range keys {
		in := im.ud[k]
		old, err := im.D.Store.UserData(ctx, k.user, k.item)
		if err != nil {
			return err
		}
		m := mergeUD(old, in.d)
		if sameUD(m, old) {
			im.rep.Add("estado sem mudança", 1)
			continue
		}
		im.rep.Add("estado gravado ("+in.kind+")", 1)
		if im.Dry {
			continue
		}
		if err := im.D.Store.SetUserData(ctx, k.user, k.item, m); err != nil {
			return err
		}
	}
	return nil
}

// mergeUD junta o estado da fonte com o que já existe sem perder nada:
// assistido e favorito somam, contagem fica a maior, onde parou fica o do
// mais recente. Juntar de novo com o mesmo resultado não muda nada (por
// isso o import pode rodar de novo).
func mergeUD(old, in store.UserData) store.UserData {
	out := old
	out.Played = old.Played || in.Played
	out.PlayCount = max(old.PlayCount, in.PlayCount)
	switch {
	case in.LastPlayed.After(old.LastPlayed):
		out.LastPlayed = in.LastPlayed
		out.Position = in.Position
	case old.LastPlayed.IsZero() && old.Position == 0:
		out.Position = in.Position
	}
	if in.Favorite && !old.Favorite {
		out.Favorite, out.FavoriteAt = true, in.FavoriteAt
	} else if in.Favorite && !in.FavoriteAt.IsZero() && (old.FavoriteAt.IsZero() || in.FavoriteAt.Before(old.FavoriteAt)) {
		out.FavoriteAt = in.FavoriteAt
	}
	if out.Rating == 0 {
		out.Rating = in.Rating
	}
	return out
}

func sameUD(a, b store.UserData) bool {
	return a.Played == b.Played && a.PlayCount == b.PlayCount && a.Position == b.Position &&
		a.LastPlayed.Equal(b.LastPlayed) && a.Favorite == b.Favorite && a.FavoriteAt.Equal(b.FavoriteAt) && a.Rating == b.Rating
}

// toTime lê data de qualquer jeito que os bancos devolvem (time.Time do
// driver, texto do EF Core, do Navidrome ou do Python). Zero se não der.
func toTime(v any) time.Time {
	var s string
	switch x := v.(type) {
	case time.Time:
		if x.Year() <= 1 {
			return time.Time{}
		}
		return x.UTC().Truncate(time.Millisecond)
	case string:
		s = x
	case []byte:
		s = string(x)
	default:
		return time.Time{}
	}
	s = strings.TrimSpace(s)
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999", "2006-01-02T15:04:05.999999999", "2006-01-02"} {
		if t, err := time.Parse(f, s); err == nil {
			if t.Year() <= 1 {
				return time.Time{}
			}
			return t.UTC().Truncate(time.Millisecond)
		}
	}
	return time.Time{}
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// openSQLite abre o banco da fonte só pra leitura (mode=ro + query_only):
// o import nunca escreve no Jellyfin/Navidrome.
func openSQLite(file string) (*sql.DB, error) {
	if _, err := os.Stat(file); err != nil {
		return nil, err
	}
	if strings.ContainsAny(file, "?#") {
		return nil, errors.New("caminho do banco com ? ou #: copie pra outro nome")
	}
	db, err := sql.Open("sqlite", "file:"+file+"?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// hasTable diz se o SQLite da fonte tem a tabela (versões diferentes).
func hasTable(ctx context.Context, db *sql.DB, name string) bool {
	var n int
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&n)
	return n > 0
}

// hasColumn diz se a tabela tem a coluna.
func hasColumn(ctx context.Context, db *sql.DB, table, col string) bool {
	var n int
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&n)
	return n > 0
}
