package subsonic

import (
	"archive/zip"
	"bufio"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dhowden/tag"

	"github.com/stanleygomes/arkflix/server/internal/media"
)

// formatos que o media.TranscodeAudio sabe gerar
var transcodable = map[string]bool{"mp3": true, "opus": true, "ogg": true, "aac": true, "flac": true}

// plan decide entre mandar o arquivo original e transcodificar. format
// "raw", ou o mesmo formato sem teto de bitrate, é o original; teto acima
// do bitrate do arquivo também. Sem formato e com teto, vira mp3 (o
// Navidrome usa opus, mas iOS não toca ogg).
func plan(format string, maxBitRate int, t *track, haveFFmpeg bool) (target string, kbps int) {
	suf := suffix(t.Path)
	format = strings.ToLower(format)
	fileKbps := t.Bitrate / 1000
	underCap := maxBitRate == 0 || (fileKbps > 0 && fileKbps <= maxBitRate)
	switch {
	case !haveFFmpeg || format == "raw":
		return "", 0
	case format == "" || format == suf:
		if underCap {
			return "", 0
		}
		target = "mp3"
		if transcodable[suf] && suf != "flac" {
			target = suf
		}
	case transcodable[format]:
		target = format
	default:
		return "", 0 // formato que não sabemos gerar: original
	}
	return target, maxBitRate
}

func (a *API) stream(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	t, err := a.oneTrack(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, err
	}
	if rg := c.r.Header.Get("Range"); c.r.Method == http.MethodGet && (rg == "" || strings.HasPrefix(rg, "bytes=0-")) {
		a.setPlaying(c, t.ID, t.Duration, 0)
	}
	c.w.Header().Set("X-Content-Type-Options", "nosniff")
	c.w.Header().Set("X-Content-Duration", strconv.FormatFloat(t.Duration, 'f', 2, 64))
	target, kbps := plan(c.p.str("format"), c.p.int("maxBitRate", 0), t, a.d.Config.FFmpeg != "")
	if target == "" {
		c.w.Header().Set("Content-Type", contentType(t.Path))
		a.d.Media.ServeFile(c.w, c.r, t.Path)
		return nil, nil
	}
	return a.transcode(c, t, target, kbps)
}

func (a *API) transcode(c *call, t *track, format string, kbps int) (*resp, error) {
	offset := max(c.p.int("timeOffset", 0), 0)
	h := c.w.Header()
	h.Set("Content-Type", media.ContentType(format))
	h.Set("Accept-Ranges", "none")
	out := &lazyWriter{w: c.w}
	if c.p.bool("estimateContentLength", false) && format != "flac" {
		rate := kbps
		if rate == 0 {
			rate = 192
		}
		n := int64((t.Duration - float64(offset)) * float64(rate) * 1000 / 8)
		if n > 0 {
			h.Set("Content-Length", strconv.FormatInt(n, 10))
			out.limit = n
		}
	}
	if c.r.Method == http.MethodHead {
		c.w.WriteHeader(http.StatusOK)
		return nil, nil
	}
	err := a.d.Media.TranscodeAudio(c.ctx, out, t.Path, media.AudioOpts{Format: format, Bitrate: kbps, Offset: float64(offset)})
	if err != nil && out.n == 0 && c.ctx.Err() == nil {
		h.Del("Content-Length")
		h.Del("Content-Type")
		return nil, err
	}
	return nil, nil
}

// lazyWriter só manda o cabeçalho no primeiro byte (dá pra responder erro
// se o ffmpeg nem começar) e corta no Content-Length estimado.
type lazyWriter struct {
	w     http.ResponseWriter
	n     int64
	limit int64
}

func (l *lazyWriter) Write(p []byte) (int, error) {
	full := len(p)
	if l.limit > 0 {
		if l.n >= l.limit {
			return full, nil
		}
		p = p[:min(int64(len(p)), l.limit-l.n)]
	}
	k, err := l.w.Write(p)
	l.n += int64(k)
	if err != nil {
		return k, err
	}
	if f, ok := l.w.(http.Flusher); ok && l.n < 1<<16 {
		f.Flush() // começa a tocar logo
	}
	return full, nil
}

func attachment(w http.ResponseWriter, name string) {
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
}

// download manda o original; álbum, artista e playlist vão num zip.
func (a *API) download(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	var name string
	var ts []*track
	switch a.kindOf(c.ctx, id) {
	case "track":
		t, err := a.oneTrack(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		attachment(c.w, filepath.Base(t.Path))
		c.w.Header().Set("Content-Type", contentType(t.Path))
		a.d.Media.ServeFile(c.w, c.r, t.Path)
		return nil, nil
	case "album":
		al, err := a.oneAlbum(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		name = al.Name
		if ts, err = a.albumTracks(c, al.ID); err != nil {
			return nil, err
		}
	case "artist":
		ar, err := a.oneArtist(c.ctx, c.user.ID, id)
		if err != nil {
			return nil, err
		}
		name = ar.Name
		if ts, err = a.tracks(c.ctx, c.user.ID, `AND (i.artist_id = ? OR i.album_id IN (SELECT id FROM items WHERE kind = 'album' AND artist_id = ?))`+sortLib, ar.ID, ar.ID); err != nil {
			return nil, err
		}
	case "playlist":
		p, err := a.plLoad(c, id)
		if err != nil {
			return nil, err
		}
		ids, err := a.plItemIDs(c.ctx, p.ID)
		if err != nil {
			return nil, err
		}
		name = p.Name
		if ts, err = a.tracksByIDs(c.ctx, c.user.ID, ids); err != nil {
			return nil, err
		}
	default:
		return nil, fail(errNotFound, "data not found")
	}
	attachment(c.w, name+".zip")
	c.w.Header().Set("Content-Type", "application/zip")
	zw := zip.NewWriter(c.w)
	seen := map[string]bool{}
	for _, t := range ts {
		inner := noSlash(t.Album) + "/" + filepath.Base(t.Path)
		if seen[inner] {
			continue
		}
		seen[inner] = true
		if err := addFile(zw, inner, t.Path); err != nil {
			break
		}
	}
	zw.Close()
	return nil, nil
}

func addFile(zw *zip.Writer, name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return nil // arquivo sumiu: pula
	}
	defer f.Close()
	w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// coverSrc acha a imagem de um id (faixa, álbum, artista, playlist).
func (a *API) coverSrc(c *call, id string) string {
	switch a.kindOf(c.ctx, id) {
	case "track":
		if t, err := a.oneTrack(c.ctx, c.user.ID, id); err == nil {
			if t.Image != "" {
				return t.Image
			}
			if al, err := a.oneAlbum(c.ctx, c.user.ID, t.AlbumID); err == nil {
				return al.cover
			}
		}
	case "album":
		if al, err := a.oneAlbum(c.ctx, c.user.ID, id); err == nil {
			return al.cover
		}
	case "artist":
		if ar, err := a.oneArtist(c.ctx, c.user.ID, id); err == nil && ar.coverAlbum != "" {
			if al, err := a.oneAlbum(c.ctx, c.user.ID, ar.coverAlbum); err == nil {
				return al.cover
			}
		}
	case "playlist":
		ids, _ := a.plItemIDs(c.ctx, cleanID(id))
		ts, _ := a.tracksByIDs(c.ctx, c.user.ID, ids)
		for _, t := range ts {
			if cv := t.cover(); cv != "" {
				return a.coverSrc(c, cv)
			}
		}
	}
	return ""
}

func (a *API) coverArt(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	src := a.coverSrc(c, id)
	if src == "" {
		return nil, fail(errNotFound, "Artwork not found")
	}
	size := max(c.p.int("size", 0), 0)
	a.d.Media.Image(c.w, c.r, src, media.ImageOpts{MaxWidth: size, MaxHeight: size})
	return nil, nil
}

// avatar: não temos foto de usuário.
func (a *API) avatar(c *call) (*resp, error) {
	http.NotFound(c.w, c.r)
	return nil, nil
}

// Letras: .lrc ou .txt ao lado do arquivo, senão a da tag.

var lrcTime = regexp.MustCompile(`\[(\d+):(\d+(?:[.:]\d+)?)\]`)
var lrcMeta = regexp.MustCompile(`^\[(ar|ti|al|by|offset|length|re|ve|au|#):(.*)\]$`)

// parseLyrics lê LRC (sincronizado) ou texto simples.
func parseLyrics(text string) structuredLyric {
	sl := structuredLyric{Lang: "xxx", Line: []line{}}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		raw := strings.TrimRight(sc.Text(), "\r")
		if m := lrcMeta.FindStringSubmatch(strings.TrimSpace(raw)); m != nil {
			val := strings.TrimSpace(m[2])
			switch m[1] {
			case "ar":
				sl.DisplayArtist = val
			case "ti":
				sl.DisplayTitle = val
			case "offset":
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					sl.Offset = &n
				}
			}
			continue
		}
		stamps := lrcTime.FindAllStringSubmatchIndex(raw, -1)
		if len(stamps) == 0 || stamps[0][0] != 0 {
			if !sl.Synced {
				sl.Line = append(sl.Line, line{Value: raw})
			}
			continue
		}
		end := 0
		var starts []int64
		for _, s := range stamps {
			if s[0] != end {
				break
			}
			end = s[1]
			starts = append(starts, lrcMs(raw[s[2]:s[3]], raw[s[4]:s[5]]))
		}
		if !sl.Synced {
			sl.Synced, sl.Line = true, []line{} // texto antes das marcas não conta
		}
		for _, st := range starts {
			v := st
			sl.Line = append(sl.Line, line{Start: &v, Value: strings.TrimSpace(raw[end:])})
		}
	}
	if sl.Synced {
		sortLines(sl.Line)
	}
	return sl
}

func lrcMs(minStr, secStr string) int64 {
	m, _ := strconv.ParseInt(minStr, 10, 64)
	secStr = strings.Replace(secStr, ":", ".", 1)
	s, _ := strconv.ParseFloat(secStr, 64)
	return m*60000 + int64(s*1000+0.5)
}

func sortLines(ls []line) {
	for i := 1; i < len(ls); i++ {
		for j := i; j > 0 && *ls[j].Start < *ls[j-1].Start; j-- {
			ls[j], ls[j-1] = ls[j-1], ls[j]
		}
	}
}

// findLyrics devolve as letras da faixa (vazio se não tiver).
func findLyrics(t *track) []structuredLyric {
	base := strings.TrimSuffix(t.Path, filepath.Ext(t.Path))
	for _, ext := range []string{".lrc", ".txt"} {
		if b, err := os.ReadFile(base + ext); err == nil && strings.TrimSpace(string(b)) != "" {
			return []structuredLyric{withNames(parseLyrics(string(b)), t)}
		}
	}
	f, err := os.Open(t.Path)
	if err != nil {
		return nil
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil || strings.TrimSpace(m.Lyrics()) == "" {
		return nil
	}
	return []structuredLyric{withNames(parseLyrics(m.Lyrics()), t)}
}

func withNames(sl structuredLyric, t *track) structuredLyric {
	if sl.DisplayArtist == "" {
		sl.DisplayArtist = t.Artist
	}
	if sl.DisplayTitle == "" {
		sl.DisplayTitle = t.Name
	}
	return sl
}

func (a *API) getLyrics(c *call) (*resp, error) {
	art, title := strings.TrimSpace(c.p.str("artist")), strings.TrimSpace(c.p.str("title"))
	res := a.newResp()
	res.Lyrics = &lyrics{}
	if title == "" {
		return res, nil
	}
	ts, err := a.tracks(c.ctx, c.user.ID, `AND lower(i.name) = lower(?) AND (? = '' OR lower(i.artist) = lower(?))`+sortLib, title, art, art)
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		ls := findLyrics(t)
		if len(ls) == 0 {
			continue
		}
		var b strings.Builder
		for _, l := range ls[0].Line {
			b.WriteString(l.Value + "\n")
		}
		res.Lyrics = &lyrics{Artist: art, Title: title, Value: b.String()}
		break
	}
	return res, nil
}

func (a *API) lyricsBySongID(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	t, err := a.oneTrack(c.ctx, c.user.ID, id)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.LyricsList = &lyricsList{StructuredLyrics: findLyrics(t)}
	return res, nil
}
