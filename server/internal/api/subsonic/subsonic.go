// Package subsonic fala a API Subsonic/OpenSubsonic do jeito que o
// Navidrome fala, pros clientes de música (Symfonium, Feishin, DSub...)
// funcionarem sem mudar nada. Só consumo: navegar, buscar, tocar, estrela,
// nota, scrobble, playlists, fila e marcadores.
package subsonic

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/deps"
)

// API é o handler de /rest/.
type API struct {
	d      *deps.Deps
	routes map[string]route

	npMu sync.Mutex
	np   map[string]*playing // usuário+cliente → tocando agora
}

// call é uma requisição já autenticada.
type call struct {
	w      http.ResponseWriter
	r      *http.Request
	ctx    context.Context
	p      params
	user   *auth.User
	client string
}

// handler devolve a resposta; nil sem erro = já escreveu (stream, imagem).
type handler func(c *call) (*resp, error)

type route struct {
	h      handler
	noAuth bool
	admin  bool
}

// New monta o handler da API.
func New(d *deps.Deps) http.Handler {
	a := &API{d: d, np: map[string]*playing{}}
	a.routes = map[string]route{}
	add := func(name string, h handler) { a.routes[strings.ToLower(name)] = route{h: h} }
	admin := func(name string, h handler) { a.routes[strings.ToLower(name)] = route{h: h, admin: true} }
	a.routes["getopensubsonicextensions"] = route{h: a.extensions, noAuth: true}

	// sistema
	add("ping", a.ping)
	add("getLicense", a.license)
	add("tokenInfo", a.tokenInfo)
	add("getUser", a.getUser)
	add("getUsers", a.getUsers)
	add("getScanStatus", a.scanStatus)
	admin("startScan", a.startScan)

	// navegação
	add("getMusicFolders", a.musicFolders)
	add("getIndexes", a.getIndexes)
	add("getArtists", a.getArtists)
	add("getMusicDirectory", a.musicDirectory)
	add("getGenres", a.getGenres)
	add("getArtist", a.getArtist)
	add("getAlbum", a.getAlbum)
	add("getSong", a.getSong)
	add("getArtistInfo", a.artistInfo)
	add("getArtistInfo2", a.artistInfo2)
	add("getAlbumInfo", a.albumInfo)
	add("getAlbumInfo2", a.albumInfo)
	add("getSimilarSongs", a.similarSongs)
	add("getSimilarSongs2", a.similarSongs2)
	add("getTopSongs", a.topSongs)

	// listas
	add("getAlbumList", a.albumList)
	add("getAlbumList2", a.albumList2)
	add("getRandomSongs", a.randomSongs)
	add("getSongsByGenre", a.songsByGenre)
	add("getNowPlaying", a.nowPlaying)
	add("getStarred", a.starred)
	add("getStarred2", a.starred2)
	add("search2", a.search2)
	add("search3", a.search3)

	// anotações
	add("star", a.star)
	add("unstar", a.unstar)
	add("setRating", a.setRating)
	add("scrobble", a.scrobble)

	// playlists
	add("getPlaylists", a.getPlaylists)
	add("getPlaylist", a.getPlaylist)
	add("createPlaylist", a.createPlaylist)
	add("updatePlaylist", a.updatePlaylist)
	add("deletePlaylist", a.deletePlaylist)

	// fila e marcadores
	add("getPlayQueue", a.getPlayQueue)
	add("savePlayQueue", a.savePlayQueue)
	add("getPlayQueueByIndex", a.getPlayQueueByIndex)
	add("savePlayQueueByIndex", a.savePlayQueueByIndex)
	add("getBookmarks", a.getBookmarks)
	add("createBookmark", a.createBookmark)
	add("deleteBookmark", a.deleteBookmark)

	// mídia
	add("stream", a.stream)
	add("download", a.download)
	add("getCoverArt", a.coverArt)
	add("getAvatar", a.avatar)
	add("getLyrics", a.getLyrics)
	add("getLyricsBySongId", a.lyricsBySongID)

	// desligados: listas vazias
	add("getInternetRadioStations", a.empty(func(r *resp) { r.Radios = &radios{} }))
	add("getPodcasts", a.empty(func(r *resp) { r.Podcasts = &podcasts{} }))
	add("getNewestPodcasts", a.empty(func(r *resp) { r.NewestPodcast = &newestPodcast{} }))
	add("getShares", a.empty(func(r *resp) { r.Shares = &shares{} }))

	// o Navidrome responde 501 pro que está desligado e 410 pro que nunca vai ter
	for _, m := range []string{"jukeboxControl", "createShare", "updateShare", "deleteShare",
		"refreshPodcasts", "createPodcastChannel", "deletePodcastChannel", "deletePodcastEpisode",
		"downloadPodcastEpisode", "createUser", "updateUser", "deleteUser", "changePassword",
		"createInternetRadioStation", "updateInternetRadioStation", "deleteInternetRadioStation"} {
		add(m, status(http.StatusNotImplemented, "This endpoint is not implemented, but may be in future releases"))
	}
	for _, m := range []string{"search", "getChatMessages", "addChatMessage", "getVideos", "getVideoInfo", "getCaptions", "hls"} {
		add(m, status(http.StatusGone, "This endpoint will not be implemented"))
	}
	return a
}

func status(code int, msg string) handler {
	return func(c *call) (*resp, error) {
		c.w.Header().Set("Cache-Control", "no-cache")
		c.w.WriteHeader(code)
		c.w.Write([]byte(msg))
		return nil, nil
	}
}

func (a *API) empty(fill func(*resp)) handler {
	return func(c *call) (*resp, error) {
		res := a.newResp()
		fill(res)
		return res, nil
	}
}

// method tira "/rest/" e ".view" do caminho.
func method(path string) string {
	p := strings.ToLower(strings.Trim(path, "/"))
	p = strings.TrimPrefix(p, "rest/")
	return strings.TrimSuffix(p, ".view")
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt, ok := a.routes[method(r.URL.Path)]
	if !ok {
		// igual ao roteador do Navidrome: método desconhecido é 404
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseForm(); err != nil {
		r.Form = r.URL.Query()
		a.sendErr(w, r, fail(errGeneric, "%s", err.Error()))
		return
	}
	c := &call{w: w, r: r, ctx: r.Context(), p: params(r.Form)}
	if !rt.noAuth {
		u, err := a.authenticate(c)
		if err != nil {
			a.sendErr(w, r, err)
			return
		}
		c.user, c.client = u, c.p.str("c")
		if rt.admin && !u.Admin {
			a.sendErr(w, r, fail(errNotAllowed, ""))
			return
		}
	}
	res, err := rt.h(c)
	switch {
	case err != nil:
		a.sendErr(w, r, err)
	case res != nil && r.Context().Err() == nil:
		a.send(w, r, res)
	}
}

func (a *API) ping(c *call) (*resp, error) { return a.newResp(), nil }

func (a *API) license(c *call) (*resp, error) {
	res := a.newResp()
	res.License = &license{Valid: true}
	return res, nil
}

func (a *API) tokenInfo(c *call) (*resp, error) {
	res := a.newResp()
	res.TokenInfo = &tokenInfo{Username: c.user.Name}
	return res, nil
}

// extensions lista só o que está implementado de verdade.
func (a *API) extensions(c *call) (*resp, error) {
	res := a.newResp()
	ext := []extension{
		{Name: "formPost", Versions: []int{1}},
		{Name: "apiKeyAuthentication", Versions: []int{1}},
		{Name: "songLyrics", Versions: []int{1}},
		{Name: "transcodeOffset", Versions: []int{1}},
		{Name: "indexBasedQueue", Versions: []int{1}},
		{Name: "topSongsByArtistId", Versions: []int{1}},
	}
	res.Extensions = &ext
	return res, nil
}

func (a *API) scanStatus(c *call) (*resp, error) {
	st := a.d.Library.Status()
	ss := &scanStatus{Scanning: st.Running, Count: int64(st.Count), Error: st.Error}
	if !st.Running {
		// fora do scan conta o acervo de música
		a.d.Store.DB.QueryRowContext(c.ctx, `SELECT COUNT(*) FROM items WHERE kind = 'track'`).Scan(&ss.Count)
	}
	a.d.Store.DB.QueryRowContext(c.ctx, `SELECT COUNT(*) FROM items WHERE kind = 'album'`).Scan(&ss.FolderCount)
	last := st.Finished
	if st.Running || last.IsZero() {
		last = st.Started
	}
	ss.LastScan = tp(last)
	res := a.newResp()
	res.ScanStatus = ss
	return res, nil
}

func (a *API) startScan(c *call) (*resp, error) {
	before := a.d.Library.Status().Started
	a.d.Library.Trigger()
	// espera o scan aparecer (ou acabar) por até 1s, como o Navidrome
	for range 20 {
		if st := a.d.Library.Status(); st.Running || st.Started != before {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return a.scanStatus(c)
}
