package media

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// subFormats: nome pedido → extensão, formato do ffmpeg e Content-Type.
var subFormats = map[string][3]string{
	"vtt":    {"vtt", "webvtt", "text/vtt; charset=utf-8"},
	"webvtt": {"vtt", "webvtt", "text/vtt; charset=utf-8"},
	"srt":    {"srt", "srt", "application/x-subrip; charset=utf-8"},
	"subrip": {"srt", "srt", "application/x-subrip; charset=utf-8"},
	"ass":    {"ass", "ass", "text/x-ssa; charset=utf-8"},
	"ssa":    {"ass", "ass", "text/x-ssa; charset=utf-8"},
}

// Subtitle manda a legenda externa path no formato pedido (vtt, srt, ass).
// Mesmo formato vai direto; srt↔vtt converte aqui; o resto pelo ffmpeg.
func (m *Media) Subtitle(w http.ResponseWriter, r *http.Request, path string, format string) {
	f, ok := subFormats[strings.ToLower(format)]
	if !ok {
		http.Error(w, "formato de legenda não suportado", http.StatusBadRequest)
		return
	}
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}
	data, err := m.convertSub(r.Context(), path, f[0], f[1])
	if err != nil {
		http.Error(w, "não deu pra converter a legenda", http.StatusUnsupportedMediaType)
		return
	}
	w.Header().Set("Content-Type", f[2])
	http.ServeContent(w, r, "", fi.ModTime(), bytes.NewReader(data))
}

func (m *Media) convertSub(ctx context.Context, path, to, ffFormat string) ([]byte, error) {
	from := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if from == "ssa" {
		from = "ass"
	}
	if from == to || (from == "srt" && to == "vtt") || (from == "vtt" && to == "srt") {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := toUTF8(raw)
		switch {
		case from == to:
			return []byte(text), nil
		case to == "vtt":
			return []byte(srtToVTT(text)), nil
		default:
			return []byte(vttToSRT(text)), nil
		}
	}
	return m.ffmpegSub(ctx, path, ffFormat)
}

func (m *Media) ffmpegSub(ctx context.Context, path, ffFormat string) ([]byte, error) {
	if m.Cfg.FFmpeg == "" {
		return nil, ErrNoFFmpeg
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	args := []string{"-v", "error", "-nostdin"}
	if raw, err := os.ReadFile(path); err == nil && !utf8.Valid(raw) {
		args = append(args, "-sub_charenc", "CP1252")
	}
	args = append(args, "-i", path, "-map", "0:s:0", "-f", ffFormat, "pipe:1")
	return exec.CommandContext(ctx, m.Cfg.FFmpeg, args...).Output()
}

// toUTF8: legenda velha costuma vir em latin1/cp1252; tira o BOM.
func toUTF8(b []byte) string {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	if utf8.Valid(b) {
		return string(b)
	}
	rs := make([]rune, len(b))
	for i, c := range b {
		rs[i] = rune(c)
	}
	return string(rs)
}

var (
	reSRTTime = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}),(\d{3})`)
	reVTTTime = regexp.MustCompile(`(\d{2}:\d{2}:\d{2})\.(\d{3})`)
	reVTTCue  = regexp.MustCompile(`(?m)^(?:(\d{2}):)?(\d{2}:\d{2}\.\d{3}) --> (?:(\d{2}):)?(\d{2}:\d{2}\.\d{3})[^\n]*$`)
)

func srtToVTT(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return "WEBVTT\n\n" + strings.TrimLeft(reSRTTime.ReplaceAllString(s, "$1.$2"), "\n")
}

// vttToSRT: tira o cabeçalho e numera os blocos.
func vttToSRT(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = reVTTCue.ReplaceAllStringFunc(s, func(line string) string {
		m := reVTTCue.FindStringSubmatch(line)
		return hours(m[1]) + m[2] + " --> " + hours(m[3]) + m[4]
	})
	var out strings.Builder
	n := 0
	for _, block := range strings.Split(s, "\n\n") {
		lines := strings.Split(strings.Trim(block, "\n"), "\n")
		at := -1
		for i, l := range lines {
			if strings.Contains(l, " --> ") {
				at = i
				break
			}
		}
		if at < 0 {
			continue // WEBVTT, NOTE, STYLE
		}
		n++
		out.WriteString(strconv.Itoa(n) + "\n" + reVTTTime.ReplaceAllString(lines[at], "$1,$2") + "\n")
		out.WriteString(strings.Join(lines[at+1:], "\n") + "\n\n")
	}
	return out.String()
}

func hours(h string) string {
	if h == "" {
		return "00:"
	}
	return h + ":"
}
