package media_test

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func get(h func(http.ResponseWriter, *http.Request), method string, hdr ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/x", nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		r.Header.Set(hdr[i], hdr[i+1])
	}
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func TestServeFile(t *testing.T) {
	k := testkit.New(t)
	webm := filepath.Join(k.Config.Movies, "Filme Sem Ano.webm")
	serve := func(p string) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) { k.Media.ServeFile(w, r, p) }
	}
	w := get(serve(webm), "GET")
	if w.Code != 200 || w.Header().Get("Content-Type") != "video/webm" || w.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	w = get(serve(webm), "GET", "Range", "bytes=0-9")
	if w.Code != 206 || w.Body.Len() != 10 {
		t.Fatalf("range: %d %d", w.Code, w.Body.Len())
	}
	w = get(serve(webm), "HEAD")
	if w.Code != 200 || w.Body.Len() != 0 || w.Header().Get("Content-Length") == "" {
		t.Fatalf("head: %d %v", w.Code, w.Header())
	}
	flac := filepath.Join(k.Config.Music, "Banda Teste", "Disco Teste", "02 - Bop.flac")
	if ct := get(serve(flac), "GET").Header().Get("Content-Type"); ct != "audio/flac" {
		t.Fatalf("flac: %s", ct)
	}
	rom := filepath.Join(k.Config.Games, "roms", "snes", "Super Teste (USA).sfc")
	if ct := get(serve(rom), "GET").Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("rom: %s", ct)
	}
	for _, p := range []string{k.Config.Movies, filepath.Join(k.Root, "nada.mkv")} {
		if w := get(serve(p), "GET"); w.Code != 404 {
			t.Fatalf("%s: %d", p, w.Code)
		}
	}
	for in, want := range map[string]string{"a.mkv": "video/x-matroska", "a.MP4": "video/mp4", "a.srt": "application/x-subrip",
		"a.vtt": "text/vtt; charset=utf-8", "a.zip": "application/zip", "a.opus": "audio/ogg", "a.m4a": "audio/mp4"} {
		if got := media.MimeType(in); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}

func image_(t *testing.T, k *testkit.Kit, src string, o media.ImageOpts) (image.Image, string, *httptest.ResponseRecorder) {
	t.Helper()
	w := get(func(w http.ResponseWriter, r *http.Request) { k.Media.Image(w, r, src, o) }, "GET")
	if w.Code != 200 {
		return nil, "", w
	}
	img, format, err := image.Decode(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	return img, format, w
}

func TestImage(t *testing.T) {
	k := testkit.New(t)
	poster := filepath.Join(k.Config.Movies, "O Grande Teste (2021)", "poster.png")
	img, format, w := image_(t, k, poster, media.ImageOpts{MaxWidth: 32})
	if img == nil || format != "png" || img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatalf("max: %v %s", w.Code, format)
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Fatalf("cache-control: %q", cc)
	}
	img, format, _ = image_(t, k, poster, media.ImageOpts{Width: 40, Height: 20, Format: "jpg", Quality: 80})
	if img == nil || format != "jpeg" || img.Bounds().Dx() != 40 || img.Bounds().Dy() != 20 {
		t.Fatalf("fill: %v %s", img.Bounds(), format)
	}
	// não aumenta
	if img, _, _ := image_(t, k, poster, media.ImageOpts{MaxWidth: 500, MaxHeight: 500}); img.Bounds().Dx() != 64 {
		t.Fatalf("aumentou: %v", img.Bounds())
	}
	ents, _ := os.ReadDir(filepath.Join(k.Config.Data, "cache", "images"))
	if len(ents) != 3 {
		t.Fatalf("cache: %d arquivos", len(ents))
	}
	bip := filepath.Join(k.Config.Music, "Banda Teste", "Disco Teste", "01 - Bip.mp3")
	if img, format, w := image_(t, k, "embedded:"+bip, media.ImageOpts{MaxHeight: 16}); img == nil || format != "jpeg" || img.Bounds().Dy() != 16 {
		t.Fatalf("embutida: %d", w.Code)
	}
	for _, src := range []string{"embedded:" + filepath.Join(k.Config.Music, "Avulsas", "Artista X - Solta.mp3"),
		filepath.Join(k.Root, "nada.png"), "embedded:" + filepath.Join(k.Root, "nada.mp3")} {
		if _, _, w := image_(t, k, src, media.ImageOpts{}); w.Code != 404 {
			t.Fatalf("%s: %d", src, w.Code)
		}
	}
	if k.Config.FFmpeg != "" {
		clip := filepath.Join(k.Config.Movies, "Curta Pixel (2019).mp4")
		if img, _, w := image_(t, k, "frame:"+clip, media.ImageOpts{}); img == nil || img.Bounds().Dx() != 96 {
			t.Fatalf("quadro: %d", w.Code)
		}
	}
}

func TestTranscodeAudio(t *testing.T) {
	k := testkit.New(t)
	if k.Config.FFmpeg == "" {
		t.Skip("sem ffmpeg")
	}
	flac := filepath.Join(k.Config.Music, "Banda Teste", "Disco Teste", "02 - Bop.flac")
	ctx := context.Background()
	for format, magic := range map[string]string{"mp3": "", "opus": "OggS", "ogg": "OggS", "aac": "\xff", "flac": "fLaC"} {
		var out bytes.Buffer
		if err := k.Media.TranscodeAudio(ctx, &out, flac, media.AudioOpts{Format: format, Bitrate: 96, Offset: 0.5}); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if out.Len() < 100 || !strings.HasPrefix(out.String(), magic) {
			t.Fatalf("%s: %d bytes %q", format, out.Len(), out.String()[:4])
		}
	}
	if err := k.Media.TranscodeAudio(ctx, io.Discard, filepath.Join(k.Root, "nada.flac"), media.AudioOpts{}); err == nil {
		t.Fatal("arquivo que não existe devia dar erro")
	}
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	if err := k.Media.TranscodeAudio(cctx, io.Discard, flac, media.AudioOpts{}); err == nil {
		t.Fatal("ctx cancelado devia dar erro")
	}
	k.Config.FFmpeg = ""
	if err := k.Media.TranscodeAudio(ctx, io.Discard, flac, media.AudioOpts{}); err != media.ErrNoFFmpeg {
		t.Fatalf("sem ffmpeg: %v", err)
	}
}

func TestSubtitle(t *testing.T) {
	k := testkit.New(t)
	srt := filepath.Join(k.Config.Movies, "O Grande Teste (2021)", "O Grande Teste (2021).pt.srt")
	sub := func(p, format string) *httptest.ResponseRecorder {
		return get(func(w http.ResponseWriter, r *http.Request) { k.Media.Subtitle(w, r, p, format) }, "GET")
	}
	w := sub(srt, "vtt")
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "WEBVTT\n\n") || !strings.Contains(w.Body.String(), "00:00:00.000 --> 00:00:01.500") ||
		!strings.HasPrefix(w.Header().Get("Content-Type"), "text/vtt") {
		t.Fatalf("vtt: %d %q", w.Code, w.Body.String())
	}
	if w := sub(srt, "srt"); w.Code != 200 || !strings.Contains(w.Body.String(), "00:00:00,000 --> 00:00:01,500") {
		t.Fatalf("srt: %d %q", w.Code, w.Body.String())
	}
	// vtt → srt (latin1 vira utf-8)
	vtt := filepath.Join(t.TempDir(), "a.vtt")
	os.WriteFile(vtt, []byte("WEBVTT\n\nNOTE x\n\n00:01.000 --> 00:02.500 align:start\nol\xe1\n"), 0o644)
	if w := sub(vtt, "srt"); w.Code != 200 || w.Body.String() != "1\n00:00:01,000 --> 00:00:02,500\nolá\n\n" {
		t.Fatalf("vtt→srt: %q", w.Body.String())
	}
	if k.Config.FFmpeg != "" {
		if w := sub(srt, "ass"); w.Code != 200 || !strings.Contains(w.Body.String(), "[Script Info]") {
			t.Fatalf("ass: %d %q", w.Code, w.Body.String())
		}
	}
	if w := sub(filepath.Join(k.Root, "nada.srt"), "vtt"); w.Code != 404 {
		t.Fatalf("sem arquivo: %d", w.Code)
	}
	if w := sub(srt, "pgs"); w.Code != 400 {
		t.Fatalf("formato: %d", w.Code)
	}
}
