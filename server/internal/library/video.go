package library

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

var videoExts = set("mkv", "mp4", "m4v", "avi", "webm", "mov", "wmv", "ts", "m2ts", "mts",
	"mpg", "mpeg", "flv", "ogv", "3gp", "divx", "vob", "iso")

var subExts = set("srt", "vtt", "ass", "ssa", "sub")

func set(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// extrasDir são pastas de extras/trailers, que não são filmes nem episódios.
func extrasDir(name string) bool {
	switch strings.ToLower(name) {
	case "extras", "featurettes", "trailers", "behind the scenes", "deleted scenes",
		"interviews", "scenes", "shorts", "samples", "sample", "other", "bonus":
		return true
	}
	return false
}

var reSample = regexp.MustCompile(`(?i)(?:^|[ ._-])(sample|trailer)$`)

// videoFiles lista os vídeos de dir, sem amostras e trailers.
func (s *scanner) videoFiles(dir string) ([]mediaFile, error) {
	all, err := s.walkFiles(dir, videoExts, extrasDir)
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, f := range all {
		if !reSample.MatchString(stem(f.info.Name())) {
			out = append(out, f)
		}
	}
	return out, nil
}

// probeNew roda o ffprobe só nos arquivos novos, mudados ou que ficaram
// sem ffprobe no scan anterior (duração 0).
func (s *scanner) probeNew(files []mediaFile) map[string]*probed {
	var need []string
	for _, f := range files {
		if old := s.unchanged(f.path, f.info); old == nil || old.Duration == 0 {
			need = append(need, f.path)
		}
	}
	return probeAll(s.ctx, s.cfg.FFprobe, need, s.warn)
}

// fileItem preenche o que vem do arquivo: tamanho, mtime, contêiner e o
// ffprobe (do scan anterior se o arquivo não mudou).
func (s *scanner) fileItem(it *store.Item, f mediaFile, probes map[string]*probed) {
	it.Path, it.RelPath = f.path, s.rel(f.path)
	it.Size, it.MTime = f.info.Size(), f.info.ModTime().Unix()
	it.Container = ext(f.path)
	if p := probes[f.path]; p != nil {
		it.Duration, it.Bitrate, it.Width, it.Height = p.Duration, p.Bitrate, p.Width, p.Height
		it.Streams = p.Streams
		return
	}
	if old := s.unchanged(f.path, f.info); old != nil {
		it.Duration, it.Bitrate, it.Width, it.Height = old.Duration, old.Bitrate, old.Width, old.Height
		for _, st := range old.Streams {
			if st.External == "" {
				it.Streams = append(it.Streams, st)
			}
		}
	}
}

// addSubs acrescenta as legendas externas <base>[.idioma][.forced].srt
// depois das faixas do arquivo.
func (s *scanner) addSubs(it *store.Item) {
	dir, base := filepath.Dir(it.Path), stem(filepath.Base(it.Path))
	next := 0
	for _, st := range it.Streams {
		if st.Index >= next {
			next = st.Index + 1
		}
	}
	var names []string
	for low, real := range s.files(dir) {
		if subExts[ext(low)] && strings.HasPrefix(low, strings.ToLower(base)+".") {
			names = append(names, real)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		st := subStream(n[len(base):], filepath.Join(dir, n))
		st.Index = next
		next++
		it.Streams = append(it.Streams, st)
	}
}

// subStream interpreta o resto do nome (".pt.forced.srt").
func subStream(rest, path string) store.Stream {
	parts := strings.Split(strings.Trim(stem(rest), "."), ".")
	st := store.Stream{Type: "subtitle", Codec: ext(path), External: path}
	for _, p := range parts {
		switch strings.ToLower(p) {
		case "":
		case "forced", "foreign":
			st.Forced = true
		case "default":
			st.Default = true
		case "sdh", "cc", "hi":
			st.Title = strings.ToUpper(p)
		default:
			if l := subLang(p); l != "" && st.Language == "" {
				st.Language = l
			}
		}
	}
	return st
}

// nfo é o pouco que lemos do .nfo do Kodi/Jellyfin.
type nfo struct {
	Title     string   `xml:"title"`
	Year      string   `xml:"year"`
	Plot      string   `xml:"plot"`
	Genres    []string `xml:"genre"`
	Premiered string   `xml:"premiered"`
	Aired     string   `xml:"aired"`
}

var reDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// readNFO lê o primeiro .nfo que existir (nil se nenhum ou inválido).
func readNFO(paths ...string) *nfo {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		var n nfo
		err = xml.NewDecoder(f).Decode(&n)
		f.Close()
		if err == nil {
			return &n
		}
	}
	return nil
}

// apply copia do .nfo o que veio preenchido.
func (n *nfo) apply(it *store.Item) {
	if n == nil {
		return
	}
	if t := strings.TrimSpace(n.Title); t != "" {
		it.Name = t
	}
	if y, err := strconv.Atoi(strings.TrimSpace(n.Year)); err == nil && y > 0 {
		it.Year = y
	}
	if p := strings.TrimSpace(n.Plot); p != "" {
		it.Overview = p
	}
	var genres []string
	for _, g := range n.Genres {
		if g = strings.TrimSpace(g); g != "" {
			genres = append(genres, g)
		}
	}
	if len(genres) > 0 {
		it.Genres = genres
	}
	for _, d := range []string{n.Premiered, n.Aired} {
		if d = strings.TrimSpace(d); reDate.MatchString(d) {
			it.Premiere = d[:10]
			if it.Year == 0 {
				it.Year, _ = strconv.Atoi(d[:4])
			}
			break
		}
	}
}
