package jellyfin

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// O app arkflix (app/internal/jellyfin) faz exatamente estas chamadas e
// decodifica estes campos; aqui elas rodam contra o servidor de verdade.

const arkFields = "Overview,Genres,ProductionYear,OfficialRating,CommunityRating,ParentId"

type arkItem struct {
	Id, Type, Name, Overview, OfficialRating string
	ProductionYear                           int
	Genres                                   []string
	CommunityRating                          float64
	RunTimeTicks                             int64
	SeriesId, SeriesName, SeasonId           string
	ParentIndexNumber, IndexNumber           int
	ImageTags                                map[string]string
	BackdropImageTags                        []string
	ParentBackdropItemId                     string
	ParentBackdropImageTags                  []string
	ParentLogoItemId, ParentLogoImageTag     string
	SeriesPrimaryImageTag                    string
	UserData                                 struct {
		PlaybackPositionTicks int64
		Played, IsFavorite    bool
		UnplayedItemCount     int
	}
}

type arkList struct {
	Items            []arkItem
	TotalRecordCount int
}

func arkNames(items []arkItem) string {
	var out []string
	for _, it := range items {
		out = append(out, it.Name)
	}
	return strings.Join(out, "|")
}

type arkClient struct {
	e                         *env
	token, userID, deviceID   string
	device, version, serverNm string
}

func (a *arkClient) auth() string {
	h := fmt.Sprintf(`MediaBrowser Client=%q, Device=%q, DeviceId=%q, Version=%q`, "arkflix", a.device, a.deviceID, a.version)
	if a.token != "" {
		h += fmt.Sprintf(`, Token=%q`, a.token)
	}
	return h
}

func (a *arkClient) call(method, p string, body, out any) int {
	a.e.t.Helper()
	return a.e.call(method, p, body, out, "Authorization", a.auth(), "Accept", "application/json")
}

func (a *arkClient) list(p string, q url.Values) arkList {
	a.e.t.Helper()
	q.Set("Fields", arkFields)
	q.Set("EnableImageTypes", "Primary,Backdrop,Logo,Thumb")
	var res arkList
	if code := a.call("GET", p+"?"+q.Encode(), nil, &res); code != 200 {
		a.e.t.Fatalf("%s: %d", p, code)
	}
	return res
}

func TestArkflixClient(t *testing.T) {
	e := setup(t)
	a := &arkClient{e: e, deviceID: "arkflix-desktop-1", device: "notebook", version: "1.2.3"}

	// Login + /System/Info/Public
	var login struct {
		AccessToken, ServerId string
		User                  struct{ Id, Name string }
	}
	eq(t, "login errado", a.call("POST", "/Users/AuthenticateByName", map[string]string{"Username": testkit.Admin, "Pw": "x"}, nil), 401)
	eq(t, "login", a.call("POST", "/Users/AuthenticateByName", map[string]string{"Username": testkit.Admin, "Pw": testkit.AdminPass}, &login), 200)
	if login.AccessToken == "" || login.User.Id == "" {
		t.Fatalf("%+v", login)
	}
	a.token, a.userID = login.AccessToken, login.User.Id
	var info struct{ ServerName string }
	eq(t, "info", a.call("GET", "/System/Info/Public", nil, &info), 200)
	eq(t, "server name", info.ServerName, "arkflix de teste")

	// Resume (vazio no começo)
	ctx := t.Context()
	big, _ := e.k.Store.Item(ctx, e.ids.MovieBig)
	e.k.Store.Progress(ctx, a.userID, big, 1.0)
	res := a.list("/Users/"+a.userID+"/Items/Resume", url.Values{"Limit": {"12"}, "MediaTypes": {"Video"}})
	eq(t, "resume", arkNames(res.Items), "O Grande Teste")
	eq(t, "posição", res.Items[0].UserData.PlaybackPositionTicks, int64(10_000_000))
	if res.Items[0].ImageTags["Primary"] == "" {
		t.Fatal("resume sem Primary")
	}

	// Latest (lista pura)
	var latest []arkItem
	q := url.Values{"IncludeItemTypes": {"Movie"}, "Limit": {"16"}, "Fields": {arkFields}, "EnableImageTypes": {"Primary,Backdrop,Logo,Thumb"}}
	eq(t, "latest", a.call("GET", "/Users/"+a.userID+"/Items/Latest?"+q.Encode(), nil, &latest), 200)
	eq(t, "latest filmes", len(latest), 3)
	q.Set("IncludeItemTypes", "Series")
	a.call("GET", "/Users/"+a.userID+"/Items/Latest?"+q.Encode(), nil, &latest)
	eq(t, "latest séries", arkNames(latest), "Série de Teste")
	eq(t, "unplayed", latest[0].UserData.UnplayedItemCount, 3)

	// Items: como o app filtra e ordena
	items := func(types, sort, order, search, genre string, fav bool, start, limit int) arkList {
		v := url.Values{"Recursive": {"true"}, "IncludeItemTypes": {types}, "SortBy": {sort}, "SortOrder": {order},
			"StartIndex": {strconv.Itoa(start)}}
		if limit > 0 {
			v.Set("Limit", strconv.Itoa(limit))
		}
		if search != "" {
			v.Set("SearchTerm", search)
		}
		if genre != "" {
			v.Set("Genres", genre)
		}
		if fav {
			v.Set("Filters", "IsFavorite")
		}
		return a.list("/Users/"+a.userID+"/Items", v)
	}
	l := items("Movie,Series", "SortName", "Ascending", "", "", false, 0, 0)
	eq(t, "nome", arkNames(l.Items), "Curta Pixel|Filme Sem Ano|O Grande Teste|Série de Teste")
	l = items("Movie,Series", "DateCreated,SortName", "Descending", "", "", false, 0, 2)
	eq(t, "adicionados: total", l.TotalRecordCount, 4)
	eq(t, "adicionados: página", len(l.Items), 2)
	l = items("Movie", "ProductionYear,SortName", "Descending", "", "", false, 0, 0)
	eq(t, "ano", arkNames(l.Items), "O Grande Teste|Curta Pixel|Filme Sem Ano")
	l = items("Movie,Series", "CommunityRating,SortName", "Descending", "grande", "", false, 0, 0)
	eq(t, "busca", arkNames(l.Items), "O Grande Teste")

	// Item
	var one arkItem
	eq(t, "item", a.call("GET", "/Users/"+a.userID+"/Items/"+url.PathEscape(e.ids.MovieBig), nil, &one), 200)
	eq(t, "item ano", one.ProductionYear, 2021)
	eq(t, "item runtime", one.RunTimeTicks, int64(20_030_000))

	// Seasons / Episodes
	l = a.list("/Shows/"+url.PathEscape(e.ids.Series)+"/Seasons", url.Values{"UserId": {a.userID}})
	eq(t, "temporadas", arkNames(l.Items), "Temporada 1|Temporada 2")
	eq(t, "índice", l.Items[1].IndexNumber, 2)
	l = a.list("/Shows/"+url.PathEscape(e.ids.Series)+"/Episodes", url.Values{"UserId": {a.userID}, "SeasonId": {e.ids.Season1}})
	eq(t, "episódios", arkNames(l.Items), "O Começo|O Segundo")
	ep := l.Items[1]
	eq(t, "série", ep.SeriesId, e.ids.Series)
	eq(t, "nome série", ep.SeriesName, "Série de Teste")
	eq(t, "temporada", ep.SeasonId, e.ids.Season1)
	eq(t, "SxE", fmt.Sprint(ep.ParentIndexNumber, "x", ep.IndexNumber), "1x2")
	if ep.SeriesPrimaryImageTag == "" {
		t.Fatal("episódio sem imagem da série")
	}
	l = a.list("/Shows/"+url.PathEscape(e.ids.Series)+"/Episodes", url.Values{"UserId": {a.userID}})
	eq(t, "todos episódios", len(l.Items), 3)

	// SetPlayed / SetFavorite (POST liga, DELETE desliga)
	eq(t, "played", a.call(http.MethodPost, "/Users/"+a.userID+"/PlayedItems/"+url.PathEscape(e.ids.Ep1), nil, nil), 200)
	l = a.list("/Shows/NextUp", url.Values{"UserId": {a.userID}, "Limit": {"12"}})
	eq(t, "nextup", arkNames(l.Items), "O Segundo")
	eq(t, "fav", a.call(http.MethodPost, "/Users/"+a.userID+"/FavoriteItems/"+url.PathEscape(e.ids.MoviePixel), nil, nil), 200)
	l = items("Movie,Series", "SortName", "Ascending", "", "", true, 0, 0)
	eq(t, "favoritos", arkNames(l.Items), "Curta Pixel")
	eq(t, "desfav", a.call(http.MethodDelete, "/Users/"+a.userID+"/FavoriteItems/"+url.PathEscape(e.ids.MoviePixel), nil, nil), 200)
	eq(t, "sem favoritos", items("Movie,Series", "SortName", "Ascending", "", "", true, 0, 0).TotalRecordCount, 0)

	// Play: PlaybackInfo com o DeviceProfile do app
	type arkStream struct {
		Type         string
		Index        int
		DisplayTitle string
		Language     string
		IsDefault    bool
		DeliveryUrl  string
	}
	var play struct {
		PlaySessionId string
		MediaSources  []struct {
			Id                      string
			Container               string
			RunTimeTicks            int64
			SupportsDirectPlay      bool
			TranscodingUrl          string
			DefaultAudioStreamIndex *int
			MediaStreams            []arkStream
		}
		ErrorCode string
	}
	body := map[string]any{"UserId": a.userID, "DeviceProfile": arkflixProfile(), "StartTimeTicks": 0,
		"EnableDirectPlay": true, "EnableDirectStream": true, "EnableTranscoding": true, "AutoOpenLiveStream": true}
	eq(t, "play", a.call("POST", "/Items/"+url.PathEscape(e.ids.MovieBig)+"/PlaybackInfo", body, &play), 200)
	if play.ErrorCode != "" || len(play.MediaSources) == 0 || play.PlaySessionId == "" {
		t.Fatalf("%+v", play)
	}
	src := play.MediaSources[0]
	eq(t, "direto", src.SupportsDirectPlay, true)
	eq(t, "fonte", src.Id, e.ids.MovieBig)
	eq(t, "áudio padrão", *src.DefaultAudioStreamIndex, 1)
	var audio, subs []arkStream
	for _, s := range src.MediaStreams {
		switch {
		case s.Type == "Audio":
			audio = append(audio, s)
		case s.Type == "Subtitle" && s.DeliveryUrl != "":
			subs = append(subs, s)
		}
	}
	eq(t, "faixas de áudio", len(audio), 1)
	eq(t, "legendas", len(subs), 1)
	eq(t, "legenda pt", subs[0].Language, "por")
	// o app toca o caminho que ele mesmo monta e a legenda pelo DeliveryUrl
	sq := url.Values{"MediaSourceId": {src.Id}, "PlaySessionId": {play.PlaySessionId}, "static": {"true"}}
	resp, raw := e.req("GET", "/Videos/"+url.PathEscape(e.ids.MovieBig)+"/stream?"+sq.Encode(), nil, "Authorization", a.auth(), "Range", "bytes=0-")
	if resp.StatusCode != 206 && resp.StatusCode != 200 || len(raw) == 0 {
		t.Fatalf("stream: %d", resp.StatusCode)
	}
	resp, raw = e.req("GET", subs[0].DeliveryUrl, nil, "Authorization", a.auth())
	eq(t, "legenda", resp.StatusCode, 200)
	if !strings.HasPrefix(string(raw), "WEBVTT") {
		t.Fatalf("%q", raw)
	}
	// imagens que o app pede (o proxy põe o token)
	for _, img := range []string{"/Items/" + e.ids.MovieBig + "/Images/Primary?fillWidth=300&quality=90",
		"/Items/" + ep.SeriesId + "/Images/Primary?maxWidth=400"} {
		resp, _ = e.req("GET", img, nil, "Authorization", a.auth())
		eq(t, img, resp.StatusCode, 200)
	}

	// escolheu outra faixa de áudio: o app exige TranscodingUrl
	if e.k.Config.FFmpeg != "" {
		body["AudioStreamIndex"] = 1
		a.call("POST", "/Items/"+url.PathEscape(e.ids.MovieBig)+"/PlaybackInfo", body, &play)
		if !strings.Contains(play.MediaSources[0].TranscodingUrl, ".m3u8") {
			t.Fatalf("sem HLS com AudioStreamIndex: %+v", play.MediaSources[0])
		}
	}

	// Report: start / progress / stop
	rep := func(p string, sec float64) {
		t.Helper()
		eq(t, p, a.call("POST", p, map[string]any{"ItemId": e.ids.Ep2, "PlaySessionId": play.PlaySessionId, "MediaSourceId": e.ids.Ep2,
			"PositionTicks": int64(sec * 1e7), "IsPaused": false, "CanSeek": true, "PlayMethod": "DirectStream"}, nil), 204)
	}
	rep("/Sessions/Playing", 0)
	rep("/Sessions/Playing/Progress", 1.0)
	eq(t, "item", a.call("GET", "/Users/"+a.userID+"/Items/"+e.ids.Ep2, nil, &one), 200)
	eq(t, "progresso", one.UserData.PlaybackPositionTicks, int64(10_000_000))
	rep("/Sessions/Playing/Stopped", 1.95)
	a.call("GET", "/Users/"+a.userID+"/Items/"+e.ids.Ep2, nil, &one)
	eq(t, "visto no fim", one.UserData.Played, true)

	// Logout invalida o token
	eq(t, "logout", a.call("POST", "/Sessions/Logout", nil, nil), 204)
	eq(t, "depois do logout", a.call("GET", "/Users/"+a.userID+"/Items/Resume", nil, nil), 401)
}
