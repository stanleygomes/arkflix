package jellyfin

import (
	"bytes"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// GET /Videos/{id}/{msid}/Subtitles/{index}[/{ticks}]/Stream.{format}
// Externa vai pelo media.Subtitle; embutida de texto, o ffmpeg extrai.
func (s *Server) subtitle(c *rc) {
	it := c.playable(store.KindMovie, store.KindEpisode)
	if it == nil {
		return
	}
	idx, err := strconv.Atoi(c.p["index"])
	st := streamByIndex(it, idx)
	if err != nil || st == nil || st.Type != "subtitle" || !isTextSub(st.Codec) {
		c.notFound()
		return
	}
	format := strings.ToLower(c.p["format"])
	switch format {
	case "webvtt":
		format = "vtt"
	case "subrip":
		format = "srt"
	}
	if format != "vtt" && format != "srt" && format != "ass" && format != "ssa" {
		c.fail(http.StatusBadRequest, "formato de legenda não suportado")
		return
	}
	if st.External != "" {
		s.d.Media.Subtitle(c.w, c.r, st.External, format)
		return
	}
	out, err := s.ffmpegSub(it.Path, "0:"+strconv.Itoa(st.Index), format)
	if err != nil {
		c.fail(http.StatusInternalServerError, err.Error())
		return
	}
	ct := map[string]string{"vtt": "text/vtt; charset=utf-8", "srt": "application/x-subrip; charset=utf-8"}[format]
	if ct == "" {
		ct = "text/x-ssa; charset=utf-8"
	}
	c.w.Header().Set("Content-Type", ct)
	if c.r.Method != http.MethodHead {
		c.w.Write(out)
	}
}

// ffmpegSub extrai uma legenda embutida de texto.
func (s *Server) ffmpegSub(src, mapping, format string) ([]byte, error) {
	if s.d.Config.FFmpeg == "" {
		return nil, errNoFFmpeg
	}
	f := map[string]string{"vtt": "webvtt", "srt": "srt", "ass": "ass", "ssa": "ass"}[format]
	var out, errb bytes.Buffer
	cmd := exec.Command(s.d.Config.FFmpeg, "-hide_banner", "-loglevel", "error", "-nostdin", "-i", src,
		"-map", mapping, "-f", f, "pipe:1")
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, &ffErr{err: err, msg: tail(errb.String())}
	}
	return out.Bytes(), nil
}

type ffErr struct {
	err error
	msg string
}

func (e *ffErr) Error() string { return "ffmpeg: " + e.err.Error() + ": " + e.msg }

type constErr string

func (e constErr) Error() string { return string(e) }

const errNoFFmpeg = constErr("sem ffmpeg")
