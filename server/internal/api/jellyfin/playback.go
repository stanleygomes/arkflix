package jellyfin

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// streamDto é o MediaStream.
type streamDto struct {
	Codec                  string  `json:",omitempty"`
	Language               string  `json:",omitempty"`
	Title                  string  `json:",omitempty"`
	VideoRange             string  `json:"VideoRange"`
	VideoRangeType         string  `json:"VideoRangeType"`
	AudioSpatialFormat     string  `json:"AudioSpatialFormat"`
	DisplayTitle           string  `json:",omitempty"`
	IsInterlaced           bool    `json:"IsInterlaced"`
	IsAVC                  *bool   `json:",omitempty"`
	ChannelLayout          string  `json:",omitempty"`
	BitRate                int     `json:",omitempty"`
	BitDepth               int     `json:",omitempty"`
	IsDefault              bool    `json:"IsDefault"`
	IsForced               bool    `json:"IsForced"`
	IsHearingImpaired      bool    `json:"IsHearingImpaired"`
	IsOriginal             bool    `json:"IsOriginal"`
	Height                 int     `json:",omitempty"`
	Width                  int     `json:",omitempty"`
	AverageFrameRate       float64 `json:",omitempty"`
	RealFrameRate          float64 `json:",omitempty"`
	Profile                string  `json:",omitempty"`
	Type                   string  `json:"Type"`
	AspectRatio            string  `json:",omitempty"`
	Index                  int     `json:"Index"`
	IsExternal             bool    `json:"IsExternal"`
	DeliveryMethod         string  `json:",omitempty"`
	DeliveryUrl            string  `json:",omitempty"`
	IsExternalUrl          *bool   `json:",omitempty"`
	IsTextSubtitleStream   bool    `json:"IsTextSubtitleStream"`
	SupportsExternalStream bool    `json:"SupportsExternalStream"`
	PixelFormat            string  `json:",omitempty"`
	Level                  float64 `json:",omitempty"`
	Channels               int     `json:",omitempty"`
	SampleRate             int     `json:",omitempty"`
}

// sourceDto é o MediaSourceInfo.
type sourceDto struct {
	Protocol                            string
	Id                                  string
	Path                                string
	Type                                string
	Container                           string
	Size                                int64  `json:",omitempty"`
	Name                                string `json:",omitempty"`
	IsRemote                            bool
	ETag                                string `json:",omitempty"`
	RunTimeTicks                        int64  `json:",omitempty"`
	ReadAtNativeFramerate               bool
	IgnoreDts                           bool
	IgnoreIndex                         bool
	GenPtsInput                         bool
	SupportsTranscoding                 bool
	SupportsDirectStream                bool
	SupportsDirectPlay                  bool
	IsInfiniteStream                    bool
	UseMostCompatibleTranscodingProfile bool
	RequiresOpening                     bool
	RequiresClosing                     bool
	RequiresLooping                     bool
	SupportsProbing                     bool
	VideoType                           string `json:",omitempty"`
	MediaStreams                        []streamDto
	MediaAttachments                    []any
	Formats                             []string
	Bitrate                             int `json:",omitempty"`
	RequiredHttpHeaders                 map[string]string
	DirectStreamUrl                     string `json:",omitempty"`
	TranscodingUrl                      string `json:",omitempty"`
	TranscodingSubProtocol              string
	TranscodingContainer                string `json:",omitempty"`
	DefaultAudioStreamIndex             *int   `json:",omitempty"`
	DefaultSubtitleStreamIndex          *int   `json:",omitempty"`
	HasSegments                         bool
}

// deviceProfile: só o que a decisão usa (o resto do perfil é ignorado).
type deviceProfile struct {
	MaxStreamingBitrate any
	DirectPlayProfiles  []struct{ Type, Container, VideoCodec, AudioCodec string }
	TranscodingProfiles []struct{ Type, Container, VideoCodec, AudioCodec, Protocol string }
	SubtitleProfiles    []struct{ Format, Method string }
}

// playOpts é o pedido de reprodução (PlaybackInfoDto + query).
type playOpts struct {
	profile             *deviceProfile
	maxBitrate          int
	audioIdx, subIdx    *int
	start               int64
	direct, transcoding bool
	playSession         string
	maxWidth, maxHeight int
}

var langNames = map[string]string{
	"por": "Português", "pt": "Português", "eng": "Inglês", "en": "Inglês", "spa": "Espanhol", "es": "Espanhol",
	"fre": "Francês", "fra": "Francês", "fr": "Francês", "ger": "Alemão", "deu": "Alemão", "de": "Alemão",
	"ita": "Italiano", "it": "Italiano", "jpn": "Japonês", "ja": "Japonês", "kor": "Coreano", "ko": "Coreano",
	"chi": "Chinês", "zho": "Chinês", "zh": "Chinês", "rus": "Russo", "ru": "Russo",
}

var imageSubs = []string{"hdmv_pgs_subtitle", "pgssub", "pgs", "dvd_subtitle", "dvdsub", "dvb_subtitle", "dvbsub", "xsub", "vobsub"}

func isTextSub(codec string) bool { return !slices.Contains(imageSubs, strings.ToLower(codec)) }

// subFormat: a extensão "natural" de uma legenda de texto.
func subFormat(codec string) string {
	switch strings.ToLower(codec) {
	case "subrip", "srt":
		return "srt"
	case "ass":
		return "ass"
	case "ssa":
		return "ssa"
	}
	return "vtt"
}

func layoutName(ch int, layout string) string {
	switch ch {
	case 1:
		return "Mono"
	case 2:
		return "Stereo"
	case 6:
		return "5.1"
	case 8:
		return "7.1"
	}
	return layout
}

func displayTitle(s store.Stream) string {
	lang := langNames[strings.ToLower(s.Language)]
	var parts []string
	switch s.Type {
	case "video":
		if s.Height > 0 {
			parts = append(parts, resLabel(s.Height))
		}
		parts = append(parts, strings.ToUpper(s.Codec))
		if s.Range != "" {
			parts = append(parts, s.Range)
		}
	case "audio":
		parts = append(parts, lang, strings.ToUpper(s.Codec), layoutName(s.Channels, s.Layout))
		if s.Default {
			parts = append(parts, "Padrão")
		}
	case "subtitle":
		parts = append(parts, lang, s.Title, strings.ToUpper(subFormat(s.Codec)))
		if s.Forced {
			parts = append(parts, "Forçada")
		}
		if s.External != "" {
			parts = append(parts, "Externa")
		}
	}
	parts = filter(parts, func(p string) bool { return p != "" })
	return strings.Join(parts, " - ")
}

func resLabel(h int) string {
	switch {
	case h >= 2000:
		return "4K"
	case h >= 1000:
		return "1080p"
	case h >= 700:
		return "720p"
	case h >= 560:
		return "576p"
	case h >= 470:
		return "480p"
	}
	return strconv.Itoa(h) + "p"
}

func defaultAudio(it *store.Item) int {
	idx := -1
	for _, s := range it.Streams {
		if s.Type == "audio" {
			if s.Default {
				return s.Index
			}
			if idx < 0 {
				idx = s.Index
			}
		}
	}
	return idx
}

func (c *rc) token() string { return c.tok.Token }

// subURL é o DeliveryUrl de uma legenda.
func (c *rc) subURL(it *store.Item, idx int, format string) string {
	return fmt.Sprintf("/Videos/%s/%s/Subtitles/%d/0/Stream.%s?api_key=%s", it.ID, it.ID, idx, format, url.QueryEscape(c.token()))
}

// subProfileFormat escolhe o formato externo que o cliente aceita.
func subProfileFormat(o *playOpts, codec string) string {
	natural := subFormat(codec)
	if o == nil || o.profile == nil {
		return "vtt"
	}
	var allowed []string
	for _, p := range o.profile.SubtitleProfiles {
		if strings.EqualFold(p.Method, "External") {
			allowed = append(allowed, strings.ToLower(p.Format))
		}
	}
	switch {
	case len(allowed) == 0, slices.Contains(allowed, natural):
		if len(allowed) == 0 {
			return "vtt"
		}
		return natural
	case slices.Contains(allowed, "vtt"):
		return "vtt"
	case slices.Contains(allowed, "srt"):
		return "srt"
	}
	return "vtt"
}

func (c *rc) streams(it *store.Item, o *playOpts) []streamDto {
	out := []streamDto{}
	for _, s := range it.Streams {
		d := streamDto{
			Codec: s.Codec, Language: s.Language, Title: s.Title, DisplayTitle: displayTitle(s),
			VideoRange: "Unknown", VideoRangeType: "Unknown", AudioSpatialFormat: "None",
			IsDefault: s.Default, IsForced: s.Forced, Index: s.Index, IsExternal: s.External != "",
			Profile: s.Profile, BitRate: s.Bitrate,
		}
		switch s.Type {
		case "video":
			d.Type = "Video"
			d.Width, d.Height, d.AverageFrameRate, d.RealFrameRate = s.Width, s.Height, s.FPS, s.FPS
			d.PixelFormat, d.Level = s.PixFmt, float64(s.Level)
			d.VideoRange, d.VideoRangeType = "SDR", "SDR"
			if s.Range != "" && s.Range != "SDR" {
				d.VideoRange, d.VideoRangeType = "HDR", s.Range
			}
			avc := s.Codec == "h264"
			d.IsAVC = &avc
			if s.Height > 0 {
				d.AspectRatio = aspect(s.Width, s.Height)
			}
		case "audio":
			d.Type = "Audio"
			d.Channels, d.ChannelLayout, d.SampleRate = s.Channels, s.Layout, s.Rate
		case "subtitle":
			d.Type = "Subtitle"
			d.IsTextSubtitleStream = isTextSub(s.Codec)
			d.SupportsExternalStream = d.IsTextSubtitleStream
			switch {
			case d.IsTextSubtitleStream:
				f := subProfileFormat(o, s.Codec)
				d.DeliveryMethod, d.DeliveryUrl = "External", c.subURL(it, s.Index, f)
				no := false
				d.IsExternalUrl = &no
			default:
				d.DeliveryMethod = "Embed"
			}
		default:
			continue
		}
		out = append(out, d)
	}
	return out
}

func aspect(w, h int) string {
	a, b := w, h
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", w/a, h/a)
}

// aliases de contêiner e codec que os perfis usam.
func normName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "matroska":
		return "mkv"
	case "mpegts":
		return "ts"
	case "avc", "h.264":
		return "h264"
	case "h265", "h.265":
		return "hevc"
	case "m4v", "mov":
		return "mp4"
	}
	return s
}

func listHas(list, v string) bool {
	if strings.TrimSpace(list) == "" {
		return true
	}
	v = normName(v)
	for p := range strings.SplitSeq(list, ",") {
		if normName(p) == v {
			return true
		}
	}
	return false
}

func firstStream(it *store.Item, typ string) *store.Stream {
	for i := range it.Streams {
		if it.Streams[i].Type == typ && it.Streams[i].External == "" {
			return &it.Streams[i]
		}
	}
	return nil
}

func streamByIndex(it *store.Item, idx int) *store.Stream {
	for i := range it.Streams {
		if it.Streams[i].Index == idx {
			return &it.Streams[i]
		}
	}
	return nil
}

// canDirect decide se o cliente toca o arquivo como está.
func canDirect(it *store.Item, o *playOpts) bool {
	if o == nil {
		return true
	}
	if !o.direct {
		return false
	}
	if o.maxBitrate > 0 && it.Bitrate > o.maxBitrate {
		return false
	}
	audio := streamByIndex(it, defaultAudio(it))
	if o.audioIdx != nil && *o.audioIdx >= 0 {
		if *o.audioIdx != defaultAudio(it) {
			return false
		}
		audio = streamByIndex(it, *o.audioIdx)
	}
	if o.profile == nil {
		return true
	}
	typ := "Video"
	if it.Kind == store.KindTrack {
		typ = "Audio"
	}
	video := firstStream(it, "video")
	for _, p := range o.profile.DirectPlayProfiles {
		if !strings.EqualFold(p.Type, typ) || !listHas(p.Container, it.Container) {
			continue
		}
		if typ == "Video" && video != nil && !listHas(p.VideoCodec, video.Codec) {
			continue
		}
		if audio != nil && !listHas(p.AudioCodec, audio.Codec) {
			continue
		}
		return true
	}
	return false
}

func (c *rc) staticURL(it *store.Item, o *playOpts) string {
	q := url.Values{"static": {"true"}, "mediaSourceId": {it.ID}, "api_key": {c.token()}}
	if o != nil && o.playSession != "" {
		q.Set("PlaySessionId", o.playSession)
	}
	if it.Kind == store.KindTrack {
		return "/Audio/" + it.ID + "/stream?" + q.Encode()
	}
	return "/Videos/" + it.ID + "/stream?" + q.Encode()
}

// audioFormat escolhe o formato da transcodificação de áudio.
func audioFormat(container, codec string) string {
	for _, v := range []string{codec, container} {
		switch normName(v) {
		case "mp3":
			return "mp3"
		case "aac", "m4a", "mp4", "ts", "m4b":
			return "aac"
		case "opus", "ogg", "webm", "oga", "vorbis":
			return "opus"
		case "flac":
			return "flac"
		}
	}
	return "mp3"
}

// source monta o MediaSource (o == nil: sem perfil, direto).
func (c *rc) source(it *store.Item, o *playOpts) sourceDto {
	s := sourceDto{
		Protocol: "File", Id: it.ID, Type: "Default", Container: it.Container, Size: it.Size, Name: it.Name,
		ETag: hash(it.ID, strconv.FormatInt(it.MTime, 10))[:16], RunTimeTicks: ticks(it.Duration),
		SupportsProbing: true, MediaStreams: c.streams(it, o), MediaAttachments: []any{}, Formats: []string{},
		Bitrate: it.Bitrate, RequiredHttpHeaders: map[string]string{}, TranscodingSubProtocol: "http",
		SupportsTranscoding: c.s.d.Config.FFmpeg != "",
	}
	if it.Kind != store.KindTrack {
		s.VideoType = "VideoFile"
	}
	if a := defaultAudio(it); a >= 0 {
		s.DefaultAudioStreamIndex = intp(a)
	}
	sub := -1
	for _, st := range it.Streams {
		if st.Type == "subtitle" && (st.Forced || st.Default) {
			sub = st.Index
			break
		}
	}
	s.DefaultSubtitleStreamIndex = intp(sub)
	if o != nil && o.audioIdx != nil && *o.audioIdx >= 0 {
		s.DefaultAudioStreamIndex = intp(*o.audioIdx)
	}
	if o != nil && o.subIdx != nil {
		s.DefaultSubtitleStreamIndex = intp(*o.subIdx)
	}
	direct := canDirect(it, o)
	if !s.SupportsTranscoding {
		direct = true // sem ffmpeg, o arquivo é o que tem
	}
	s.SupportsDirectPlay, s.SupportsDirectStream = direct, direct
	s.Path = c.staticURL(it, o)
	if direct {
		s.DirectStreamUrl = s.Path
	}
	wantURL := !direct || (o != nil && o.audioIdx != nil && *o.audioIdx >= 0)
	if o != nil && o.transcoding && s.SupportsTranscoding && wantURL {
		if it.Kind == store.KindTrack {
			s.TranscodingUrl, s.TranscodingContainer = c.audioTranscodeURL(it, o)
		} else {
			s.TranscodingUrl, s.TranscodingContainer, s.TranscodingSubProtocol = c.hlsURL(it, o), "ts", "hls"
		}
	}
	return s
}

func (c *rc) audioTranscodeURL(it *store.Item, o *playOpts) (string, string) {
	container, codec := "", ""
	if o.profile != nil {
		for _, p := range o.profile.TranscodingProfiles {
			if strings.EqualFold(p.Type, "Audio") {
				container, codec = p.Container, p.AudioCodec
				break
			}
		}
	}
	f := audioFormat(container, strings.Split(codec, ",")[0])
	q := url.Values{"api_key": {c.token()}, "PlaySessionId": {o.playSession}, "AudioCodec": {f}, "MediaSourceId": {it.ID}}
	if o.maxBitrate > 0 {
		q.Set("MaxStreamingBitrate", strconv.Itoa(o.maxBitrate))
	}
	if o.start > 0 {
		q.Set("StartTimeTicks", strconv.FormatInt(o.start, 10))
	}
	return "/Audio/" + it.ID + "/stream." + f + "?" + q.Encode(), f
}

func (c *rc) hlsURL(it *store.Item, o *playOpts) string {
	q := url.Values{"api_key": {c.token()}, "PlaySessionId": {o.playSession}, "MediaSourceId": {it.ID},
		"VideoCodec": {"h264"}, "AudioCodec": {"aac"}, "SegmentContainer": {"ts"}, "TranscodingMaxAudioChannels": {"2"}}
	if c.ah.DeviceID != "" {
		q.Set("DeviceId", c.ah.DeviceID)
	}
	if o.audioIdx != nil && *o.audioIdx >= 0 {
		q.Set("AudioStreamIndex", strconv.Itoa(*o.audioIdx))
	}
	if o.subIdx != nil && *o.subIdx >= 0 {
		q.Set("SubtitleStreamIndex", strconv.Itoa(*o.subIdx))
	}
	if o.maxBitrate > 0 {
		q.Set("VideoBitrate", strconv.Itoa(max(o.maxBitrate-192_000, 500_000)))
	}
	if o.maxWidth > 0 {
		q.Set("MaxWidth", strconv.Itoa(o.maxWidth))
	}
	if o.maxHeight > 0 {
		q.Set("MaxHeight", strconv.Itoa(o.maxHeight))
	}
	if o.start > 0 {
		q.Set("StartTimeTicks", strconv.FormatInt(o.start, 10))
	}
	return "/Videos/" + it.ID + "/master.m3u8?" + q.Encode()
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}

// GET/POST /Items/{id}/PlaybackInfo
func (s *Server) playbackInfo(c *rc) {
	it := c.lookup(c.pid("id"))
	if it == nil || (it.Kind != store.KindMovie && it.Kind != store.KindEpisode && it.Kind != store.KindTrack) {
		c.notFound()
		return
	}
	var body struct {
		MaxStreamingBitrate                                     any
		StartTimeTicks                                          *int64
		AudioStreamIndex, SubtitleStreamIndex                   *int
		MaxWidth, MaxHeight                                     *int
		DeviceProfile                                           *deviceProfile
		EnableDirectPlay, EnableDirectStream, EnableTranscoding *bool
	}
	if c.r.Method == http.MethodPost {
		if err := c.body(&body); err != nil {
			c.fail(http.StatusBadRequest, "corpo inválido: "+err.Error())
			return
		}
	}
	o := &playOpts{profile: body.DeviceProfile, direct: true, transcoding: true, playSession: auth.RandomHex(16)}
	o.maxBitrate = toInt(body.MaxStreamingBitrate)
	if o.maxBitrate == 0 {
		o.maxBitrate = c.qi("MaxStreamingBitrate", 0)
	}
	if o.maxBitrate == 0 && o.profile != nil {
		o.maxBitrate = toInt(o.profile.MaxStreamingBitrate)
	}
	pick := func(b *int, name string) *int {
		if b != nil {
			return b
		}
		if c.qs(name) != "" {
			return intp(c.qi(name, -1))
		}
		return nil
	}
	o.audioIdx, o.subIdx = pick(body.AudioStreamIndex, "AudioStreamIndex"), pick(body.SubtitleStreamIndex, "SubtitleStreamIndex")
	if w := pick(body.MaxWidth, "MaxWidth"); w != nil {
		o.maxWidth = *w
	}
	if h := pick(body.MaxHeight, "MaxHeight"); h != nil {
		o.maxHeight = *h
	}
	o.start = c.qi64("StartTimeTicks")
	if body.StartTimeTicks != nil {
		o.start = *body.StartTimeTicks
	}
	flag := func(b *bool, name string) bool {
		if b != nil {
			return *b
		}
		if v, ok := c.qb(name); ok {
			return v
		}
		return true
	}
	o.direct = flag(body.EnableDirectPlay, "EnableDirectPlay") || flag(body.EnableDirectStream, "EnableDirectStream")
	o.transcoding = flag(body.EnableTranscoding, "EnableTranscoding")
	c.json(map[string]any{"MediaSources": []sourceDto{c.source(it, o)}, "PlaySessionId": o.playSession})
}

func (c *rc) playable(kinds ...string) *store.Item {
	it := c.lookup(c.pid("id"))
	if it == nil || !slices.Contains(kinds, it.Kind) {
		c.notFound()
		return nil
	}
	return it
}

// GET/HEAD /Videos/{id}/stream[.{container}]
func (s *Server) videoStream(c *rc) {
	it := c.playable(store.KindMovie, store.KindEpisode)
	if it == nil {
		return
	}
	static, _ := c.qb("static")
	if static || s.d.Config.FFmpeg == "" || !c.wantsVideoTranscode(it) {
		s.d.Media.ServeFile(c.w, c.r, it.Path)
		return
	}
	s.progressive(c, it)
}

func (c *rc) wantsVideoTranscode(it *store.Item) bool {
	if ext := c.p["container"]; ext != "" && normName(ext) != normName(it.Container) {
		return true
	}
	if c.qs("AudioStreamIndex") != "" && c.qi("AudioStreamIndex", -1) != defaultAudio(it) {
		return true
	}
	if v := firstStream(it, "video"); v != nil && c.qs("VideoCodec") != "" && !listHas(c.qs("VideoCodec"), v.Codec) {
		return true
	}
	if a := firstStream(it, "audio"); a != nil && c.qs("AudioCodec") != "" && !listHas(c.qs("AudioCodec"), a.Codec) {
		return true
	}
	return false
}

// GET/HEAD /Audio/{id}/stream[.{container}]
func (s *Server) audioStream(c *rc) {
	it := c.playable(store.KindTrack)
	if it == nil {
		return
	}
	static, _ := c.qb("static")
	ext := c.p["container"]
	codec := c.qs("AudioCodec")
	if static || s.d.Config.FFmpeg == "" || ((ext == "" || normName(ext) == normName(it.Container)) &&
		(codec == "" || listHas(codec, it.Container))) {
		s.d.Media.ServeFile(c.w, c.r, it.Path)
		return
	}
	s.transcodeAudio(c, it, audioFormat(ext, strings.Split(codec, ",")[0]))
}

// GET/HEAD /Audio/{id}/universal: direto se o contêiner serve, senão
// transcodifica (sempre por HTTP; HLS de áudio não tem).
func (s *Server) audioUniversal(c *rc) {
	it := c.playable(store.KindTrack)
	if it == nil {
		return
	}
	ok := len(c.ql("Container", ",")) == 0
	for _, entry := range c.ql("Container", ",") {
		cont, codec, _ := strings.Cut(entry, "|")
		if normName(cont) == normName(it.Container) && (codec == "" || listHas(codec, it.Container)) {
			ok = true
		}
	}
	maxBr := c.qi("MaxStreamingBitrate", 0)
	if maxBr > 0 && it.Bitrate > maxBr {
		ok = false
	}
	if ok || s.d.Config.FFmpeg == "" {
		s.d.Media.ServeFile(c.w, c.r, it.Path)
		return
	}
	s.transcodeAudio(c, it, audioFormat(c.qs("TranscodingContainer"), c.qs("AudioCodec")))
}

func (s *Server) transcodeAudio(c *rc, it *store.Item, format string) {
	br := c.qi("AudioBitRate", 0)
	if br == 0 {
		br = c.qi("MaxStreamingBitrate", 0)
	}
	kbps := 192
	if br > 0 {
		kbps = min(max(br/1000, 64), 320)
	}
	c.w.Header().Set("Content-Type", media.ContentType(format))
	c.w.Header().Set("Cache-Control", "no-cache")
	if c.r.Method == http.MethodHead {
		return
	}
	s.d.Media.TranscodeAudio(c.ctx(), c.w, it.Path, media.AudioOpts{Format: format, Bitrate: kbps,
		Offset: float64(c.qi64("StartTimeTicks")) / ticksPerSecond})
}

// GET /Items/{id}/Download e /Items/{id}/File: o arquivo original.
func (s *Server) download(c *rc) {
	it := c.playable(store.KindMovie, store.KindEpisode, store.KindTrack)
	if it == nil {
		return
	}
	s.d.Media.ServeFile(c.w, c.r, it.Path)
}
