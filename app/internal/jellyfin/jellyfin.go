// Package jellyfin fala com um servidor Jellyfin (https://jellyfin.org): login,
// filmes, séries, episódios, o que continuar assistindo e o progresso de
// reprodução. A senha só é usada no login: dela sai um token de acesso, que é
// o que fica guardado (e que o Jellyfin invalida no Logout).
package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client é o nome do app no painel de dispositivos do Jellyfin.
const Client = "arkflix"

var (
	ErrAuth = errors.New("o Jellyfin recusou o acesso: vincule a conta de novo no perfil")
	ErrHost = errors.New("endereço do servidor inválido")
)

// Ticks: o Jellyfin conta o tempo em unidades de 100 ns.
const ticksPerSecond = 10_000_000

// Seconds converte ticks em segundos.
func Seconds(ticks int64) float64 { return float64(ticks) / ticksPerSecond }

// Ticks converte segundos em ticks.
func Ticks(seconds float64) int64 { return int64(seconds * ticksPerSecond) }

// Account é o que fica guardado depois do login (nunca a senha).
type Account struct {
	Base     string `json:"base"`
	User     string `json:"user"`
	UserID   string `json:"userId"`
	Token    string `json:"token"`
	Server   string `json:"server"`   // nome do servidor
	DeviceID string `json:"deviceId"` // o mesmo em toda chamada (o Jellyfin agrupa por ele)
	Device   string `json:"device"`
	Version  string `json:"version"`
}

// NormalizeBase aceita "servidor.com", "https://servidor.com/" ou um caminho
// (proxy). HTTP puro só na rede de casa (localhost e IPs privados): o token
// não viaja aberto na internet.
func NormalizeBase(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrHost
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", ErrHost
	}
	if u.Scheme == "http" && !homeNetwork(u.Hostname()) {
		return "", errors.New("use https (http só pra servidor na rede de casa)")
	}
	u.RawQuery, u.Fragment = "", ""
	return strings.TrimRight(u.String(), "/"), nil
}

func homeNetwork(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".lan") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

// Authorization é o cabeçalho que o Jellyfin espera (com ou sem token).
func Authorization(acc Account) string {
	h := fmt.Sprintf(`MediaBrowser Client=%q, Device=%q, DeviceId=%q, Version=%q`, Client, acc.Device, acc.DeviceID, acc.Version)
	if acc.Token != "" {
		h += fmt.Sprintf(`, Token=%q`, acc.Token)
	}
	return h
}

// API chama o servidor com a conta.
type API struct {
	Account
	HTTP *http.Client
}

func (c *API) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *API) do(ctx context.Context, method, p string, body any) (*http.Response, error) {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+p, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", Authorization(c.Account))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("servidor fora do ar? %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, ErrAuth
	}
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		resp.Body.Close()
		return nil, fmt.Errorf("servidor respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return resp, nil
}

func (c *API) call(ctx context.Context, method, p string, body, out any) error {
	resp, err := c.do(ctx, method, p, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Login troca usuário e senha por um token de acesso. A senha não sai daqui.
func Login(ctx context.Context, base, user, pass string, acc Account) (Account, error) {
	base, err := NormalizeBase(base)
	if err != nil {
		return Account{}, err
	}
	acc.Base, acc.Token = base, ""
	c := &API{Account: acc}
	var res struct {
		AccessToken string
		ServerId    string
		User        struct {
			Id   string
			Name string
		}
	}
	err = c.call(ctx, http.MethodPost, "/Users/AuthenticateByName", map[string]string{"Username": user, "Pw": pass}, &res)
	if errors.Is(err, ErrAuth) {
		return Account{}, errors.New("usuário ou senha errados")
	}
	if err != nil {
		return Account{}, err
	}
	if res.AccessToken == "" || res.User.Id == "" {
		return Account{}, errors.New("isso não parece um servidor Jellyfin")
	}
	acc.Token, acc.UserID, acc.User = res.AccessToken, res.User.Id, res.User.Name
	c.Account = acc
	var info struct{ ServerName string }
	if c.call(ctx, http.MethodGet, "/System/Info/Public", nil, &info) == nil {
		acc.Server = info.ServerName
	}
	return acc, nil
}

// Logout invalida o token no servidor.
func (c *API) Logout(ctx context.Context) error {
	return c.call(ctx, http.MethodPost, "/Sessions/Logout", nil, nil)
}

// Item é um filme, série, temporada ou episódio.
type Item struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // Movie, Series, Season, Episode
	Name     string   `json:"name"`
	Overview string   `json:"overview"`
	Year     int      `json:"year"`
	Genres   []string `json:"genres"`
	Rating   string   `json:"rating"` // classificação indicativa
	Score    float64  `json:"score"`  // nota da comunidade (0-10)
	Runtime  float64  `json:"runtime"`
	Position float64  `json:"position"` // segundos assistidos
	Played   bool     `json:"played"`
	Favorite bool     `json:"favorite"`
	Unplayed int      `json:"unplayed"` // episódios não vistos (séries)

	SeriesID   string `json:"seriesId"`
	SeriesName string `json:"seriesName"`
	SeasonID   string `json:"seasonId"`
	Season     int    `json:"season"`
	Episode    int    `json:"episode"`

	// Imagens: item dono e tag (vazio = sem a imagem).
	Primary      string `json:"-"`
	PrimaryItem  string `json:"-"`
	Backdrop     string `json:"-"`
	BackdropItem string `json:"-"`
	Logo         string `json:"-"`
	LogoItem     string `json:"-"`
	Thumb        string `json:"-"`
	ThumbItem    string `json:"-"`
}

type apiItem struct {
	Id                      string
	Type                    string
	Name                    string
	Overview                string
	ProductionYear          int
	Genres                  []string
	OfficialRating          string
	CommunityRating         float64
	RunTimeTicks            int64
	SeriesId                string
	SeriesName              string
	SeasonId                string
	ParentIndexNumber       int
	IndexNumber             int
	ImageTags               map[string]string
	BackdropImageTags       []string
	ParentBackdropItemId    string
	ParentBackdropImageTags []string
	ParentLogoItemId        string
	ParentLogoImageTag      string
	ParentThumbItemId       string
	ParentThumbImageTag     string
	SeriesPrimaryImageTag   string
	UserData                struct {
		PlaybackPositionTicks int64
		Played                bool
		IsFavorite            bool
		UnplayedItemCount     int
	}
}

func (a apiItem) item() Item {
	it := Item{
		ID: a.Id, Type: a.Type, Name: a.Name, Overview: a.Overview, Year: a.ProductionYear, Genres: a.Genres,
		Rating: a.OfficialRating, Score: a.CommunityRating, Runtime: Seconds(a.RunTimeTicks),
		Position: Seconds(a.UserData.PlaybackPositionTicks), Played: a.UserData.Played, Favorite: a.UserData.IsFavorite,
		Unplayed: a.UserData.UnplayedItemCount, SeriesID: a.SeriesId, SeriesName: a.SeriesName, SeasonID: a.SeasonId,
		Season: a.ParentIndexNumber, Episode: a.IndexNumber,
	}
	if it.Genres == nil {
		it.Genres = []string{}
	}
	if t := a.ImageTags["Primary"]; t != "" {
		it.Primary, it.PrimaryItem = t, a.Id
	} else if a.SeriesPrimaryImageTag != "" {
		it.Primary, it.PrimaryItem = a.SeriesPrimaryImageTag, a.SeriesId
	}
	if len(a.BackdropImageTags) > 0 {
		it.Backdrop, it.BackdropItem = a.BackdropImageTags[0], a.Id
	} else if len(a.ParentBackdropImageTags) > 0 {
		it.Backdrop, it.BackdropItem = a.ParentBackdropImageTags[0], a.ParentBackdropItemId
	}
	if t := a.ImageTags["Logo"]; t != "" {
		it.Logo, it.LogoItem = t, a.Id
	} else if a.ParentLogoImageTag != "" {
		it.Logo, it.LogoItem = a.ParentLogoImageTag, a.ParentLogoItemId
	}
	if t := a.ImageTags["Thumb"]; t != "" {
		it.Thumb, it.ThumbItem = t, a.Id
	} else if a.ParentThumbImageTag != "" {
		it.Thumb, it.ThumbItem = a.ParentThumbImageTag, a.ParentThumbItemId
	}
	return it
}

func items(list []apiItem) []Item {
	out := make([]Item, 0, len(list))
	for _, a := range list {
		out = append(out, a.item())
	}
	return out
}

// campos extras que a interface usa (descrição, gêneros, imagens dos pais)
const fields = "Overview,Genres,ProductionYear,OfficialRating,CommunityRating,ParentId"

func (c *API) list(ctx context.Context, p string, q url.Values) ([]Item, int, error) {
	q.Set("Fields", fields)
	q.Set("EnableImageTypes", "Primary,Backdrop,Logo,Thumb")
	var res struct {
		Items            []apiItem
		TotalRecordCount int
	}
	if err := c.call(ctx, http.MethodGet, p+"?"+q.Encode(), nil, &res); err != nil {
		return nil, 0, err
	}
	return items(res.Items), res.TotalRecordCount, nil
}

// Resume: o que parou no meio (filmes e episódios).
func (c *API) Resume(ctx context.Context, limit int) ([]Item, error) {
	out, _, err := c.list(ctx, "/Users/"+c.UserID+"/Items/Resume", url.Values{
		"Limit": {strconv.Itoa(limit)}, "MediaTypes": {"Video"},
	})
	return out, err
}

// NextUp: o próximo episódio de cada série em andamento.
func (c *API) NextUp(ctx context.Context, limit int) ([]Item, error) {
	out, _, err := c.list(ctx, "/Shows/NextUp", url.Values{"UserId": {c.UserID}, "Limit": {strconv.Itoa(limit)}})
	return out, err
}

// Latest: o que chegou por último (Movie ou Series).
func (c *API) Latest(ctx context.Context, kind string, limit int) ([]Item, error) {
	q := url.Values{"IncludeItemTypes": {kind}, "Limit": {strconv.Itoa(limit)}, "Fields": {fields},
		"EnableImageTypes": {"Primary,Backdrop,Logo,Thumb"}}
	var res []apiItem
	if err := c.call(ctx, http.MethodGet, "/Users/"+c.UserID+"/Items/Latest?"+q.Encode(), nil, &res); err != nil {
		return nil, err
	}
	return items(res), nil
}

// Query filtra a lista de filmes e séries.
type Query struct {
	Types    string // "Movie", "Series" ou "Movie,Series"
	Sort     string // name, added, year, score
	Search   string
	Genre    string
	Favorite bool
	Start    int
	Limit    int
}

var sorts = map[string][2]string{
	"name":  {"SortName", "Ascending"},
	"added": {"DateCreated,SortName", "Descending"},
	"year":  {"ProductionYear,SortName", "Descending"},
	"score": {"CommunityRating,SortName", "Descending"},
}

// Items lista filmes e séries com filtro e paginação.
func (c *API) Items(ctx context.Context, q Query) ([]Item, int, error) {
	sort, ok := sorts[q.Sort]
	if !ok {
		sort = sorts["name"]
	}
	v := url.Values{"Recursive": {"true"}, "IncludeItemTypes": {q.Types}, "SortBy": {sort[0]}, "SortOrder": {sort[1]},
		"StartIndex": {strconv.Itoa(q.Start)}}
	if q.Limit > 0 {
		v.Set("Limit", strconv.Itoa(q.Limit))
	}
	if q.Search != "" {
		v.Set("SearchTerm", q.Search)
	}
	if q.Genre != "" {
		v.Set("Genres", q.Genre)
	}
	if q.Favorite {
		v.Set("Filters", "IsFavorite")
	}
	return c.list(ctx, "/Users/"+c.UserID+"/Items", v)
}

// Item busca um item.
func (c *API) Item(ctx context.Context, id string) (Item, error) {
	var a apiItem
	err := c.call(ctx, http.MethodGet, "/Users/"+c.UserID+"/Items/"+url.PathEscape(id), nil, &a)
	return a.item(), err
}

// Seasons lista as temporadas de uma série.
func (c *API) Seasons(ctx context.Context, series string) ([]Item, error) {
	out, _, err := c.list(ctx, "/Shows/"+url.PathEscape(series)+"/Seasons", url.Values{"UserId": {c.UserID}})
	return out, err
}

// Episodes lista os episódios de uma temporada (season vazio = todos).
func (c *API) Episodes(ctx context.Context, series, season string) ([]Item, error) {
	q := url.Values{"UserId": {c.UserID}}
	if season != "" {
		q.Set("SeasonId", season)
	}
	out, _, err := c.list(ctx, "/Shows/"+url.PathEscape(series)+"/Episodes", q)
	return out, err
}

// SetFavorite marca ou desmarca como favorito.
func (c *API) SetFavorite(ctx context.Context, id string, fav bool) error {
	return c.call(ctx, onOff(fav), "/Users/"+c.UserID+"/FavoriteItems/"+url.PathEscape(id), nil, nil)
}

// SetPlayed marca como assistido ou não.
func (c *API) SetPlayed(ctx context.Context, id string, played bool) error {
	return c.call(ctx, onOff(played), "/Users/"+c.UserID+"/PlayedItems/"+url.PathEscape(id), nil, nil)
}

func onOff(on bool) string {
	if on {
		return http.MethodPost
	}
	return http.MethodDelete
}

// Track é uma faixa de áudio ou legenda.
type Track struct {
	Index    int    `json:"index"`
	Label    string `json:"label"`
	Language string `json:"language"`
	Default  bool   `json:"default"`
	URL      string `json:"url,omitempty"` // legenda: caminho no servidor (WebVTT)
}

// Stream é como tocar um item: o caminho no servidor (direto ou HLS).
type Stream struct {
	Path      string  `json:"path"`
	HLS       bool    `json:"hls"`
	Session   string  `json:"session"`
	Source    string  `json:"source"`
	Duration  float64 `json:"duration"`
	Audio     []Track `json:"audio"`
	Subtitles []Track `json:"subtitles"`
	AudioIdx  int     `json:"audioIndex"`
}

// deviceProfile diz o que a webview toca sem ajuda (o resto vira HLS H.264/AAC).
func deviceProfile() map[string]any {
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
		"SubtitleProfiles": []map[string]string{
			{"Format": "vtt", "Method": "External"},
		},
	}
}

type apiStream struct {
	Type         string
	Index        int
	DisplayTitle string
	Language     string
	IsDefault    bool
	DeliveryUrl  string
}

// Play pergunta ao servidor como tocar (audio < 0 = a faixa padrão).
func (c *API) Play(ctx context.Context, id string, audio int, start float64) (Stream, error) {
	body := map[string]any{
		"UserId": c.UserID, "DeviceProfile": deviceProfile(), "StartTimeTicks": Ticks(start),
		"EnableDirectPlay": true, "EnableDirectStream": true, "EnableTranscoding": true, "AutoOpenLiveStream": true,
	}
	if audio >= 0 {
		body["AudioStreamIndex"] = audio
	}
	var res struct {
		PlaySessionId string
		MediaSources  []struct {
			Id                      string
			Container               string
			RunTimeTicks            int64
			SupportsDirectPlay      bool
			TranscodingUrl          string
			DefaultAudioStreamIndex *int
			MediaStreams            []apiStream
		}
		ErrorCode string
	}
	if err := c.call(ctx, http.MethodPost, "/Items/"+url.PathEscape(id)+"/PlaybackInfo", body, &res); err != nil {
		return Stream{}, err
	}
	if res.ErrorCode != "" || len(res.MediaSources) == 0 {
		return Stream{}, fmt.Errorf("o servidor não consegue tocar este vídeo (%s)", res.ErrorCode)
	}
	src := res.MediaSources[0]
	out := Stream{Session: res.PlaySessionId, Source: src.Id, Duration: Seconds(src.RunTimeTicks), Audio: []Track{}, Subtitles: []Track{}, AudioIdx: -1}
	if src.DefaultAudioStreamIndex != nil {
		out.AudioIdx = *src.DefaultAudioStreamIndex
	}
	if audio >= 0 {
		out.AudioIdx = audio
	}
	for _, s := range src.MediaStreams {
		t := Track{Index: s.Index, Label: s.DisplayTitle, Language: s.Language, Default: s.IsDefault}
		switch {
		case s.Type == "Audio":
			out.Audio = append(out.Audio, t)
		case s.Type == "Subtitle" && s.DeliveryUrl != "":
			t.URL = s.DeliveryUrl
			out.Subtitles = append(out.Subtitles, t)
		}
	}
	q := url.Values{"MediaSourceId": {src.Id}, "PlaySessionId": {res.PlaySessionId}}
	switch {
	case src.SupportsDirectPlay && audio < 0:
		q.Set("static", "true")
		out.Path = "/Videos/" + url.PathEscape(id) + "/stream?" + q.Encode()
	case src.TranscodingUrl != "":
		out.Path, out.HLS = src.TranscodingUrl, strings.Contains(src.TranscodingUrl, ".m3u8")
	default:
		return Stream{}, errors.New("o servidor não ofereceu um jeito de tocar este vídeo")
	}
	return out, nil
}

// Report avisa o servidor do progresso: event é "start", "progress" ou "stop".
func (c *API) Report(ctx context.Context, event, id, session, source string, position float64, paused bool) error {
	p := map[string]string{"start": "/Sessions/Playing", "progress": "/Sessions/Playing/Progress", "stop": "/Sessions/Playing/Stopped"}[event]
	if p == "" {
		return fmt.Errorf("evento desconhecido: %s", event)
	}
	return c.call(ctx, http.MethodPost, p, map[string]any{
		"ItemId": id, "PlaySessionId": session, "MediaSourceId": source,
		"PositionTicks": Ticks(position), "IsPaused": paused, "CanSeek": true, "PlayMethod": "DirectStream",
	}, nil)
}
