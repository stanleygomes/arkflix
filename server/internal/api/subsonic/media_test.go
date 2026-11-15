package subsonic

import (
	"archive/zip"
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

func TestStreamRaw(t *testing.T) {
	e := setup(t)
	file := filepath.Join(e.k.Config.Music, "Banda Teste", "Disco Teste", "01 - Bip.mp3")
	want, _ := os.ReadFile(file)
	w := e.raw(http.MethodGet, "/rest/stream", login(q("id", e.ids.TrackBip)))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), want) || w.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("stream: %d %q %d bytes", w.Code, w.Header().Get("Content-Type"), w.Body.Len())
	}
	// Range
	w = e.raw(http.MethodGet, "/rest/stream.view", login(q("id", e.ids.TrackBip, "format", "raw", "maxBitRate", "32")), "Range", "bytes=10-19")
	if w.Code != http.StatusPartialContent || !bytes.Equal(w.Body.Bytes(), want[10:20]) {
		t.Fatalf("range: %d %d", w.Code, w.Body.Len())
	}
	// mesmo formato e teto acima do arquivo: original
	w = e.raw(http.MethodGet, "/rest/stream", login(q("id", e.ids.TrackBip, "format", "mp3", "maxBitRate", "320")))
	if !bytes.Equal(w.Body.Bytes(), want) {
		t.Fatal("mp3 320 devia ser o original")
	}
	// tocando agora vem do stream
	if r := e.ok("getNowPlaying", q()); len(r.NowPlaying.Entry) != 1 || r.NowPlaying.Entry[0].ID != e.ids.TrackBip {
		t.Fatalf("nowPlaying: %+v", r.NowPlaying)
	}
	e.failed("stream", q("id", "nada"), 70)
	e.failed("stream", q(), 10)

	// download: original como anexo
	w = e.raw(http.MethodGet, "/rest/download", login(q("id", e.ids.TrackBip)))
	if !bytes.Equal(w.Body.Bytes(), want) || !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("download: %v", w.Header())
	}
	// álbum vira zip
	w = e.raw(http.MethodGet, "/rest/download", login(q("id", e.ids.Album)))
	zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil || len(zr.File) != 2 || zr.File[0].Name != "Disco Teste/01 - Bip.mp3" {
		t.Fatalf("zip: %v %v", err, zr)
	}
}

func TestPlan(t *testing.T) {
	flac := &track{Item: itemWith("/m/a.flac", 900000)}
	mp3 := &track{Item: itemWith("/m/a.mp3", 320000)}
	cases := []struct {
		t      *track
		format string
		max    int
		want   string
		kbps   int
	}{
		{flac, "", 0, "", 0},
		{flac, "raw", 128, "", 0},
		{flac, "flac", 0, "", 0},
		{flac, "mp3", 0, "mp3", 0},
		{flac, "", 320, "mp3", 320},
		{flac, "opus", 96, "opus", 96},
		{flac, "wav", 0, "", 0},
		{mp3, "", 320, "", 0},
		{mp3, "", 128, "mp3", 128},
		{mp3, "mp3", 128, "mp3", 128},
	}
	for _, c := range cases {
		got, kbps := plan(c.format, c.max, c.t, true)
		if got != c.want || kbps != c.kbps {
			t.Errorf("%s %q %d: %q %d, esperava %q %d", c.t.Path, c.format, c.max, got, kbps, c.want, c.kbps)
		}
	}
	if got, _ := plan("mp3", 0, flac, false); got != "" {
		t.Error("sem ffmpeg tem que mandar o original")
	}
}

func TestStreamTranscode(t *testing.T) {
	e := setup(t)
	if e.k.Config.FFmpeg == "" {
		t.Skip("sem ffmpeg")
	}
	w := e.raw(http.MethodGet, "/rest/stream", login(q("id", e.ids.TrackBop, "format", "mp3", "maxBitRate", "128")))
	if w.Body.Len() == 0 {
		t.Skip("media.TranscodeAudio ainda não implementado")
	}
	b := w.Body.Bytes()
	isMP3 := bytes.HasPrefix(b, []byte("ID3")) || (len(b) > 1 && b[0] == 0xFF && b[1]&0xE0 == 0xE0)
	if w.Code != 200 || w.Header().Get("Content-Type") != "audio/mpeg" || !isMP3 {
		t.Fatalf("transcode: %d %q % x", w.Code, w.Header().Get("Content-Type"), b[:min(8, len(b))])
	}
	// tamanho estimado: corta no Content-Length
	w = e.raw(http.MethodGet, "/rest/stream", login(q("id", e.ids.TrackBop, "format", "mp3", "maxBitRate", "64", "estimateContentLength", "true")))
	if cl := w.Header().Get("Content-Length"); cl != "16000" || w.Body.Len() > 16000 {
		t.Fatalf("estimativa: %q %d", cl, w.Body.Len())
	}
	// HEAD só cabeçalho
	w = e.raw(http.MethodHead, "/rest/stream", login(q("id", e.ids.TrackBop, "format", "mp3")))
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("head: %d %d", w.Code, w.Body.Len())
	}
}

func TestCoverArt(t *testing.T) {
	e := setup(t)
	e.failed("getCoverArt", q("id", e.ids.TrackSolta), 70)
	e.failed("getCoverArt", q("id", e.ids.AlbumSolo), 70)
	e.failed("getCoverArt", q(), 10)
	for _, id := range []string{e.ids.Album, "al-" + e.ids.Album + "_0", e.ids.TrackBip, e.ids.TrackBop, e.ids.Artist} {
		w := e.raw(http.MethodGet, "/rest/getCoverArt", login(q("id", id, "size", "4")))
		if w.Code == http.StatusNotFound {
			t.Skip("media.Image ainda não implementado")
		}
		img, _, err := image.Decode(bytes.NewReader(w.Body.Bytes()))
		if err != nil || !strings.HasPrefix(w.Header().Get("Content-Type"), "image/") {
			t.Fatalf("%s: %v %q %s", id, err, w.Header().Get("Content-Type"), w.Body.String())
		}
		if b := img.Bounds(); b.Dx() > 4 || b.Dy() > 4 {
			t.Fatalf("%s: tamanho %v", id, b)
		}
	}
	if w := e.raw(http.MethodGet, "/rest/getAvatar", login(q("username", "stanley"))); w.Code != http.StatusNotFound {
		t.Fatalf("avatar: %d", w.Code)
	}
}

func TestLyrics(t *testing.T) {
	e := setup(t)
	r := e.ok("getLyricsBySongId", q("id", e.ids.TrackBip))
	if r.LyricsList == nil || len(r.LyricsList.StructuredLyrics) != 0 {
		t.Fatalf("sem letra: %+v", r.LyricsList)
	}
	lrc := "[ar:Banda]\n[ti:Bip]\n[00:01.50]primeira\n[00:00.25][00:03.00]refrão\n"
	os.WriteFile(filepath.Join(e.k.Config.Music, "Banda Teste", "Disco Teste", "01 - Bip.lrc"), []byte(lrc), 0o644)
	r = e.ok("getLyricsBySongId", q("id", e.ids.TrackBip))
	sl := r.LyricsList.StructuredLyrics
	if len(sl) != 1 || !sl[0].Synced || sl[0].DisplayArtist != "Banda" || len(sl[0].Line) != 3 {
		t.Fatalf("lrc: %+v", sl)
	}
	if l := sl[0].Line; *l[0].Start != 250 || l[0].Value != "refrão" || *l[1].Start != 1500 || *l[2].Start != 3000 {
		t.Fatalf("linhas: %+v", l)
	}
	os.WriteFile(filepath.Join(e.k.Config.Music, "Avulsas", "Artista X - Solta.txt"), []byte("um\ndois\n"), 0o644)
	r = e.ok("getLyricsBySongId", q("id", e.ids.TrackSolta))
	if sl := r.LyricsList.StructuredLyrics; len(sl) != 1 || sl[0].Synced || len(sl[0].Line) != 2 || sl[0].Line[1].Value != "dois" {
		t.Fatalf("txt: %+v", sl)
	}
	r = e.ok("getLyrics", q("artist", "Artista X", "title", "solta"))
	if r.Lyrics.Value != "um\ndois\n" {
		t.Fatalf("getLyrics: %+v", r.Lyrics)
	}
	if r := e.ok("getLyrics", q("title", "nada")); r.Lyrics == nil || r.Lyrics.Value != "" {
		t.Fatalf("vazio: %+v", r.Lyrics)
	}
	xml := e.xmlBody("getLyricsBySongId", q("id", e.ids.TrackBip))
	if !strings.Contains(xml, `<lyricsList><structuredLyrics displayArtist="Banda" displayTitle="Bip" lang="xxx" synced="true"><line start="250">refrão</line>`) {
		t.Fatalf("xml: %s", xml)
	}
}

func itemWith(path string, bitrate int) *store.Item { return &store.Item{Path: path, Bitrate: bitrate} }
