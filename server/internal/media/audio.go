package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

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

// ErrNoFFmpeg vem quando falta o ffmpeg pra transcodificar.
var ErrNoFFmpeg = errors.New("ffmpeg não configurado")

// codecs: formato → codec e contêiner do ffmpeg.
var codecs = map[string][2]string{
	"mp3":  {"libmp3lame", "mp3"},
	"opus": {"libopus", "ogg"},
	"ogg":  {"libvorbis", "ogg"},
	"aac":  {"aac", "adts"},
	"flac": {"flac", "flac"},
}

// audioArgs monta a linha do ffmpeg.
func audioArgs(src string, o AudioOpts) []string {
	c, ok := codecs[o.Format]
	if !ok {
		c = codecs["mp3"]
	}
	if o.Bitrate <= 0 {
		o.Bitrate = 192
	}
	args := []string{"-v", "error", "-nostdin"}
	if o.Offset > 0 {
		args = append(args, "-ss", strconv.FormatFloat(o.Offset, 'f', 3, 64))
	}
	args = append(args, "-i", src, "-vn", "-map", "0:a:0", "-map_metadata", "-1", "-c:a", c[0])
	if c[0] != "flac" {
		args = append(args, "-b:a", strconv.Itoa(o.Bitrate)+"k")
	}
	if c[0] == "libopus" {
		args = append(args, "-ar", "48000")
	}
	return append(args, "-f", c[1], "pipe:1")
}

// TranscodeAudio escreve em out o áudio de src transcodificado pelo ffmpeg
// até acabar ou ctx ser cancelado. Erro se não houver ffmpeg.
func (m *Media) TranscodeAudio(ctx context.Context, out io.Writer, src string, o AudioOpts) error {
	if m.Cfg.FFmpeg == "" {
		return ErrNoFFmpeg
	}
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, m.Cfg.FFmpeg, audioArgs(src, o)...)
	cmd.Stdout, cmd.Stderr = out, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(lastLine(stderr.String())))
	}
	return nil
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}
