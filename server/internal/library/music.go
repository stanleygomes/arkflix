package library

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dhowden/tag"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

var audioExts = set("mp3", "flac", "ogg", "oga", "opus", "m4a", "m4b", "aac", "wav", "wma",
	"aiff", "aif", "ape", "wv", "mka", "dsf", "alac")

// variousArtists é o artista do álbum de coletânea sem tag de artista do álbum.
const variousArtists = "Vários Artistas"

// track é uma faixa lida (tag + ffprobe), antes de agrupar em álbum.
type track struct {
	f           mediaFile
	title       string
	artist      string
	albumArtist string // da tag; "" = usa o artista
	album       string
	num, disc   int
	year        int
	genres      []string
	picture     bool
	duration    float64
	bitrate     int
	streams     []store.Stream
}

// music: as tags mandam; sem tag, "Artista - Título" no nome do arquivo e
// a pasta vira o álbum.
func (s *scanner) music() error {
	files, err := s.walkFiles(s.root, audioExts, nil)
	if err != nil {
		return err
	}
	probes := s.probeNew(files)
	tracks := make([]*track, 0, len(files))
	for _, f := range files {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		tracks = append(tracks, s.readTrack(f, probes[f.path]))
	}
	fixCompilations(tracks)
	return s.saveMusic(tracks)
}

// readTrack lê a faixa; arquivo igual ao do último scan reaproveita tudo.
func (s *scanner) readTrack(f mediaFile, p *probed) *track {
	if old := s.unchanged(f.path, f.info); old != nil && (p == nil || old.Duration > 0) {
		return trackFromItem(f, old)
	}
	t := &track{f: f}
	s.readTags(t)
	if t.title == "" || t.artist == "" || t.album == "" {
		s.nameFallback(t)
	}
	t.streams = []store.Stream{{Index: 0, Type: "audio", Codec: ext(f.path), Default: true}}
	if p != nil {
		t.duration, t.bitrate, t.streams = p.Duration, p.Bitrate, p.Streams
		if b := audioBitrate(p.Streams); b > 0 {
			t.bitrate = b
		}
		for i := range t.streams {
			t.streams[i].Bitrate = 0 // na faixa o bitrate fica no item
		}
	}
	return t
}

func audioBitrate(ss []store.Stream) int {
	for _, s := range ss {
		if s.Type == "audio" {
			return s.Bitrate
		}
	}
	return 0
}

func trackFromItem(f mediaFile, it *store.Item) *track {
	t := &track{f: f, title: it.Name, artist: it.Artist, album: it.Album, num: it.Index, disc: it.ParentIndex,
		year: it.Year, genres: it.Genres, picture: it.Image != "", duration: it.Duration, bitrate: it.Bitrate,
		streams: it.Streams}
	if it.AlbumArtist != variousArtists {
		t.albumArtist = it.AlbumArtist // coletânea é recalculada
	}
	return t
}

func (s *scanner) readTags(t *track) {
	fh, err := os.Open(t.f.path)
	if err != nil {
		s.warn(t.f.path, err)
		return
	}
	defer fh.Close()
	m, err := tag.ReadFrom(fh)
	if err != nil {
		return // sem tag: o nome do arquivo resolve
	}
	raw := m.Raw()
	t.title, t.artist, t.album = strings.TrimSpace(m.Title()), strings.TrimSpace(m.Artist()), strings.TrimSpace(m.Album())
	t.albumArtist = firstNonEmpty(m.AlbumArtist(), rawStr(raw, "album_artist", "albumartist", "album artist", "TPE2", "aART"))
	t.num, _ = m.Track()
	t.disc, _ = m.Disc()
	if t.disc == 0 {
		t.disc = leadInt(rawStr(raw, "disc", "discnumber"))
	}
	t.year = m.Year()
	if t.year == 0 {
		t.year = leadInt(rawStr(raw, "date", "year", "originaldate"))
	}
	t.genres = splitGenres(m.Genre())
	t.picture = m.Picture() != nil
}

var reTrackNum = regexp.MustCompile(`^(\d{1,3})(?:\s*[-._]\s*|\s+)`)

// nameFallback completa o que a tag não deu: "[NN - ]Artista - Título" e
// a pasta como álbum.
func (s *scanner) nameFallback(t *track) {
	base := stem(t.f.info.Name())
	if m := reTrackNum.FindStringSubmatch(base); m != nil {
		if t.num == 0 {
			t.num, _ = strconv.Atoi(m[1])
		}
		base = base[len(m[0]):]
	}
	artist, title := "", base
	if a, b, ok := strings.Cut(base, " - "); ok {
		artist, title = strings.TrimSpace(a), strings.TrimSpace(b)
	}
	dir := filepath.Dir(t.f.path)
	if artist == "" && filepath.Dir(dir) != s.root && dir != s.root {
		artist = filepath.Base(filepath.Dir(dir)) // Artista/Álbum/faixa
	}
	if t.title == "" {
		t.title = title
	}
	if t.artist == "" {
		t.artist = firstNonEmpty(t.albumArtist, artist, "Artista desconhecido")
	}
	if t.album == "" {
		t.album = "Sem álbum"
		if dir != s.root {
			t.album = filepath.Base(dir)
		}
	}
}

// fixCompilations: mesma pasta e álbum, artistas diferentes e nenhuma tag
// de artista do álbum = coletânea.
func fixCompilations(ts []*track) {
	type key struct{ dir, album string }
	artists := map[key]map[string]bool{}
	tagged := map[key]bool{}
	for _, t := range ts {
		k := key{filepath.Dir(t.f.path), strings.ToLower(t.album)}
		if artists[k] == nil {
			artists[k] = map[string]bool{}
		}
		artists[k][strings.ToLower(t.artist)] = true
		tagged[k] = tagged[k] || t.albumArtist != ""
	}
	for _, t := range ts {
		k := key{filepath.Dir(t.f.path), strings.ToLower(t.album)}
		if !tagged[k] && len(artists[k]) > 1 {
			t.albumArtist = variousArtists
		}
	}
}

func rawStr(raw map[string]any, keys ...string) string {
	for _, k := range keys {
		for rk, v := range raw {
			if !strings.EqualFold(rk, k) {
				continue
			}
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x = strings.TrimSpace(x); x != "" {
			return x
		}
	}
	return ""
}

var reLeadInt = regexp.MustCompile(`^\s*(\d+)`)

func leadInt(s string) int {
	m := reLeadInt.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// splitGenres separa "Rock; Pop" / "Rock/Pop".
func splitGenres(g string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(g, func(r rune) bool { return r == ';' || r == '/' || r == '|' || r == '\x00' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
