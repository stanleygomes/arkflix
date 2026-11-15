package media

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // decodifica gif
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // decodifica webp
)

// ImageOpts pede um tamanho: 0 = sem limite. Format "" = mantém (jpg/png);
// "webp" não é suportado e vira jpg.
type ImageOpts struct {
	MaxWidth, MaxHeight int
	Width, Height       int // exato (Jellyfin fillWidth/fillHeight: corta pra preencher)
	Quality             int // jpeg, padrão 90
	Format              string
}

var errNoImage = errors.New("sem imagem")

// Image manda a imagem src redimensionada (cache em disco por src+mtime+
// opções). src pode ser um arquivo de imagem, "embedded:<audio>" (capa
// dentro da tag) ou "frame:<vídeo>" (quadro tirado pelo ffmpeg a 10% do
// filme). 404 se não tiver imagem.
func (m *Media) Image(w http.ResponseWriter, r *http.Request, src string, o ImageOpts) {
	file := sourceFile(src)
	fi, err := os.Stat(file)
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}
	o = normalize(o, src)
	cached := filepath.Join(m.Cache, "images", cacheKey(src, fi.ModTime(), o)+"."+o.Format)
	if _, err := os.Stat(cached); err != nil {
		if err := m.render(r.Context(), src, file, o, cached); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	f, err := os.Open(cached)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/"+map[string]string{"jpg": "jpeg", "png": "png"}[o.Format])
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.ServeContent(w, r, "", fi.ModTime(), f)
}

// sourceFile é o arquivo de verdade por trás de src.
func sourceFile(src string) string {
	for _, p := range []string{"embedded:", "frame:"} {
		if strings.HasPrefix(src, p) {
			return src[len(p):]
		}
	}
	return src
}

func normalize(o ImageOpts, src string) ImageOpts {
	if o.Quality <= 0 || o.Quality > 100 {
		o.Quality = 90
	}
	switch strings.ToLower(o.Format) {
	case "png":
		o.Format = "png"
	case "", "orig", "original":
		// mantém png/gif como png (transparência); o resto vira jpg
		o.Format = "jpg"
		if e := strings.ToLower(filepath.Ext(src)); src == sourceFile(src) && (e == ".png" || e == ".gif") {
			o.Format = "png"
		}
	default:
		o.Format = "jpg"
	}
	return o
}

func cacheKey(src string, mtime time.Time, o ImageOpts) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d|%d|%d|%d|%d|%s", src, mtime.UnixNano(),
		o.MaxWidth, o.MaxHeight, o.Width, o.Height, o.Quality, o.Format)))
	return hex.EncodeToString(h[:])
}

// render decodifica, redimensiona e grava no cache (troca atômica).
func (m *Media) render(ctx context.Context, src, file string, o ImageOpts, dst string) error {
	img, err := m.decode(ctx, src, file)
	if err != nil {
		return err
	}
	img = resize(img, o)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".img-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := encode(tmp, img, o); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

// encode grava jpg ou png.
func encode(w io.Writer, img image.Image, o ImageOpts) error {
	if o.Format == "png" {
		return png.Encode(w, img)
	}
	return jpeg.Encode(w, img, &jpeg.Options{Quality: o.Quality})
}

// decode abre a imagem de qualquer uma das fontes.
func (m *Media) decode(ctx context.Context, src, file string) (image.Image, error) {
	var data []byte
	var err error
	switch {
	case strings.HasPrefix(src, "embedded:"):
		data, err = m.embedded(ctx, file)
	case strings.HasPrefix(src, "frame:"):
		data, err = m.frame(ctx, file)
	default:
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// embedded lê a capa da tag; se a lib não entender o formato, tenta o ffmpeg.
func (m *Media) embedded(ctx context.Context, file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if md, err := tag.ReadFrom(f); err == nil && md.Picture() != nil {
		return md.Picture().Data, nil
	}
	if m.Cfg.FFmpeg == "" {
		return nil, errNoImage
	}
	return m.ffmpegImage(ctx, "-i", file, "-an", "-map", "0:v:0", "-frames:v", "1")
}

// frame tira um quadro a 10% da duração.
func (m *Media) frame(ctx context.Context, file string) ([]byte, error) {
	if m.Cfg.FFmpeg == "" {
		return nil, errNoImage
	}
	at := m.duration(ctx, file) * 0.1
	return m.ffmpegImage(ctx, "-ss", strconv.FormatFloat(at, 'f', 3, 64), "-i", file,
		"-an", "-map", "0:v:0", "-frames:v", "1")
}

func (m *Media) ffmpegImage(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	args = append([]string{"-v", "error", "-nostdin"}, args...)
	args = append(args, "-c:v", "png", "-f", "image2pipe", "pipe:1")
	out, err := exec.CommandContext(ctx, m.Cfg.FFmpeg, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg: %w", err)
	}
	if len(out) == 0 {
		return nil, errNoImage
	}
	return out, nil
}

// duration pelo ffprobe (0 sem ffprobe: pega o primeiro quadro).
func (m *Media) duration(ctx context.Context, file string) float64 {
	if m.Cfg.FFprobe == "" {
		return 0
	}
	out, err := exec.CommandContext(ctx, m.Cfg.FFprobe, "-v", "quiet", "-show_entries", "format=duration",
		"-of", "default=nw=1:nk=1", file).Output()
	if err != nil {
		return 0
	}
	d, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	return d
}

// resize: Width/Height corta pra preencher; Max* só encolhe mantendo a
// proporção.
func resize(img image.Image, o ImageOpts) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	switch {
	case o.Width > 0 && o.Height > 0:
		return fill(img, o.Width, o.Height)
	case o.Width > 0:
		return scale(img, o.Width, max(1, h*o.Width/w))
	case o.Height > 0:
		return scale(img, max(1, w*o.Height/h), o.Height)
	}
	f := 1.0
	if o.MaxWidth > 0 && w > o.MaxWidth {
		f = float64(o.MaxWidth) / float64(w)
	}
	if o.MaxHeight > 0 && float64(h)*f > float64(o.MaxHeight) {
		f = float64(o.MaxHeight) / float64(h)
	}
	if f >= 1 {
		return img
	}
	return scale(img, max(1, int(float64(w)*f+0.5)), max(1, int(float64(h)*f+0.5)))
}

func scale(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

// fill escala pra cobrir w×h e corta o centro.
func fill(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	// recorte da fonte com a proporção do destino
	cw, ch := sw, sw*h/w
	if ch > sh {
		cw, ch = sh*w/h, sh
	}
	x0, y0 := b.Min.X+(sw-cw)/2, b.Min.Y+(sh-ch)/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, image.Rect(x0, y0, x0+cw, y0+ch), draw.Over, nil)
	return dst
}
