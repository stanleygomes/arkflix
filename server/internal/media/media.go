// Package media entrega arquivos: original com Range, imagem redimensionada
// (com cache em disco) e áudio transcodificado pelo ffmpeg.
package media

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/config"
)

// Media guarda o caminho do cache de imagens e os binários do ffmpeg.
type Media struct {
	Cfg   *config.Config
	Cache string // <data>/cache
}

// New cria o Media com cache em <data>/cache.
func New(cfg *config.Config) *Media { return &Media{Cfg: cfg, Cache: cfg.Data + "/cache"} }

// mimeTypes cobre o que o mime do sistema costuma não saber (ou erra).
var mimeTypes = map[string]string{
	"mkv": "video/x-matroska", "mka": "audio/x-matroska", "webm": "video/webm", "mp4": "video/mp4",
	"m4v": "video/x-m4v", "mov": "video/quicktime", "avi": "video/x-msvideo", "wmv": "video/x-ms-wmv",
	"ts": "video/mp2t", "m2ts": "video/mp2t", "mts": "video/mp2t", "mpg": "video/mpeg", "mpeg": "video/mpeg",
	"flv": "video/x-flv", "ogv": "video/ogg", "3gp": "video/3gpp",
	"mp3": "audio/mpeg", "flac": "audio/flac", "ogg": "audio/ogg", "oga": "audio/ogg", "opus": "audio/ogg",
	"m4a": "audio/mp4", "m4b": "audio/mp4", "aac": "audio/aac", "wav": "audio/wav", "wma": "audio/x-ms-wma",
	"aiff": "audio/aiff", "aif": "audio/aiff", "ape": "audio/x-ape", "wv": "audio/x-wavpack", "dsf": "audio/x-dsf",
	"srt": "application/x-subrip", "vtt": "text/vtt; charset=utf-8", "ass": "text/x-ssa", "ssa": "text/x-ssa",
	"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp", "gif": "image/gif",
	"zip": "application/zip", "7z": "application/x-7z-compressed", "rar": "application/vnd.rar",
	"iso": "application/x-iso9660-image", "chd": "application/octet-stream", "nfo": "text/plain; charset=utf-8",
}

// MimeType pela extensão; desconhecido vira application/octet-stream
// (ROMs: .sfc, .bin, .cue...).
func MimeType(path string) string {
	e := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if t, ok := mimeTypes[e]; ok {
		return t
	}
	return "application/octet-stream" // sem depender do mime do sistema
}

// ServeFile manda o arquivo com Range, Content-Type pela extensão e
// Accept-Ranges; HEAD funciona. 404 se não existir (ou for pasta).
func (m *Media) ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", MimeType(path))
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}
