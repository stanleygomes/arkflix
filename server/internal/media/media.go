// Package media entrega arquivos: original com Range, imagem redimensionada
// (com cache em disco) e áudio transcodificado pelo ffmpeg.
package media

import (
	"context"
	"io"
	"net/http"

	"github.com/stanleygomes/arkflix/server/internal/config"
)

// Media guarda o caminho do cache de imagens e os binários do ffmpeg.
type Media struct {
	Cfg   *config.Config
	Cache string // <data>/cache
}

// New cria o Media com cache em <data>/cache.
func New(cfg *config.Config) *Media { return &Media{Cfg: cfg, Cache: cfg.Data + "/cache"} }

// ServeFile manda o arquivo com Range, Content-Type pela extensão e
// Accept-Ranges; HEAD funciona. 404 se não existir.
func (m *Media) ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	http.ServeFile(w, r, path) // A FAZER: tipos de mídia (mkv, flac...) e sem listar pasta
}

// ImageOpts pede um tamanho: 0 = sem limite. Format "" = mantém (jpg/png);
// "webp" não é suportado e vira jpg.
type ImageOpts struct {
	MaxWidth, MaxHeight int
	Width, Height       int // exato (Jellyfin fillWidth/fillHeight: corta pra preencher)
	Quality             int // jpeg, padrão 90
	Format              string
}

// Image manda a imagem src redimensionada (cache em disco por src+mtime+
// opções). src pode ser um arquivo de imagem, "embedded:<audio>" (capa
// dentro da tag) ou "frame:<vídeo>" (quadro tirado pelo ffmpeg a 10% do
// filme). 404 se não tiver imagem.
func (m *Media) Image(w http.ResponseWriter, r *http.Request, src string, o ImageOpts) {
	http.NotFound(w, r) // A FAZER
}

// AudioOpts pede uma transcodificação.
type AudioOpts struct {
	Format  string  // mp3 | opus | ogg | aac | flac (padrão mp3)
	Bitrate int     // kbit/s (padrão 192; ignorado em flac)
	Offset  float64 // segundos a pular do começo
}

// ContentType do formato transcodificado.
func ContentType(format string) string {
	switch format {
	case "opus", "ogg":
		return "audio/ogg"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}

// TranscodeAudio escreve em out o áudio de src transcodificado pelo ffmpeg
// até acabar ou ctx ser cancelado. Erro se não houver ffmpeg.
func (m *Media) TranscodeAudio(ctx context.Context, out io.Writer, src string, o AudioOpts) error {
	return nil // A FAZER
}
