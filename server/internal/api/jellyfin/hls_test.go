package jellyfin

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestHLS(t *testing.T) {
	e := setup(t)
	if e.k.Config.FFmpeg == "" {
		t.Skip("sem ffmpeg")
	}
	var pi pbInfo
	profile := map[string]any{"DirectPlayProfiles": []map[string]string{{"Type": "Video", "Container": "mp4", "VideoCodec": "h264", "AudioCodec": "aac"}}}
	e.call("POST", "/Items/"+e.ids.MovieBig+"/PlaybackInfo", map[string]any{"DeviceProfile": profile}, &pi)
	master := pi.MediaSources[0].TranscodingUrl
	if master == "" {
		t.Fatal("sem TranscodingUrl")
	}
	resp, raw := e.req("GET", master, nil, "Authorization", "")
	eq(t, "master", resp.StatusCode, 200)
	eq(t, "mime", resp.Header.Get("Content-Type"), "application/vnd.apple.mpegurl")
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if !strings.HasPrefix(lines[1], "#EXT-X-STREAM-INF:") || !strings.HasPrefix(lines[2], "main.m3u8?") {
		t.Fatalf("%s", raw)
	}
	base := "/Videos/" + e.ids.MovieBig + "/"
	resp, raw = e.req("GET", base+lines[2], nil, "Authorization", "")
	eq(t, "main", resp.StatusCode, 200)
	main := string(raw)
	for _, want := range []string{"#EXT-X-PLAYLIST-TYPE:VOD", "#EXT-X-TARGETDURATION:6", "#EXTINF:2.003000,", "hls1/main/0.ts?", "#EXT-X-ENDLIST"} {
		if !strings.Contains(main, want) {
			t.Fatalf("falta %q:\n%s", want, main)
		}
	}
	var seg string
	for _, l := range strings.Split(main, "\n") {
		if strings.HasPrefix(l, "hls1/") {
			seg = l
		}
	}
	resp, raw = e.req("GET", base+seg, nil, "Authorization", "")
	eq(t, "segmento", resp.StatusCode, 200)
	eq(t, "ts", resp.Header.Get("Content-Type"), "video/mp2t")
	if len(raw) < 188 || raw[0] != 0x47 {
		t.Fatalf("não é MPEG-TS: %d bytes", len(raw))
	}
	resp, _ = e.req("GET", base+strings.Replace(seg, "/0.ts", "/9.ts", 1), nil, "Authorization", "")
	eq(t, "fora da duração", resp.StatusCode, 404)

	// parou: o ffmpeg e a pasta somem
	e.s.hls.mu.Lock()
	var dir string
	for _, j := range e.s.hls.jobs {
		dir = j.dir
	}
	e.s.hls.mu.Unlock()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("pasta do HLS: %v", err)
	}
	e.call("POST", "/Sessions/Playing/Stopped", map[string]any{"ItemId": e.ids.MovieBig, "PlaySessionId": pi.PlaySessionId}, nil)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("pasta ficou")
	}
	e.s.hls.mu.Lock()
	n := len(e.s.hls.jobs)
	e.s.hls.mu.Unlock()
	eq(t, "jobs", n, 0)
}

func TestHLSIdleCleanup(t *testing.T) {
	e := setup(t)
	if e.k.Config.FFmpeg == "" {
		t.Skip("sem ffmpeg")
	}
	e.s.hls.idle = 50 * time.Millisecond
	resp, _ := e.req("GET", "/Videos/"+e.ids.Ep1+"/main.m3u8?PlaySessionId=idle&AudioStreamIndex=1", nil)
	eq(t, "main", resp.StatusCode, 200)
	deadline := time.Now().Add(10 * time.Second)
	for {
		e.s.hls.mu.Lock()
		n := len(e.s.hls.jobs)
		e.s.hls.mu.Unlock()
		if n == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("job ocioso não foi limpo")
		}
		time.Sleep(100 * time.Millisecond)
	}
	// DELETE /Videos/ActiveEncodings
	e.req("GET", "/Videos/"+e.ids.Ep1+"/main.m3u8?PlaySessionId=x2", nil)
	eq(t, "delete", e.call("DELETE", "/Videos/ActiveEncodings?deviceId=aparelho-1&playSessionId=x2", nil, nil), 204)
}
