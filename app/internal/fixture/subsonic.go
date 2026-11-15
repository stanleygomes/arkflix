package fixture

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/subsonic"
)

// Áudios de 2 s (os mesmos dos testes do servidor).
var (
	//go:embed testdata/tone.mp3
	ToneMP3 []byte
	//go:embed testdata/solo.mp3
	SoloMP3 []byte
	//go:embed testdata/tone.flac
	ToneFLAC []byte
)

// FakeSubsonic imita um Navidrome (o que o app usa da API Subsonic, em JSON):
// login por token (t/s), artistas, álbuns, faixas, busca, favoritas,
// playlists, scrobble, fila, scan e usuários. Usuários "stanley" (admin) e
// "ana", senha "senha123". Toda faixa toca um áudio de verdade de 2 s.
type FakeSubsonic struct {
	mu        sync.Mutex
	artists   []*fakeArtist
	albums    []*fakeAlbum
	songs     map[string]*fakeSong
	order     []string // faixas na ordem do acervo
	lists     []*fakeList
	nextList  int
	starred   map[string]time.Time
	plays     map[string]int64
	scrobbles []string // "now t1", "play t1"
	queries   []url.Values
	queue     url.Values
	scanning  int // pedidos de getScanStatus até o scan acabar
	lastScan  time.Time
	// Type: o que o ping diz que é (navidrome ou arkflix-server).
	Type string
	// NoAPIKey: não anuncia apiKeyAuthentication (Navidrome antigo).
	NoAPIKey bool
}

type fakeArtist struct{ ID, Name string }

type fakeAlbum struct {
	ID, Name, ArtistID, Genre string
	Year                      int
	Songs                     []string
	Created                   int
}

type fakeSong struct {
	ID, Title, AlbumID string
	Track              int
	Suffix             string
	Data               []byte
}

type fakeList struct {
	ID, Name, Owner string
	Songs           []string
	Changed         time.Time
}

// NewFakeSubsonic monta o acervo: dois artistas, três álbuns, cinco faixas e
// uma playlist do stanley.
func NewFakeSubsonic() *FakeSubsonic {
	f := &FakeSubsonic{songs: map[string]*fakeSong{}, starred: map[string]time.Time{}, plays: map[string]int64{}, Type: "navidrome", nextList: 2}
	f.artists = []*fakeArtist{{ID: "ar1", Name: "Banda Teste"}, {ID: "ar2", Name: "Orquestra Pixel"}}
	song := func(id, title, album string, track int, suffix string, data []byte) string {
		f.songs[id] = &fakeSong{ID: id, Title: title, AlbumID: album, Track: track, Suffix: suffix, Data: data}
		f.order = append(f.order, id)
		return id
	}
	f.albums = []*fakeAlbum{
		{ID: "al1", Name: "Disco Teste", ArtistID: "ar1", Genre: "Rock", Year: 2021, Created: 3,
			Songs: []string{song("t1", "Bip", "al1", 1, "mp3", ToneMP3), song("t2", "Solo de Teste", "al1", 2, "mp3", SoloMP3)}},
		{ID: "al2", Name: "Ao Vivo no Teste", ArtistID: "ar1", Genre: "Rock", Year: 2023, Created: 5,
			Songs: []string{song("t3", "Bip Sem Perdas", "al2", 1, "flac", ToneFLAC)}},
		{ID: "al3", Name: "Trilhas 8-bit", ArtistID: "ar2", Genre: "Chiptune", Year: 2019, Created: 1,
			Songs: []string{song("t4", "Tema do Menu", "al3", 1, "mp3", SoloMP3), song("t5", "Fase Final", "al3", 2, "mp3", ToneMP3)}},
	}
	f.starred["al3"] = time.Unix(1_700_000_000, 0)
	f.starred["t2"] = time.Unix(1_700_000_000, 0)
	f.plays["t4"], f.plays["t1"] = 9, 3
	f.lists = []*fakeList{{ID: "pl1", Name: "Pra Viagem", Owner: "stanley", Songs: []string{"t1", "t4"}, Changed: time.Unix(1_700_000_000, 0)}}
	f.lastScan = time.Unix(1_700_000_000, 0)
	return f
}

// Scrobbles devolve os avisos recebidos ("now t1" = tocando, "play t1" = contou).
func (f *FakeSubsonic) Scrobbles() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.scrobbles)
}

// Queries devolve as queries recebidas (testes conferem que não vai senha).
func (f *FakeSubsonic) Queries() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.queries)
}

// SavedQueue é a última fila salva (savePlayQueue).
func (f *FakeSubsonic) SavedQueue() url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queue
}

// Starred diz se o item está nas favoritas.
func (f *FakeSubsonic) Starred(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.starred[id]
	return ok
}

// PlaylistSongs são as faixas de uma playlist (nil = não existe).
func (f *FakeSubsonic) PlaylistSongs(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if l := f.list(id); l != nil {
		return slices.Clone(l.Songs)
	}
	return nil
}

var fakeSubUsers = map[string]bool{"stanley": true, "ana": false} // nome -> admin

const fakeSubPass = "senha123"

func (f *FakeSubsonic) artist(id string) *fakeArtist {
	for _, a := range f.artists {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func (f *FakeSubsonic) album(id string) *fakeAlbum {
	for _, a := range f.albums {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func (f *FakeSubsonic) list(id string) *fakeList {
	for _, l := range f.lists {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func (f *FakeSubsonic) star(id string, v map[string]any) map[string]any {
	if at, ok := f.starred[id]; ok {
		v["starred"] = at.UTC().Format(time.RFC3339)
	}
	return v
}

func (f *FakeSubsonic) songView(id string) map[string]any {
	s := f.songs[id]
	al := f.album(s.AlbumID)
	ar := f.artist(al.ArtistID)
	ct := map[string]string{"mp3": "audio/mpeg", "flac": "audio/flac"}[s.Suffix]
	return f.star(id, map[string]any{
		"id": s.ID, "parent": al.ID, "isDir": false, "title": s.Title, "album": al.Name, "albumId": al.ID,
		"artist": ar.Name, "artistId": ar.ID, "track": s.Track, "discNumber": 1, "year": al.Year, "genre": al.Genre,
		"coverArt": al.ID, "size": len(s.Data), "contentType": ct, "suffix": s.Suffix, "duration": 2, "bitRate": 128,
		"path":      ar.Name + "/" + al.Name + "/" + fmt.Sprintf("%02d - %s.%s", s.Track, s.Title, s.Suffix),
		"playCount": f.plays[id], "type": "music", "mediaType": "song",
	})
}

func (f *FakeSubsonic) albumView(a *fakeAlbum) map[string]any {
	ar := f.artist(a.ArtistID)
	var plays int64
	for _, s := range a.Songs {
		plays += f.plays[s]
	}
	return f.star(a.ID, map[string]any{
		"id": a.ID, "name": a.Name, "artist": ar.Name, "artistId": ar.ID, "coverArt": a.ID, "songCount": len(a.Songs),
		"duration": 2 * len(a.Songs), "year": a.Year, "genre": a.Genre, "playCount": plays,
		"created": time.Unix(1_700_000_000+int64(a.Created)*86400, 0).UTC().Format(time.RFC3339),
	})
}

func (f *FakeSubsonic) artistView(a *fakeArtist) map[string]any {
	n := 0
	for _, al := range f.albums {
		if al.ArtistID == a.ID {
			n++
		}
	}
	return f.star(a.ID, map[string]any{"id": a.ID, "name": a.Name, "albumCount": n, "coverArt": "ar-" + a.ID})
}

func (f *FakeSubsonic) listView(l *fakeList, entries bool) map[string]any {
	v := map[string]any{"id": l.ID, "name": l.Name, "owner": l.Owner, "public": false, "songCount": len(l.Songs),
		"duration": 2 * len(l.Songs), "changed": l.Changed.UTC().Format(time.RFC3339), "coverArt": "pl-" + l.ID}
	if entries {
		out := []map[string]any{}
		for _, id := range l.Songs {
			if _, ok := f.songs[id]; ok {
				out = append(out, f.songView(id))
			}
		}
		v["entry"] = out
	}
	return v
}

// user confere a autenticação do jeito do Navidrome (sem p=: o app nunca
// manda a senha depois do login, e o falso recusa pra pegar regressão).
func (f *FakeSubsonic) user(q url.Values) (string, int) {
	if q.Get("v") == "" || q.Get("c") == "" {
		return "", 10
	}
	if q.Get("p") != "" {
		return "", 41
	}
	name := q.Get("u")
	_, ok := fakeSubUsers[name]
	if !ok || q.Get("s") == "" || subsonic.TokenFor(fakeSubPass, q.Get("s")) != strings.ToLower(q.Get("t")) {
		return "", 40
	}
	return name, 0
}

func (f *FakeSubsonic) ok(w http.ResponseWriter, fields map[string]any) {
	body := map[string]any{"status": "ok", "version": "1.16.1", "type": f.Type, "serverVersion": "0.64.2 (fake)", "openSubsonic": true}
	for k, v := range fields {
		body[k] = v
	}
	writeJSON(w, map[string]any{"subsonic-response": body})
}

func (f *FakeSubsonic) fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, map[string]any{"subsonic-response": map[string]any{"status": "failed", "version": "1.16.1", "type": f.Type,
		"serverVersion": "0.64.2 (fake)", "openSubsonic": true, "error": map[string]any{"code": code, "message": msg}}})
}

func (f *FakeSubsonic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	method, ok := strings.CutPrefix(r.URL.Path, "/rest/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	method = strings.TrimSuffix(method, ".view")
	r.ParseForm()
	q := r.Form
	f.queries = append(f.queries, q)
	if method == "getOpenSubsonicExtensions" {
		ext := []map[string]any{{"name": "formPost", "versions": []int{1}}, {"name": "transcodeOffset", "versions": []int{1}}}
		if !f.NoAPIKey {
			ext = append(ext, map[string]any{"name": "apiKeyAuthentication", "versions": []int{1}})
		}
		f.ok(w, map[string]any{"openSubsonicExtensions": ext})
		return
	}
	user, code := f.user(q)
	if code != 0 {
		f.fail(w, code, map[int]string{10: "Required parameter is missing", 40: "Wrong username or password", 41: "Token authentication not supported"}[code])
		return
	}
	admin := fakeSubUsers[user]
	switch method {
	case "ping":
		f.ok(w, nil)
	case "getUser":
		name := q.Get("username")
		a, ok := fakeSubUsers[name]
		if !ok || (name != user && !admin) {
			f.fail(w, 50, "User is not authorized for the given operation")
			return
		}
		f.ok(w, map[string]any{"user": map[string]any{"username": name, "adminRole": a, "streamRole": true, "downloadRole": true, "playlistRole": true, "scrobblingEnabled": true}})
	case "getUsers":
		if !admin {
			f.fail(w, 50, "User is not authorized for the given operation")
			return
		}
		f.ok(w, map[string]any{"users": map[string]any{"user": []map[string]any{
			{"username": "ana", "adminRole": false, "streamRole": true}, {"username": "stanley", "adminRole": true, "streamRole": true}}}})
	case "getArtists":
		idx := map[string][]map[string]any{}
		var letters []string
		for _, a := range f.artists {
			l := strings.ToUpper(a.Name[:1])
			if _, ok := idx[l]; !ok {
				letters = append(letters, l)
			}
			idx[l] = append(idx[l], f.artistView(a))
		}
		slices.Sort(letters)
		out := []map[string]any{}
		for _, l := range letters {
			out = append(out, map[string]any{"name": l, "artist": idx[l]})
		}
		f.ok(w, map[string]any{"artists": map[string]any{"ignoredArticles": "The", "index": out}})
	case "getArtist":
		a := f.artist(q.Get("id"))
		if a == nil {
			f.fail(w, 70, "Artist not found")
			return
		}
		v := f.artistView(a)
		albums := []map[string]any{}
		for _, al := range f.albums {
			if al.ArtistID == a.ID {
				albums = append(albums, f.albumView(al))
			}
		}
		v["album"] = albums
		f.ok(w, map[string]any{"artist": v})
	case "getTopSongs":
		songs := []map[string]any{}
		for _, id := range f.order {
			s := f.songs[id]
			if f.artist(f.album(s.AlbumID).ArtistID).Name == q.Get("artist") {
				songs = append(songs, f.songView(id))
			}
		}
		slices.SortStableFunc(songs, func(a, b map[string]any) int { return int(b["playCount"].(int64) - a["playCount"].(int64)) })
		f.ok(w, map[string]any{"topSongs": map[string]any{"song": songs}})
	case "getAlbumList2":
		f.albumList(w, q)
	case "getAlbum":
		a := f.album(q.Get("id"))
		if a == nil {
			f.fail(w, 70, "Album not found")
			return
		}
		v := f.albumView(a)
		songs := []map[string]any{}
		for _, id := range a.Songs {
			songs = append(songs, f.songView(id))
		}
		v["song"] = songs
		f.ok(w, map[string]any{"album": v})
	case "getSong":
		if _, ok := f.songs[q.Get("id")]; !ok {
			f.fail(w, 70, "Song not found")
			return
		}
		f.ok(w, map[string]any{"song": f.songView(q.Get("id"))})
	case "search3":
		term := strings.ToLower(strings.Trim(q.Get("query"), `"*`))
		res := map[string]any{"artist": []map[string]any{}, "album": []map[string]any{}, "song": []map[string]any{}}
		match := func(s string) bool { return term == "" || strings.Contains(strings.ToLower(s), term) }
		for _, a := range f.artists {
			if match(a.Name) {
				res["artist"] = append(res["artist"].([]map[string]any), f.artistView(a))
			}
		}
		for _, a := range f.albums {
			if match(a.Name) {
				res["album"] = append(res["album"].([]map[string]any), f.albumView(a))
			}
		}
		for _, id := range f.order {
			if match(f.songs[id].Title) {
				res["song"] = append(res["song"].([]map[string]any), f.songView(id))
			}
		}
		f.ok(w, map[string]any{"searchResult3": res})
	case "getStarred2":
		res := map[string]any{"artist": []map[string]any{}, "album": []map[string]any{}, "song": []map[string]any{}}
		for _, a := range f.artists {
			if _, ok := f.starred[a.ID]; ok {
				res["artist"] = append(res["artist"].([]map[string]any), f.artistView(a))
			}
		}
		for _, a := range f.albums {
			if _, ok := f.starred[a.ID]; ok {
				res["album"] = append(res["album"].([]map[string]any), f.albumView(a))
			}
		}
		for _, id := range f.order {
			if _, ok := f.starred[id]; ok {
				res["song"] = append(res["song"].([]map[string]any), f.songView(id))
			}
		}
		f.ok(w, map[string]any{"starred2": res})
	case "star", "unstar":
		for _, k := range []string{"id", "albumId", "artistId"} {
			for _, id := range q[k] {
				if method == "star" {
					f.starred[id] = time.Now()
				} else {
					delete(f.starred, id)
				}
			}
		}
		f.ok(w, nil)
	case "scrobble":
		kind := "play"
		if q.Get("submission") == "false" {
			kind = "now"
		}
		for _, id := range q["id"] {
			f.scrobbles = append(f.scrobbles, kind+" "+id)
			if kind == "play" {
				f.plays[id]++
			}
		}
		f.ok(w, nil)
	case "getPlaylists":
		out := []map[string]any{}
		for _, l := range f.lists {
			if l.Owner == user {
				out = append(out, f.listView(l, false))
			}
		}
		f.ok(w, map[string]any{"playlists": map[string]any{"playlist": out}})
	case "getPlaylist":
		l := f.list(q.Get("id"))
		if l == nil || l.Owner != user {
			f.fail(w, 70, "Playlist not found")
			return
		}
		f.ok(w, map[string]any{"playlist": f.listView(l, true)})
	case "createPlaylist":
		if id := q.Get("playlistId"); id != "" {
			l := f.list(id)
			if l == nil || l.Owner != user {
				f.fail(w, 70, "Playlist not found")
				return
			}
			l.Songs, l.Changed = slices.Clone(q["songId"]), time.Now()
			if n := q.Get("name"); n != "" {
				l.Name = n
			}
			f.ok(w, map[string]any{"playlist": f.listView(l, true)})
			return
		}
		if q.Get("name") == "" {
			f.fail(w, 10, "required parameter name is missing")
			return
		}
		l := &fakeList{ID: "pl" + strconv.Itoa(f.nextList), Name: q.Get("name"), Owner: user, Songs: slices.Clone(q["songId"]), Changed: time.Now()}
		f.nextList++
		f.lists = append(f.lists, l)
		f.ok(w, map[string]any{"playlist": f.listView(l, true)})
	case "updatePlaylist":
		l := f.list(q.Get("playlistId"))
		if l == nil || l.Owner != user {
			f.fail(w, 70, "Playlist not found")
			return
		}
		var drop []int
		for _, s := range q["songIndexToRemove"] {
			if n, err := strconv.Atoi(s); err == nil {
				drop = append(drop, n)
			}
		}
		var keep []string
		for i, id := range l.Songs {
			if !slices.Contains(drop, i) {
				keep = append(keep, id)
			}
		}
		l.Songs = append(keep, q["songIdToAdd"]...)
		if q.Has("name") {
			l.Name = q.Get("name")
		}
		l.Changed = time.Now()
		f.ok(w, nil)
	case "deletePlaylist":
		i := slices.IndexFunc(f.lists, func(l *fakeList) bool { return l.ID == q.Get("id") && l.Owner == user })
		if i < 0 {
			f.fail(w, 70, "Playlist not found")
			return
		}
		f.lists = slices.Delete(f.lists, i, i+1)
		f.ok(w, nil)
	case "startScan":
		if !admin {
			f.fail(w, 50, "User is not authorized for the given operation")
			return
		}
		f.scanning = 2
		f.ok(w, map[string]any{"scanStatus": f.scanView()})
	case "getScanStatus":
		v := f.scanView()
		if f.scanning > 0 {
			f.scanning--
			if f.scanning == 0 {
				f.lastScan = time.Now()
			}
		}
		f.ok(w, map[string]any{"scanStatus": v})
	case "savePlayQueue":
		f.queue = q
		f.ok(w, nil)
	case "getPlayQueue":
		if f.queue == nil {
			f.ok(w, nil)
			return
		}
		entries := []map[string]any{}
		for _, id := range f.queue["id"] {
			if _, ok := f.songs[id]; ok {
				entries = append(entries, f.songView(id))
			}
		}
		pos, _ := strconv.ParseInt(f.queue.Get("position"), 10, 64)
		f.ok(w, map[string]any{"playQueue": map[string]any{"current": f.queue.Get("current"), "position": pos, "entry": entries, "username": user}})
	case "stream", "download":
		s, ok := f.songs[q.Get("id")]
		if !ok {
			f.fail(w, 70, "Song not found")
			return
		}
		ct := map[string]string{"mp3": "audio/mpeg", "flac": "audio/flac"}[s.Suffix]
		if method == "stream" && q.Get("format") == "mp3" {
			ct = "audio/mpeg"
			http.ServeContent(withType(w, ct), r, s.ID+".mp3", time.Unix(0, 0), bytes.NewReader(ToneMP3))
			return
		}
		if method == "download" {
			w.Header().Set("Content-Disposition", `attachment; filename="`+s.ID+"."+s.Suffix+`"`)
		}
		http.ServeContent(withType(w, ct), r, s.ID+"."+s.Suffix, time.Unix(0, 0), bytes.NewReader(s.Data))
	case "getCoverArt":
		c := color.RGBA{R: 160, G: 90, B: 40, A: 255}
		if strings.HasPrefix(q.Get("id"), "ar") {
			c = color.RGBA{R: 40, G: 110, B: 100, A: 255}
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(CoverPNG(c))
	default:
		http.NotFound(w, r)
	}
}

func withType(w http.ResponseWriter, ct string) http.ResponseWriter {
	w.Header().Set("Content-Type", ct)
	return w
}

func (f *FakeSubsonic) scanView() map[string]any {
	v := map[string]any{"scanning": f.scanning > 0, "count": len(f.songs), "folderCount": len(f.albums), "lastScan": f.lastScan.UTC().Format(time.RFC3339)}
	return v
}

func (f *FakeSubsonic) albumList(w http.ResponseWriter, q url.Values) {
	all := slices.Clone(f.albums)
	plays := func(a *fakeAlbum) int64 {
		var n int64
		for _, s := range a.Songs {
			n += f.plays[s]
		}
		return n
	}
	switch q.Get("type") {
	case "newest":
		slices.SortStableFunc(all, func(a, b *fakeAlbum) int { return b.Created - a.Created })
	case "frequent", "recent":
		all = slices.DeleteFunc(all, func(a *fakeAlbum) bool { return plays(a) == 0 })
		slices.SortStableFunc(all, func(a, b *fakeAlbum) int { return int(plays(b) - plays(a)) })
	case "starred":
		all = slices.DeleteFunc(all, func(a *fakeAlbum) bool { _, ok := f.starred[a.ID]; return !ok })
	case "alphabeticalByArtist":
		slices.SortStableFunc(all, func(a, b *fakeAlbum) int {
			return strings.Compare(f.artist(a.ArtistID).Name, f.artist(b.ArtistID).Name)
		})
	case "byYear":
		slices.SortStableFunc(all, func(a, b *fakeAlbum) int { return b.Year - a.Year })
	case "random":
		slices.Reverse(all)
	default:
		slices.SortStableFunc(all, func(a, b *fakeAlbum) int { return strings.Compare(a.Name, b.Name) })
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	size, _ := strconv.Atoi(q.Get("size"))
	offset = min(max(offset, 0), len(all))
	all = all[offset:]
	if size > 0 && size < len(all) {
		all = all[:size]
	}
	out := []map[string]any{}
	for _, a := range all {
		out = append(out, f.albumView(a))
	}
	f.ok(w, map[string]any{"albumList2": map[string]any{"album": out}})
}
