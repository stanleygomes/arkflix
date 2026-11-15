package library

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// probed é o que o ffprobe diz de um arquivo.
type probed struct {
	Duration float64
	Bitrate  int
	Width    int
	Height   int
	Streams  []store.Stream
}

type ffStream struct {
	Index         int               `json:"index"`
	CodecType     string            `json:"codec_type"`
	CodecName     string            `json:"codec_name"`
	Profile       string            `json:"profile"`
	Level         int               `json:"level"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	AvgFrameRate  string            `json:"avg_frame_rate"`
	RFrameRate    string            `json:"r_frame_rate"`
	PixFmt        string            `json:"pix_fmt"`
	ColorTransfer string            `json:"color_transfer"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	SampleRate    string            `json:"sample_rate"`
	BitRate       string            `json:"bit_rate"`
	Disposition   map[string]int    `json:"disposition"`
	Tags          map[string]string `json:"tags"`
	SideData      []map[string]any  `json:"side_data_list"`
}

type ffOutput struct {
	Streams []ffStream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
}

// probe roda o ffprobe. bin vazio = erro (quem chama degrada).
func probe(ctx context.Context, bin, path string) (*probed, error) {
	if bin == "" {
		return nil, fmt.Errorf("sem ffprobe")
	}
	out, err := exec.CommandContext(ctx, bin, "-v", "quiet", "-print_format", "json",
		"-show_format", "-show_streams", path).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe %s: %w", path, err)
	}
	var ff ffOutput
	if err := json.Unmarshal(out, &ff); err != nil {
		return nil, fmt.Errorf("ffprobe %s: %w", path, err)
	}
	return fromFF(&ff), nil
}

func fromFF(ff *ffOutput) *probed {
	p := &probed{Duration: round3(atof(ff.Format.Duration)), Bitrate: atoi(ff.Format.BitRate)}
	for _, s := range ff.Streams {
		if s.Disposition["attached_pic"] == 1 {
			continue // capa embutida não é faixa
		}
		st, ok := toStream(s)
		if !ok {
			continue
		}
		if st.Type == "video" && p.Width == 0 {
			p.Width, p.Height = st.Width, st.Height
		}
		p.Streams = append(p.Streams, st)
	}
	markDefaults(p.Streams)
	return p
}

func toStream(s ffStream) (store.Stream, bool) {
	switch s.CodecType {
	case "video", "audio", "subtitle":
	default:
		return store.Stream{}, false
	}
	st := store.Stream{Index: s.Index, Type: s.CodecType, Codec: s.CodecName,
		Language: lang(s.Tags["language"]), Title: s.Tags["title"],
		Default: s.Disposition["default"] == 1, Forced: s.Disposition["forced"] == 1,
		Bitrate: atoi(s.BitRate)}
	if s.Profile != "" && !numeric(s.Profile) {
		// perfil numérico (vp8 "0") não diz nada; pix_fmt só junto do perfil
		st.Profile, st.PixFmt = s.Profile, s.PixFmt
	}
	if s.Level > 0 {
		st.Level = s.Level
	}
	switch s.CodecType {
	case "video":
		st.Width, st.Height = s.Width, s.Height
		st.FPS = fps(s.AvgFrameRate)
		if st.FPS == 0 {
			st.FPS = fps(s.RFrameRate)
		}
		st.Range = videoRange(s)
	case "audio":
		st.Channels, st.Layout, st.Rate = s.Channels, s.ChannelLayout, atoi(s.SampleRate)
	}
	return st, true
}

// markDefaults: sem nenhuma faixa padrão de vídeo/áudio, a primeira vira
// padrão (webm/mkv costumam não marcar).
func markDefaults(ss []store.Stream) {
	for _, typ := range []string{"video", "audio"} {
		first, has := -1, false
		for i, s := range ss {
			if s.Type != typ {
				continue
			}
			if first < 0 {
				first = i
			}
			has = has || s.Default
		}
		if first >= 0 && !has {
			ss[first].Default = true
		}
	}
}

func videoRange(s ffStream) string {
	for _, sd := range s.SideData {
		if t, _ := sd["side_data_type"].(string); strings.Contains(t, "DOVI") {
			return "DOVI"
		}
	}
	switch s.ColorTransfer {
	case "smpte2084":
		return "HDR10"
	case "arib-std-b67":
		return "HLG"
	}
	return "SDR"
}

func fps(r string) float64 {
	a, b, ok := strings.Cut(r, "/")
	if !ok {
		return round3(atof(r))
	}
	d := atof(b)
	if d == 0 {
		return 0
	}
	return round3(atof(a) / d)
}

// lang tira o "und" (indefinido).
func lang(s string) string {
	if s == "und" {
		return ""
	}
	return s
}

func atoi(s string) int     { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
func atof(s string) float64 { f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64); return f }
func round3(f float64) float64 {
	return math.Round(f*1000) / 1000
}

func numeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// probeAll roda o ffprobe em paralelo (poucos de cada vez). Falha num
// arquivo só vira log e resultado nil.
func probeAll(ctx context.Context, bin string, paths []string, warn func(string, error)) map[string]*probed {
	out := make(map[string]*probed, len(paths))
	if bin == "" || len(paths) == 0 {
		return out
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	jobs := make(chan string)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				r, err := probe(ctx, bin, p)
				if err != nil && ctx.Err() == nil {
					warn(p, err)
				}
				mu.Lock()
				out[p] = r
				mu.Unlock()
			}
		}()
	}
	for _, p := range paths {
		if ctx.Err() != nil {
			break
		}
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	return out
}
