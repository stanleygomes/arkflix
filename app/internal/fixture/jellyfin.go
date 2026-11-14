package fixture

import (
	"bytes"
	"encoding/json"
	"image/color"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FakeJellyfin imita a API de um servidor Jellyfin (o que o app usa): login,
// filmes, séries, temporadas, episódios, continuar assistindo, favoritos,
// reprodução (o vídeo é o Clip) e relatórios de progresso. Usuário
// "stanley", senha "senha123".
type FakeJellyfin struct {
	mu      sync.Mutex
	items   map[string]*fakeItem
	order   []string
	reports []string // "start m1 12.0"
	auths   []string // cabeçalhos Authorization das mídias (o proxy põe o token)
	logouts int
}

type fakeItem struct {
	ID, Type, Name, Overview string
	Year                     int
	Genres                   []string
	Runtime                  float64
	Position                 float64
	Played, Favorite         bool
	SeriesID, SeasonID       string
	Season, Episode          int
	Added                    int
}

const (
	fakeJFUser  = "u-stanley"
	fakeJFToken = "tok-stanley"
)

// NewFakeJellyfin monta a biblioteca: três filmes e uma série com duas
// temporadas. "O Grande Teste" parou na metade; "Série de Teste" tem o
// episódio 2 como próximo.
func NewFakeJellyfin() *FakeJellyfin {
	f := &FakeJellyfin{items: map[string]*fakeItem{}}
	add := func(it fakeItem) {
		f.items[it.ID] = &it
		f.order = append(f.order, it.ID)
	}
	add(fakeItem{ID: "m1", Type: "Movie", Name: "O Grande Teste", Overview: "Um filme sobre testes de ponta a ponta.", Year: 2024, Genres: []string{"Ação", "Aventura"}, Runtime: 7200, Position: 1800, Added: 5})
	add(fakeItem{ID: "m2", Type: "Movie", Name: "Filme Sem Fim", Overview: "Já assistido e favorito.", Year: 2019, Genres: []string{"Drama"}, Runtime: 5400, Played: true, Favorite: true, Added: 3})
	add(fakeItem{ID: "m3", Type: "Movie", Name: "Curta Pixel", Overview: "Animação em pixel art.", Year: 2022, Genres: []string{"Animação"}, Runtime: 600, Added: 4})
	add(fakeItem{ID: "s1", Type: "Series", Name: "Série de Teste", Overview: "Uma série com duas temporadas.", Year: 2021, Genres: []string{"Comédia"}, Added: 6})
	add(fakeItem{ID: "s1-t1", Type: "Season", Name: "Temporada 1", SeriesID: "s1", Season: 1})
	add(fakeItem{ID: "s1-t2", Type: "Season", Name: "Temporada 2", SeriesID: "s1", Season: 2})
	add(fakeItem{ID: "e1", Type: "Episode", Name: "Piloto", Overview: "Onde tudo começa.", SeriesID: "s1", SeasonID: "s1-t1", Season: 1, Episode: 1, Runtime: 1500, Played: true})
	add(fakeItem{ID: "e2", Type: "Episode", Name: "O Segundo", Overview: "O episódio seguinte.", SeriesID: "s1", SeasonID: "s1-t1", Season: 1, Episode: 2, Runtime: 1500})
	add(fakeItem{ID: "e3", Type: "Episode", Name: "Final da Temporada", SeriesID: "s1", SeasonID: "s1-t1", Season: 1, Episode: 3, Runtime: 1500})
	add(fakeItem{ID: "e4", Type: "Episode", Name: "Recomeço", SeriesID: "s1", SeasonID: "s1-t2", Season: 2, Episode: 1, Runtime: 1500})
	return f
}

// Reports devolve os relatórios de progresso recebidos ("start m1 12.0").
func (f *FakeJellyfin) Reports() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reports)
}

// Auths devolve o Authorization dos pedidos de mídia (imagem, vídeo).
func (f *FakeJellyfin) Auths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.auths)
}

// Logouts conta os tokens invalidados.
func (f *FakeJellyfin) Logouts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logouts
}

// Item devolve uma cópia do item (testes conferem favorito e posição).
func (f *FakeJellyfin) Item(id string) (fakeItem, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	it, ok := f.items[id]
	if !ok {
		return fakeItem{}, false
	}
	return *it, true
}

func (f *FakeJellyfin) view(it *fakeItem) map[string]any {
	tags := map[string]string{"Primary": "p-" + it.ID}
	if it.Type == "Movie" || it.Type == "Series" {
		tags["Logo"] = "l-" + it.ID
	}
	v := map[string]any{
		"Id": it.ID, "Type": it.Type, "Name": it.Name, "Overview": it.Overview, "ProductionYear": it.Year,
		"Genres": it.Genres, "OfficialRating": "12", "CommunityRating": 7.5, "RunTimeTicks": int64(it.Runtime * 1e7),
		"ImageTags": tags,
		"UserData": map[string]any{"PlaybackPositionTicks": int64(it.Position * 1e7), "Played": it.Played, "IsFavorite": it.Favorite,
			"UnplayedItemCount": f.unplayed(it)},
	}
	if it.Type == "Movie" || it.Type == "Series" {
		v["BackdropImageTags"] = []string{"b-" + it.ID}
	}
	if it.SeriesID != "" {
		v["SeriesId"], v["SeriesName"] = it.SeriesID, f.items[it.SeriesID].Name
		v["ParentBackdropItemId"], v["ParentBackdropImageTags"] = it.SeriesID, []string{"b-" + it.SeriesID}
		v["ParentLogoItemId"], v["ParentLogoImageTag"] = it.SeriesID, "l-"+it.SeriesID
	}
	if it.Type == "Episode" {
		v["SeasonId"], v["ParentIndexNumber"], v["IndexNumber"] = it.SeasonID, it.Season, it.Episode
	}
	if it.Type == "Season" {
		v["IndexNumber"] = it.Season
	}
	return v
}

func (f *FakeJellyfin) unplayed(it *fakeItem) int {
	if it.Type != "Series" && it.Type != "Season" {
		return 0
	}
	n := 0
	for _, e := range f.items {
		if e.Type == "Episode" && !e.Played && (e.SeriesID == it.ID || e.SeasonID == it.ID) {
			n++
		}
	}
	return n
}

func (f *FakeJellyfin) list(keep func(*fakeItem) bool) []map[string]any {
	out := []map[string]any{}
	for _, id := range f.order {
		if it := f.items[id]; keep(it) {
			out = append(out, f.view(it))
		}
	}
	return out
}

func page(items []map[string]any, r *http.Request) map[string]any {
	total := len(items)
	start, _ := strconv.Atoi(r.URL.Query().Get("StartIndex"))
	start = min(start, total)
	items = items[start:]
	if limit, _ := strconv.Atoi(r.URL.Query().Get("Limit")); limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return map[string]any{"Items": items, "TotalRecordCount": total}
}

func (f *FakeJellyfin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := r.URL.Path
	q := r.URL.Query()
	auth := r.Header.Get("Authorization")
	switch {
	case p == "/System/Info/Public":
		writeJSON(w, map[string]any{"ServerName": "Jellyfin de Teste", "Version": "10.10.0"})
		return
	case p == "/Users/AuthenticateByName" && r.Method == http.MethodPost:
		var body struct{ Username, Pw string }
		json.NewDecoder(r.Body).Decode(&body)
		if body.Username != "stanley" || body.Pw != "senha123" || !strings.Contains(auth, "DeviceId=") {
			http.Error(w, "Error processing request.", http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"AccessToken": fakeJFToken, "ServerId": "srv1", "User": map[string]any{"Id": fakeJFUser, "Name": "stanley"}})
		return
	}
	if !strings.Contains(auth, `Token="`+fakeJFToken+`"`) {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	switch {
	case p == "/Sessions/Logout":
		f.logouts++
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 4 && parts[0] == "Items" && parts[2] == "Images":
		f.auths = append(f.auths, auth)
		c := map[string]color.RGBA{"Primary": {R: 40, G: 90, B: 200, A: 255}, "Backdrop": {R: 30, G: 30, B: 60, A: 255}}[parts[3]]
		if c.A == 0 {
			c = color.RGBA{R: 230, G: 230, B: 230, A: 255}
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(CoverPNG(c))
	case len(parts) >= 3 && parts[0] == "Videos" && strings.Contains(p, "/Subtitles/"):
		w.Header().Set("Content-Type", "text/vtt")
		w.Write([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:02.000\nLegenda de teste\n"))
	case len(parts) == 3 && parts[0] == "Videos" && strings.HasPrefix(parts[2], "stream"):
		f.auths = append(f.auths, auth)
		w.Header().Set("Content-Type", "video/webm")
		http.ServeContent(w, r, "clip.webm", time.Unix(0, 0), bytes.NewReader(Clip))
	case len(parts) == 3 && parts[0] == "Items" && parts[2] == "PlaybackInfo":
		f.playbackInfo(w, r, parts[1])
	case p == "/Sessions/Playing" || p == "/Sessions/Playing/Progress" || p == "/Sessions/Playing/Stopped":
		f.report(w, r)
	case p == "/Shows/NextUp":
		writeJSON(w, page(f.list(func(it *fakeItem) bool { return it.ID == "e2" }), r))
	case len(parts) == 3 && parts[0] == "Shows" && parts[2] == "Seasons":
		writeJSON(w, page(f.list(func(it *fakeItem) bool { return it.Type == "Season" && it.SeriesID == parts[1] }), r))
	case len(parts) == 3 && parts[0] == "Shows" && parts[2] == "Episodes":
		season := q.Get("SeasonId")
		writeJSON(w, page(f.list(func(it *fakeItem) bool {
			return it.Type == "Episode" && it.SeriesID == parts[1] && (season == "" || it.SeasonID == season)
		}), r))
	case len(parts) >= 3 && parts[0] == "Users" && parts[1] == fakeJFUser:
		f.user(w, r, parts[2:])
	default:
		http.NotFound(w, r)
	}
}

func (f *FakeJellyfin) user(w http.ResponseWriter, r *http.Request, parts []string) {
	q := r.URL.Query()
	switch {
	case len(parts) == 2 && parts[0] == "Items" && parts[1] == "Resume":
		writeJSON(w, page(f.list(func(it *fakeItem) bool { return it.Position > 0 && !it.Played }), r))
	case len(parts) == 2 && parts[0] == "Items" && parts[1] == "Latest":
		kind := q.Get("IncludeItemTypes")
		items := f.list(func(it *fakeItem) bool { return it.Type == kind })
		slices.SortFunc(items, func(a, b map[string]any) int {
			return f.items[b["Id"].(string)].Added - f.items[a["Id"].(string)].Added
		})
		writeJSON(w, items)
	case len(parts) == 2 && parts[0] == "Items":
		it, ok := f.items[parts[1]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, f.view(it))
	case len(parts) == 1 && parts[0] == "Items":
		types := strings.Split(q.Get("IncludeItemTypes"), ",")
		term := strings.ToLower(q.Get("SearchTerm"))
		fav := q.Get("Filters") == "IsFavorite"
		genre := q.Get("Genres")
		items := f.list(func(it *fakeItem) bool {
			return slices.Contains(types, it.Type) && (term == "" || strings.Contains(strings.ToLower(it.Name), term)) &&
				(!fav || it.Favorite) && (genre == "" || slices.Contains(it.Genres, genre))
		})
		if strings.HasPrefix(q.Get("SortBy"), "DateCreated") {
			slices.SortStableFunc(items, func(a, b map[string]any) int {
				return f.items[b["Id"].(string)].Added - f.items[a["Id"].(string)].Added
			})
		} else {
			slices.SortStableFunc(items, func(a, b map[string]any) int { return strings.Compare(a["Name"].(string), b["Name"].(string)) })
		}
		writeJSON(w, page(items, r))
	case len(parts) == 2 && (parts[0] == "FavoriteItems" || parts[0] == "PlayedItems"):
		it, ok := f.items[parts[1]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		on := r.Method == http.MethodPost
		if parts[0] == "FavoriteItems" {
			it.Favorite = on
		} else {
			it.Played = on
			it.Position = 0
		}
		writeJSON(w, map[string]any{"IsFavorite": it.Favorite, "Played": it.Played})
	default:
		http.NotFound(w, r)
	}
}

func (f *FakeJellyfin) playbackInfo(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := f.items[id]; !ok {
		http.NotFound(w, r)
		return
	}
	var body struct{ AudioStreamIndex *int }
	json.NewDecoder(r.Body).Decode(&body)
	def := 1
	src := map[string]any{
		"Id": id, "Container": "webm", "RunTimeTicks": int64(f.items[id].Runtime * 1e7), "SupportsDirectPlay": body.AudioStreamIndex == nil,
		"DefaultAudioStreamIndex": def,
		"MediaStreams": []map[string]any{
			{"Type": "Video", "Index": 0, "DisplayTitle": "VP8"},
			{"Type": "Audio", "Index": 1, "DisplayTitle": "Português - Stereo", "Language": "por", "IsDefault": true},
			{"Type": "Audio", "Index": 2, "DisplayTitle": "Inglês - Stereo", "Language": "eng"},
			{"Type": "Subtitle", "Index": 3, "DisplayTitle": "Português", "Language": "por", "DeliveryUrl": "/Videos/" + id + "/" + id + "/Subtitles/3/0/Stream.vtt"},
		},
	}
	if body.AudioStreamIndex != nil {
		src["TranscodingUrl"] = "/Videos/" + id + "/stream.webm?AudioStreamIndex=" + strconv.Itoa(*body.AudioStreamIndex)
	}
	writeJSON(w, map[string]any{"PlaySessionId": "sess-" + id, "MediaSources": []any{src}})
}

func (f *FakeJellyfin) report(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ItemId        string
		PositionTicks int64
	}
	json.NewDecoder(r.Body).Decode(&body)
	event := map[string]string{"/Sessions/Playing": "start", "/Sessions/Playing/Progress": "progress", "/Sessions/Playing/Stopped": "stop"}[r.URL.Path]
	pos := float64(body.PositionTicks) / 1e7
	f.reports = append(f.reports, event+" "+body.ItemId+" "+strconv.FormatFloat(pos, 'f', 1, 64))
	if it, ok := f.items[body.ItemId]; ok && event == "stop" {
		it.Position = pos
		if it.Runtime > 0 && pos/it.Runtime > 0.9 {
			it.Played, it.Position = true, 0
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
