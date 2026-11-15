package jellyfin

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// HLS: a playlist sai inteira (duração / 6 s) antes do ffmpeg rodar; cada
// segmento i cobre [6i, 6i+6). O ffmpeg começa no segmento pedido (-ss) com
// keyframe forçado a cada 6 s, então qualquer execução gera os mesmos
// cortes e um seek longe só reinicia o ffmpeg ali. Tudo numa pasta por
// PlaySessionId em <data>/transcodes.
const segLen = 6.0

type transcoder struct {
	ffmpeg string
	dir    string
	idle   time.Duration // sem pedido por esse tempo: mata e apaga
	wait   time.Duration // espera máxima por um segmento

	mu      sync.Mutex
	jobs    map[string]*hlsJob
	started bool
}

func newTranscoder(ffmpeg, dir string) *transcoder {
	os.RemoveAll(dir) // sobras de uma execução anterior
	return &transcoder{ffmpeg: ffmpeg, dir: dir, idle: 3 * time.Minute, wait: 90 * time.Second, jobs: map[string]*hlsJob{}}
}

// hlsArgs é o que muda a saída do ffmpeg (mudou: recomeça do zero).
type hlsArgs struct {
	audio               int // índice absoluto; -1 = a padrão
	videoBitrate        int
	maxWidth, maxHeight int
}

type ffRun struct {
	proc   *os.Process
	cancel context.CancelFunc
	exited chan struct{}
	stderr *strings.Builder
}

func (r *ffRun) running() bool {
	select {
	case <-r.exited:
		return false
	default:
		return true
	}
}

type hlsJob struct {
	key, token, playSession, device string
	dir                             string
	item                            *store.Item
	args                            hlsArgs

	mu         sync.Mutex
	run        *ffRun
	start      int // primeiro segmento da execução atual
	done       map[int]bool
	lastReq    int
	lastAccess time.Time
	paused     bool
}

func (t *transcoder) janitor() {
	for range time.Tick(2 * time.Second) {
		t.mu.Lock()
		var stale []*hlsJob
		for k, j := range t.jobs {
			j.mu.Lock()
			if time.Since(j.lastAccess) > t.idle {
				stale = append(stale, j)
				delete(t.jobs, k)
			} else {
				j.throttle()
			}
			j.mu.Unlock()
		}
		t.mu.Unlock()
		for _, j := range stale {
			j.close()
		}
	}
}

// throttle congela o ffmpeg quando ele está muito à frente do player.
func (j *hlsJob) throttle() {
	if j.run == nil || !j.run.running() {
		return
	}
	j.refresh()
	ahead := j.head() - j.lastReq
	switch {
	case !j.paused && ahead > 40:
		j.paused = pauseProc(j.run, true)
	case j.paused && ahead < 20:
		pauseProc(j.run, false)
		j.paused = false
	}
}

func (t *transcoder) get(key, token, playSession, device string, it *store.Item, args hlsArgs) *hlsJob {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.started {
		t.started = true
		go t.janitor()
	}
	if j := t.jobs[key]; j != nil {
		if j.args == args && j.item.MTime == it.MTime {
			return j
		}
		delete(t.jobs, key)
		go j.close()
	}
	j := &hlsJob{key: key, token: token, playSession: playSession, device: device, item: it, args: args,
		dir:  filepath.Join(t.dir, hash(key, fmt.Sprint(args), strconv.FormatInt(time.Now().UnixNano(), 10))[:20]),
		done: map[int]bool{}, lastAccess: time.Now()}
	t.jobs[key] = j
	return j
}

func (t *transcoder) stopWhere(match func(*hlsJob) bool) {
	t.mu.Lock()
	var out []*hlsJob
	for k, j := range t.jobs {
		if match(j) {
			out = append(out, j)
			delete(t.jobs, k)
		}
	}
	t.mu.Unlock()
	for _, j := range out {
		j.close()
	}
}

func (t *transcoder) stopSession(ps string) {
	if ps != "" {
		t.stopWhere(func(j *hlsJob) bool { return j.playSession == ps })
	}
}

func (t *transcoder) stopDevice(dev string) {
	if dev != "" {
		t.stopWhere(func(j *hlsJob) bool { return j.device == dev })
	}
}

func (t *transcoder) stopToken(tok string) {
	t.stopWhere(func(j *hlsJob) bool { return j.token == tok })
}

func (j *hlsJob) kill() {
	if j.run == nil {
		return
	}
	if j.paused {
		pauseProc(j.run, false)
		j.paused = false
	}
	j.run.cancel()
	<-j.run.exited
}

func (j *hlsJob) close() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.kill()
	os.RemoveAll(j.dir)
}

var segName = regexp.MustCompile(`^s(\d+)\.ts$`)

func segFile(dir string, i int) string { return filepath.Join(dir, "s"+strconv.Itoa(i)+".ts") }

// refresh marca os segmentos que o ffmpeg já fechou (os que estão na
// playlist dele).
func (j *hlsJob) refresh() {
	f, err := os.Open(filepath.Join(j.dir, "play.m3u8"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := segName.FindStringSubmatch(strings.TrimSpace(sc.Text())); m != nil {
			n, _ := strconv.Atoi(m[1])
			j.done[n] = true
		}
	}
}

// head é o próximo segmento que a execução atual ainda vai fazer.
func (j *hlsJob) head() int {
	i := j.start
	for j.done[i] {
		i++
	}
	return i
}

func (t *transcoder) restart(j *hlsJob, seg int) error {
	j.kill()
	// tira restos pela metade da execução anterior
	entries, _ := os.ReadDir(j.dir)
	for _, e := range entries {
		m := segName.FindStringSubmatch(e.Name())
		if m == nil {
			os.Remove(filepath.Join(j.dir, e.Name()))
			continue
		}
		if n, _ := strconv.Atoi(m[1]); !j.done[n] {
			os.Remove(filepath.Join(j.dir, e.Name()))
		}
	}
	if err := os.MkdirAll(j.dir, 0o700); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, t.ffmpeg, ffmpegHLSArgs(j, seg)...)
	stderr := &strings.Builder{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}
	run := &ffRun{proc: cmd.Process, cancel: cancel, exited: make(chan struct{}), stderr: stderr}
	go func() {
		err := cmd.Wait()
		if err != nil && ctx.Err() == nil {
			slog.Warn("jellyfin: ffmpeg do HLS falhou", "item", j.item.ID, "err", err, "stderr", tail(stderr.String()))
		}
		close(run.exited)
	}()
	j.run, j.start, j.paused = run, seg, false
	return nil
}

func tail(s string) string {
	if len(s) > 500 {
		return s[len(s)-500:]
	}
	return s
}

func ffmpegHLSArgs(j *hlsJob, seg int) []string {
	start := float64(seg) * segLen
	a := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y"}
	if start > 0 {
		a = append(a, "-ss", fmt.Sprintf("%.3f", start))
	}
	a = append(a, "-i", j.item.Path, "-map", "0:v:0")
	if j.args.audio >= 0 && streamByIndex(j.item, j.args.audio) != nil && streamByIndex(j.item, j.args.audio).External == "" {
		a = append(a, "-map", "0:"+strconv.Itoa(j.args.audio))
	} else {
		a = append(a, "-map", "0:a:0?")
	}
	vf := "scale=trunc(iw/2)*2:trunc(ih/2)*2"
	if j.args.maxWidth > 0 || j.args.maxHeight > 0 {
		w, h := j.args.maxWidth, j.args.maxHeight
		if w <= 0 {
			w = 99999
		}
		if h <= 0 {
			h = 99999
		}
		vf = fmt.Sprintf("scale=w='min(iw,%d)':h='min(ih,%d)':force_original_aspect_ratio=decrease,%s", w, h, vf)
	}
	vb := j.args.videoBitrate
	a = append(a, "-vf", vf, "-c:v", "libx264", "-preset", "veryfast", "-pix_fmt", "yuv420p", "-profile:v", "high",
		"-crf", "23", "-maxrate", strconv.Itoa(vb), "-bufsize", strconv.Itoa(vb*2), "-sc_threshold", "0",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%g)", segLen),
		"-c:a", "aac", "-ac", "2", "-b:a", "192k",
		"-output_ts_offset", fmt.Sprintf("%.3f", start),
		"-f", "hls", "-hls_time", fmt.Sprintf("%g", segLen), "-hls_list_size", "0", "-hls_flags", "temp_file",
		"-hls_segment_type", "mpegts", "-start_number", strconv.Itoa(seg),
		"-hls_segment_filename", filepath.Join(j.dir, "s%d.ts"), filepath.Join(j.dir, "play.m3u8"))
	return a
}

// --- handlers ---

func (c *rc) hlsArgs(it *store.Item) hlsArgs {
	a := hlsArgs{audio: c.qi("AudioStreamIndex", -1), maxWidth: c.qi("MaxWidth", 0), maxHeight: c.qi("MaxHeight", 0)}
	a.videoBitrate = c.qi("VideoBitrate", 0)
	if a.videoBitrate <= 0 {
		if m := c.qi("MaxStreamingBitrate", 0); m > 0 {
			a.videoBitrate = max(m-192_000, 500_000)
		}
	}
	if a.videoBitrate <= 0 {
		a.videoBitrate = 8_000_000
	}
	return a
}

func (c *rc) hlsJob(it *store.Item) *hlsJob {
	ps := c.qs("PlaySessionId")
	dev := c.qs("DeviceId")
	if dev == "" {
		dev = c.ah.DeviceID
	}
	key := it.ID + "|" + ps
	if ps == "" {
		key = it.ID + "|" + dev + "|" + c.tok.Token
	}
	return c.s.hls.get(key, c.tok.Token, ps, dev, it, c.hlsArgs(it))
}

func segments(it *store.Item) int { return max(int(math.Ceil(it.Duration/segLen)), 1) }

// GET /Videos/{id}/master.m3u8
func (s *Server) hlsMaster(c *rc) {
	it := c.playable(store.KindMovie, store.KindEpisode)
	if it == nil {
		return
	}
	if s.d.Config.FFmpeg == "" {
		c.fail(http.StatusServiceUnavailable, "sem ffmpeg")
		return
	}
	a := c.hlsArgs(it)
	w, h := it.Width, it.Height
	if a.maxWidth > 0 && w > a.maxWidth && w > 0 {
		w, h = a.maxWidth, h*a.maxWidth/w
	}
	if a.maxHeight > 0 && h > a.maxHeight && h > 0 {
		w, h = w*a.maxHeight/h, a.maxHeight
	}
	bw := a.videoBitrate + 192_000
	if it.Bitrate > 0 && it.Bitrate+192_000 < bw {
		bw = it.Bitrate + 192_000
	}
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,AVERAGE-BANDWIDTH=%d,CODECS=\"avc1.640028,mp4a.40.2\"", bw, bw)
	if w > 0 && h > 0 {
		fmt.Fprintf(&b, ",RESOLUTION=%dx%d", w&^1, h&^1)
	}
	b.WriteString("\nmain.m3u8?" + c.r.URL.RawQuery + "\n")
	c.playlist(b.String())
}

// GET /Videos/{id}/main.m3u8: a playlist inteira (VOD), já esquentando o
// ffmpeg no ponto de partida.
func (s *Server) hlsMain(c *rc) {
	it := c.playable(store.KindMovie, store.KindEpisode)
	if it == nil {
		return
	}
	if s.d.Config.FFmpeg == "" {
		c.fail(http.StatusServiceUnavailable, "sem ffmpeg")
		return
	}
	n := segments(it)
	j := c.hlsJob(it)
	j.mu.Lock()
	j.lastAccess = time.Now()
	if j.run == nil {
		first := min(int(float64(c.qi64("StartTimeTicks"))/ticksPerSecond/segLen), n-1)
		j.lastReq = first
		if err := s.hls.restart(j, first); err != nil {
			j.mu.Unlock()
			c.err(err)
			return
		}
	}
	j.mu.Unlock()
	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-VERSION:3\n")
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n#EXT-X-MEDIA-SEQUENCE:0\n", int(segLen))
	q := c.r.URL.RawQuery
	for i := range n {
		d := min(segLen, it.Duration-float64(i)*segLen)
		if d <= 0 {
			d = segLen
		}
		fmt.Fprintf(&b, "#EXTINF:%.6f, nocache\nhls1/main/%d.ts?%s\n", d, i, q)
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	c.playlist(b.String())
}

func (c *rc) playlist(body string) {
	c.w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	c.w.Header().Set("Cache-Control", "no-cache")
	if c.r.Method != http.MethodHead {
		c.w.Write([]byte(body))
	}
}

// GET /Videos/{id}/hls1/{playlistId}/{seg}.ts
func (s *Server) hlsSegment(c *rc) {
	it := c.playable(store.KindMovie, store.KindEpisode)
	if it == nil {
		return
	}
	seg, err := strconv.Atoi(c.p["seg"])
	if err != nil || seg < 0 || seg >= segments(it) || s.d.Config.FFmpeg == "" {
		c.notFound()
		return
	}
	j := c.hlsJob(it)
	j.mu.Lock()
	j.lastAccess, j.lastReq = time.Now(), seg
	j.refresh()
	if !j.done[seg] {
		running := j.run != nil && j.run.running()
		if !running || seg < j.start || seg > j.head()+2 {
			if err := s.hls.restart(j, seg); err != nil {
				j.mu.Unlock()
				c.err(err)
				return
			}
		} else if j.paused {
			pauseProc(j.run, false)
			j.paused = false
		}
	}
	j.mu.Unlock()

	deadline := time.Now().Add(s.hls.wait)
	for {
		j.mu.Lock()
		j.refresh()
		ok, run := j.done[seg], j.run
		j.mu.Unlock()
		if ok {
			break
		}
		if run == nil || !run.running() {
			j.mu.Lock()
			j.refresh()
			ok = j.done[seg]
			j.mu.Unlock()
			if !ok {
				c.fail(http.StatusInternalServerError, "o ffmpeg não gerou o segmento")
				return
			}
			break
		}
		if time.Now().After(deadline) {
			c.fail(http.StatusGatewayTimeout, "segmento demorou demais")
			return
		}
		select {
		case <-c.ctx().Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
	c.w.Header().Set("Content-Type", "video/mp2t")
	c.w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(c.w, c.r, segFile(j.dir, seg))
}

// DELETE /Videos/ActiveEncodings?deviceId=&playSessionId=
func (s *Server) stopEncodings(c *rc) {
	if ps := c.qs("PlaySessionId"); ps != "" {
		s.hls.stopSession(ps)
	} else {
		s.hls.stopDevice(c.qs("DeviceId"))
	}
	c.noContent()
}

// progressive: /Videos/{id}/stream.{ext} não estático vira um ffmpeg só
// (sem Range), copiando o que o cliente aceita.
func (s *Server) progressive(c *rc, it *store.Item) {
	ext := normName(c.p["container"])
	if ext == "" {
		ext = normName(it.Container)
	}
	format, mime := "matroska", "video/x-matroska"
	switch ext {
	case "mp4":
		format, mime = "mp4", "video/mp4"
	case "ts":
		format, mime = "mpegts", "video/mp2t"
	case "webm":
		format, mime = "webm", "video/webm"
	}
	a := []string{"-hide_banner", "-loglevel", "error", "-nostdin"}
	if st := c.qi64("StartTimeTicks"); st > 0 {
		a = append(a, "-ss", fmt.Sprintf("%.3f", float64(st)/ticksPerSecond))
	}
	a = append(a, "-i", it.Path, "-map", "0:v:0")
	audio := streamByIndex(it, c.qi("AudioStreamIndex", defaultAudio(it)))
	if audio != nil && audio.Type == "audio" && audio.External == "" {
		a = append(a, "-map", "0:"+strconv.Itoa(audio.Index))
	} else {
		a = append(a, "-map", "0:a:0?")
		audio = firstStream(it, "audio")
	}
	video := firstStream(it, "video")
	vcodecs, acodecs := c.qs("VideoCodec"), c.qs("AudioCodec")
	switch {
	case format == "webm":
		a = append(a, "-c:v", "libvpx-vp9", "-deadline", "realtime", "-cpu-used", "8", "-row-mt", "1", "-b:v", "4M", "-c:a", "libopus")
	default:
		if video != nil && (vcodecs == "" || listHas(vcodecs, video.Codec)) && (format != "mp4" || video.Codec == "h264" || video.Codec == "hevc") {
			a = append(a, "-c:v", "copy")
		} else {
			a = append(a, "-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p")
		}
		if audio != nil && acodecs != "" && listHas(acodecs, audio.Codec) {
			a = append(a, "-c:a", "copy")
		} else {
			a = append(a, "-c:a", "aac", "-ac", "2", "-b:a", "192k")
		}
	}
	if format == "mp4" {
		a = append(a, "-movflags", "frag_keyframe+empty_moov+default_base_moof")
	}
	a = append(a, "-f", format, "pipe:1")
	c.w.Header().Set("Content-Type", mime)
	c.w.Header().Set("Cache-Control", "no-cache")
	if c.r.Method == http.MethodHead {
		return
	}
	cmd := exec.CommandContext(c.ctx(), s.d.Config.FFmpeg, a...)
	cmd.Stdout = c.w
	if err := cmd.Run(); err != nil && c.ctx().Err() == nil {
		slog.Warn("jellyfin: transcodificação progressiva falhou", "item", it.ID, "err", err)
	}
}
