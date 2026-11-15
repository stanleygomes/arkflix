package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type pbInfo struct {
	PlaySessionId string
	MediaSources  []sourceDto
}

// perfil do app arkflix (webview)
func arkflixProfile() map[string]any {
	return map[string]any{
		"MaxStreamingBitrate": 120_000_000,
		"DirectPlayProfiles": []map[string]string{
			{"Type": "Video", "Container": "mp4,m4v", "VideoCodec": "h264,av1", "AudioCodec": "aac,mp3,opus,flac"},
			{"Type": "Video", "Container": "webm", "VideoCodec": "vp8,vp9,av1", "AudioCodec": "vorbis,opus"},
		},
		"TranscodingProfiles": []map[string]any{
			{"Type": "Video", "Container": "ts", "VideoCodec": "h264", "AudioCodec": "aac", "Protocol": "hls",
				"Context": "Streaming", "MaxAudioChannels": "2", "MinSegments": 1, "BreakOnNonKeyFrames": true},
		},
		"SubtitleProfiles": []map[string]string{{"Format": "vtt", "Method": "External"}},
	}
}

func TestPlaybackInfo(t *testing.T) {
	e := setup(t)
	var pi pbInfo
	// sem perfil: direto
	e.get("/Items/"+e.ids.MoviePixel+"/PlaybackInfo?UserId="+e.userID, &pi)
	src := pi.MediaSources[0]
	if !src.SupportsDirectPlay || src.TranscodingUrl != "" || len(pi.PlaySessionId) != 32 {
		t.Fatalf("%+v", pi)
	}
	if !strings.HasPrefix(src.Path, "/Videos/"+e.ids.MoviePixel+"/stream?") || !strings.Contains(src.Path, "api_key="+e.token) {
		t.Fatal(src.Path)
	}
	eq(t, "protocol", src.Protocol, "File")
	eq(t, "audio padrão", *src.DefaultAudioStreamIndex, 1)

	// webm com o perfil do arkflix: direto, legenda externa em vtt
	e.call("POST", "/Items/"+e.ids.MovieBig+"/PlaybackInfo", map[string]any{"UserId": e.userID, "DeviceProfile": arkflixProfile(),
		"EnableDirectPlay": true, "EnableDirectStream": true, "EnableTranscoding": true}, &pi)
	src = pi.MediaSources[0]
	eq(t, "webm direto", src.SupportsDirectPlay, true)
	var sub *streamDto
	for i := range src.MediaStreams {
		if src.MediaStreams[i].Type == "Subtitle" {
			sub = &src.MediaStreams[i]
		}
	}
	if sub == nil || sub.DeliveryMethod != "External" || !strings.Contains(sub.DeliveryUrl, "/Subtitles/2/0/Stream.vtt") || !sub.IsExternal {
		t.Fatalf("legenda: %+v", sub)
	}

	// perfil só mp4/h264: o webm precisa de HLS
	profile := map[string]any{"DirectPlayProfiles": []map[string]string{{"Type": "Video", "Container": "mp4", "VideoCodec": "h264", "AudioCodec": "aac"}},
		"TranscodingProfiles": []map[string]string{{"Type": "Video", "Container": "ts", "Protocol": "hls"}}}
	e.call("POST", "/Items/"+e.ids.MovieBig+"/PlaybackInfo?StartTimeTicks=10000000", map[string]any{"DeviceProfile": profile, "MaxStreamingBitrate": 3_000_000}, &pi)
	src = pi.MediaSources[0]
	if e.k.Config.FFmpeg == "" {
		eq(t, "sem ffmpeg vai direto", src.SupportsDirectPlay, true)
	} else {
		eq(t, "transcodifica", src.SupportsDirectPlay, false)
		if !strings.HasPrefix(src.TranscodingUrl, "/Videos/"+e.ids.MovieBig+"/master.m3u8?") ||
			!strings.Contains(src.TranscodingUrl, "PlaySessionId="+pi.PlaySessionId) ||
			!strings.Contains(src.TranscodingUrl, "StartTimeTicks=10000000") {
			t.Fatal(src.TranscodingUrl)
		}
		eq(t, "subprotocol", src.TranscodingSubProtocol, "hls")
	}
	// mp4 h264/aac com o mesmo perfil: direto
	e.call("POST", "/Items/"+e.ids.MoviePixel+"/PlaybackInfo", map[string]any{"DeviceProfile": profile}, &pi)
	eq(t, "mp4 direto", pi.MediaSources[0].SupportsDirectPlay, true)
	// bitrate baixo demais: não vai direto
	e.call("POST", "/Items/"+e.ids.MoviePixel+"/PlaybackInfo", map[string]any{"DeviceProfile": profile, "MaxStreamingBitrate": 1000}, &pi)
	if e.k.Config.FFmpeg != "" {
		eq(t, "bitrate", pi.MediaSources[0].SupportsDirectPlay, false)
	}
	// áudio
	e.call("POST", "/Items/"+e.ids.TrackBop+"/PlaybackInfo", map[string]any{"DeviceProfile": map[string]any{
		"DirectPlayProfiles":  []map[string]string{{"Type": "Audio", "Container": "mp3"}},
		"TranscodingProfiles": []map[string]string{{"Type": "Audio", "Container": "mp3", "AudioCodec": "mp3", "Protocol": "http"}}}}, &pi)
	src = pi.MediaSources[0]
	if e.k.Config.FFmpeg != "" && (src.SupportsDirectPlay || !strings.HasPrefix(src.TranscodingUrl, "/Audio/"+e.ids.TrackBop+"/stream.mp3?")) {
		t.Fatalf("%+v", src)
	}
	eq(t, "pasta não toca", e.call("GET", "/Items/"+e.ids.Series+"/PlaybackInfo", nil, nil), 404)
}

func TestStreamRange(t *testing.T) {
	e := setup(t)
	full, _ := os.ReadFile(e.k.Root + "/Movies/Curta Pixel (2019).mp4")
	resp, raw := e.req("GET", "/Videos/"+e.ids.MoviePixel+"/stream?static=true&mediaSourceId="+e.ids.MoviePixel, nil, "Range", "bytes=10-109")
	eq(t, "206", resp.StatusCode, 206)
	eq(t, "tamanho", len(raw), 100)
	if !bytes.Equal(raw, full[10:110]) {
		t.Fatal("bytes diferentes")
	}
	// api_key na query, sem header (players externos)
	resp, raw = e.req("GET", "/Videos/"+e.ids.MoviePixel+"/stream.mp4?Static=true&api_key="+e.token, nil, "Authorization", "")
	eq(t, "api_key", resp.StatusCode, 200)
	eq(t, "inteiro", len(raw), len(full))
	resp, _ = e.req("HEAD", "/Videos/"+e.ids.MoviePixel+"/stream?static=true", nil)
	eq(t, "head", resp.StatusCode, 200)
	eq(t, "sem login", e.call("GET", "/Videos/"+e.ids.MoviePixel+"/stream?static=true", nil, nil, "Authorization", ""), 401)
	resp, raw = e.req("GET", "/Audio/"+e.ids.TrackBip+"/stream?static=true", nil)
	eq(t, "áudio", resp.StatusCode, 200)
	if len(raw) == 0 {
		t.Fatal("áudio vazio")
	}
	resp, raw = e.req("GET", "/Audio/"+e.ids.TrackBip+"/universal?Container=mp3,flac&api_key="+e.token, nil, "Authorization", "")
	eq(t, "universal direto", resp.StatusCode, 200)
	if len(raw) == 0 {
		t.Fatal("universal vazio")
	}
	if e.k.Config.FFmpeg != "" {
		resp, raw = e.req("GET", "/Audio/"+e.ids.TrackBop+"/universal?Container=mp3&TranscodingContainer=mp3&AudioCodec=mp3", nil)
		eq(t, "universal transcodifica", resp.StatusCode, 200)
		eq(t, "mime", resp.Header.Get("Content-Type"), "audio/mpeg")
		if len(raw) < 100 {
			t.Fatalf("mp3 curto: %d", len(raw))
		}
	}
	resp, raw = e.req("GET", "/Items/"+e.ids.MoviePixel+"/Download", nil)
	eq(t, "download", len(raw), len(full))
}

func TestImages(t *testing.T) {
	e := setup(t)
	check := func(id, typ string, want int) {
		t.Helper()
		resp, raw := e.req("GET", "/Items/"+id+"/Images/"+typ+"?maxWidth=40&quality=80", nil, "Authorization", "")
		eq(t, id+" "+typ, resp.StatusCode, want)
		if want == 200 && (len(raw) == 0 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/")) {
			t.Fatalf("%s %s: %q", id, typ, resp.Header.Get("Content-Type"))
		}
	}
	check(e.ids.MovieBig, "Primary", 200)
	check(e.ids.MovieBig, "primary/0", 200)
	check(e.ids.Ep1, "Primary", 200)     // cai na série
	check(e.ids.Season2, "Primary", 200) // idem
	check(e.ids.TrackBop, "Primary", 200)
	check(e.ids.Album, "Primary", 200)
	check(e.ids.Artist, "Primary", 200) // capa de um álbum
	check(e.ids.MovieBig, "Logo", 404)
	check(e.ids.TrackSolta, "Primary", 404)
	check(viewID("movies"), "Primary", 200)
	if e.k.Config.FFmpeg != "" {
		check(e.ids.MoviePixel, "Primary", 200) // quadro do vídeo
		check(e.ids.MoviePixel, "Backdrop", 200)
	}
	resp, _ := e.req("HEAD", "/Items/"+e.ids.MovieBig+"/Images/Primary", nil)
	eq(t, "head", resp.StatusCode, 200)
	resp, _ = e.req("GET", "/Artists/Banda%20Teste/Images/Primary/0", nil, "Authorization", "")
	eq(t, "artista por nome", resp.StatusCode, 200)
}

func TestProgressReporting(t *testing.T) {
	e := setup(t)
	id := e.ids.Ep1
	report := func(path string, ticks int64) {
		t.Helper()
		eq(t, path, e.call("POST", path, map[string]any{"ItemId": id, "PositionTicks": ticks, "PlaySessionId": "ps1",
			"MediaSourceId": id, "IsPaused": false, "CanSeek": true, "PlayMethod": "DirectPlay"}, nil), 204)
	}
	report("/Sessions/Playing", 0)
	report("/Sessions/Playing/Progress", 10_000_000)
	var it dtoT
	e.get("/Items/"+id, &it)
	eq(t, "posição", it.UserData.PlaybackPositionTicks, int64(10_000_000))
	eq(t, "resume", joined(names(e.list("/Items/Resume").Items)), "O Começo")
	// sessão mostra o que toca
	var sess []map[string]any
	e.get("/Sessions", &sess)
	if len(sess) != 1 || sess[0]["NowPlayingItem"] == nil {
		t.Fatalf("%+v", sess)
	}
	report("/Sessions/Playing/Stopped", 19_500_000)
	e.get("/Items/"+id, &it)
	eq(t, "visto", it.UserData.Played, true)
	eq(t, "zerou", it.UserData.PlaybackPositionTicks, int64(0))
	eq(t, "contagem", it.UserData.PlayCount, 1)
	eq(t, "resume vazio", e.list("/Items/Resume").TotalRecordCount, 0)

	// legado
	eq(t, "legado start", e.call("POST", "/Users/"+e.userID+"/PlayingItems/"+e.ids.Ep2+"?PositionTicks=10000000", nil, nil), 204)
	e.get("/Items/"+e.ids.Ep2, &it)
	eq(t, "legado pos", it.UserData.PlaybackPositionTicks, int64(10_000_000))
	eq(t, "legado stop", e.call("DELETE", "/PlayingItems/"+e.ids.Ep2+"?PositionTicks=19000000", nil, nil), 204)
	e.get("/Items/"+e.ids.Ep2, &it)
	eq(t, "legado visto", it.UserData.Played, true)
	eq(t, "ping", e.call("POST", "/Sessions/Playing/Ping?playSessionId=ps1", nil, nil), 204)
}

func TestUserDataEndpoints(t *testing.T) {
	e := setup(t)
	type ud struct {
		IsFavorite, Played bool
		PlayCount          int
		Key, ItemId        string
		UnplayedItemCount  *int
	}
	var d ud
	for _, c := range []struct{ method, path string }{
		{"POST", "/Users/" + e.userID + "/FavoriteItems/" + e.ids.MovieBig},
		{"DELETE", "/Users/" + e.userID + "/FavoriteItems/" + e.ids.MovieBig},
		{"POST", "/UserFavoriteItems/" + e.ids.MovieBig},
	} {
		eq(t, c.path, e.call(c.method, c.path, nil, &d), 200)
		eq(t, c.method+" fav", d.IsFavorite, c.method == "POST")
		eq(t, "item id", d.ItemId, e.ids.MovieBig)
	}
	eq(t, "played", e.call("POST", "/UserPlayedItems/"+e.ids.MovieBig+"?DatePlayed=2024-01-02T03:04:05Z", nil, &d), 200)
	eq(t, "played", d.Played, true)
	eq(t, "unplayed", e.call("DELETE", "/Users/"+e.userID+"/PlayedItems/"+e.ids.MovieBig, nil, &d), 200)
	eq(t, "unplayed", d.Played, false)

	// série inteira
	eq(t, "série", e.call("POST", "/Users/"+e.userID+"/PlayedItems/"+e.ids.Series, nil, &d), 200)
	eq(t, "série vista", d.Played, true)
	eq(t, "nada falta", *d.UnplayedItemCount, 0)
	var ep dtoT
	e.get("/Items/"+e.ids.Ep3, &ep)
	eq(t, "episódio visto", ep.UserData.Played, true)
	e.call("DELETE", "/UserPlayedItems/"+e.ids.Season1, nil, &d)
	var sr dtoT
	e.get("/Items/"+e.ids.Series, &sr)
	eq(t, "falta 2", *sr.UserData.UnplayedItemCount, 2)
	eq(t, "série não vista", sr.UserData.Played, false)

	// UserData direto
	eq(t, "get", e.call("GET", "/Items/"+e.ids.MoviePixel+"/UserData", nil, &d), 200)
	eq(t, "post", e.call("POST", "/UserItems/"+e.ids.MoviePixel+"/UserData", map[string]any{"PlaybackPositionTicks": 5_000_000, "IsFavorite": true}, &d), 200)
	eq(t, "fav via userdata", d.IsFavorite, true)
	var it dtoT
	e.get("/Items/"+e.ids.MoviePixel, &it)
	eq(t, "pos via userdata", it.UserData.PlaybackPositionTicks, int64(5_000_000))
	eq(t, "404", e.call("POST", "/UserFavoriteItems/ffffffffffffffffffffffffffffffff", nil, nil), 404)
}

func TestSubtitles(t *testing.T) {
	e := setup(t)
	id := e.ids.MovieBig
	resp, raw := e.req("GET", "/Videos/"+id+"/"+id+"/Subtitles/2/0/Stream.vtt", nil)
	eq(t, "vtt", resp.StatusCode, 200)
	if !strings.HasPrefix(string(raw), "WEBVTT") || !strings.Contains(string(raw), " --> ") || strings.Contains(string(raw), ",0") {
		t.Fatalf("%q", raw)
	}
	resp, raw = e.req("GET", "/Videos/"+id+"/"+id+"/Subtitles/2/Stream.srt?api_key="+e.token, nil, "Authorization", "")
	eq(t, "srt", resp.StatusCode, 200)
	if !strings.Contains(string(raw), "-->") {
		t.Fatalf("%q", raw)
	}
	eq(t, "faixa de áudio não é legenda", e.call("GET", "/Videos/"+id+"/"+id+"/Subtitles/1/0/Stream.vtt", nil, nil), 404)
}

func TestSocketKeepAlive(t *testing.T) {
	e := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	u := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/socket?api_key=" + e.token + "&deviceId=aparelho-1"
	conn, _, err := websocket.Dial(ctx, u, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	read := func() wsMsg {
		_, raw, err := conn.Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var m wsMsg
		json.Unmarshal(raw, &m)
		return m
	}
	m := read()
	eq(t, "force", m.MessageType, "ForceKeepAlive")
	eq(t, "intervalo", m.Data, any(float64(keepAlive)))
	conn.Write(ctx, websocket.MessageText, []byte(`{"MessageType":"SessionsStart","Data":"0,1500"}`))
	conn.Write(ctx, websocket.MessageText, []byte(`{"MessageType":"KeepAlive"}`))
	eq(t, "keepalive", read().MessageType, "KeepAlive")

	if _, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(e.srv.URL, "http")+"/socket?api_key=errado", nil); err == nil {
		t.Fatal("aceitou sem token")
	}
}
