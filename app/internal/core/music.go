package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/subsonic"
)

// Músicas: cada perfil vincula a própria conta de um servidor Subsonic
// (Navidrome ou arkflix-server). Fica em <pasta do perfil>/subsonic.json
// (0600) só o par salgado do Subsonic (ou a chave de API), nunca a senha.
// Áudio e capas passam pelo proxy em 127.0.0.1, que põe a autenticação.

var errNoMusic = errors.New("vincule uma conta de músicas (Navidrome/Subsonic) ao perfil")

type musicLink struct {
	subsonic.Account
	Linked int64 `json:"linked"`
}

type musicState struct {
	mu    sync.Mutex
	links map[string]*musicLink // por perfil (nil = sem conta)
	proxy *subsonic.Proxy

	idxMu sync.Mutex
	idx   *localIndex // cópias baixadas (carregado na primeira vez)
}

func (s *Service) musicFile(profile string) string {
	return filepath.Join(s.profiles.DataDir(profile), "subsonic.json")
}

// musicLinkOf lê (uma vez) a conta de músicas do perfil.
func (s *Service) musicLinkOf(profile string) *musicLink {
	s.music.mu.Lock()
	defer s.music.mu.Unlock()
	if l, ok := s.music.links[profile]; ok {
		return l
	}
	var l *musicLink
	if raw, err := os.ReadFile(s.musicFile(profile)); err == nil {
		var m musicLink
		if json.Unmarshal(raw, &m) == nil && m.Base != "" && (m.Token != "" || m.APIKey != "") {
			l = &m
		}
	}
	s.music.links[profile] = l
	return l
}

func (s *Service) musicAPI() (*subsonic.API, error) {
	l := s.musicLinkOf(s.cur().id)
	if l == nil {
		return nil, errNoMusic
	}
	return &subsonic.API{Account: l.Account}, nil
}

func musicCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// MusicInfo é o estado da conta de músicas de um perfil.
type MusicInfo struct {
	Connected bool   `json:"connected"`
	Base      string `json:"base"`
	User      string `json:"user"`
	Server    string `json:"server"` // navidrome, arkflix-server...
	Version   string `json:"version"`
}

func musicInfo(l *musicLink) MusicInfo {
	if l == nil {
		return MusicInfo{}
	}
	return MusicInfo{Connected: true, Base: l.Base, User: l.User, Server: l.Server, Version: l.ServerVersion}
}

// MusicStatus diz se o perfil em uso tem conta de músicas.
func (s *Service) MusicStatus() MusicInfo { return musicInfo(s.musicLinkOf(s.cur().id)) }

// ConnectMusic vincula uma conta de músicas a um perfil. A senha só confere
// o login e vira o par salgado do Subsonic. Só adulto vincula (inclusive
// pro perfil infantil). key: chave de API do OpenSubsonic (opcional, no
// lugar da senha).
func (s *Service) ConnectMusic(profile, host, user, pass, key string) (MusicInfo, error) {
	if err := s.adult(); err != nil {
		return MusicInfo{}, err
	}
	if _, err := s.profiles.Get(profile); err != nil {
		return MusicInfo{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	acc, err := subsonic.Login(ctx, host, user, pass, key)
	if err != nil {
		return MusicInfo{}, err
	}
	l := &musicLink{Account: acc, Linked: time.Now().Unix()}
	if err := writePrivate(s.musicFile(profile), l); err != nil {
		return MusicInfo{}, err
	}
	s.music.mu.Lock()
	s.music.links[profile] = l
	s.music.mu.Unlock()
	s.arkDetect(acc.Base)
	return musicInfo(l), nil
}

// DisconnectMusic desvincula a conta de músicas do perfil (o Subsonic não
// tem logout: o par guardado só é apagado). As cópias baixadas ficam.
func (s *Service) DisconnectMusic(profile string) error {
	if err := s.adult(); err != nil {
		return err
	}
	if err := os.Remove(s.musicFile(profile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s.music.mu.Lock()
	s.music.links[profile] = nil
	s.music.mu.Unlock()
	return nil
}

func (s *Service) forgetMusic(profile string) {
	s.music.mu.Lock()
	delete(s.music.links, profile)
	s.music.mu.Unlock()
}

// MusicServer é o que a interface sabe do servidor de músicas do perfil.
type MusicServer struct {
	MusicInfo
	// Admin: a conta é admin no servidor e o perfil é adulto (aba de admin).
	Admin bool `json:"admin"`
	// Ark: o servidor é um arkflix-server; ArkPanel: o painel do Servidor
	// arkflix já está liberado pra este perfil no mesmo endereço (usuários e
	// envio de arquivos ficam lá). Upload: dá pra enviar músicas por ele.
	Ark      bool   `json:"ark"`
	ArkPanel bool   `json:"arkPanel"`
	Upload   bool   `json:"upload"`
	Error    string `json:"error,omitempty"`
}

func sameBase(a, b string) bool { return strings.TrimRight(a, "/") == strings.TrimRight(b, "/") }

// MusicServerInfo pergunta ao servidor quem é a conta (admin) e se é um
// arkflix-server (pelo mesmo /arkflix/api/info que os filmes e o RomM usam).
func (s *Service) MusicServerInfo() MusicServer {
	l := s.musicLinkOf(s.cur().id)
	out := MusicServer{MusicInfo: musicInfo(l)}
	if l == nil {
		return out
	}
	if _, ok := s.arkProbe(l.Base, false); ok {
		out.Ark = true
		if c, err := s.arkConn(); err == nil && sameBase(c.client.Base, l.Base) && s.adult() == nil && c.info.Has("admin") {
			ctx, cancel := arkCtx()
			me, err := c.client.Me(ctx)
			cancel()
			if err == nil && me.Admin {
				out.ArkPanel = true
				out.Upload = c.info.Writable && c.info.Has("upload")
			}
		}
	}
	ctx, cancel := musicCtx()
	defer cancel()
	u, err := (&subsonic.API{Account: l.Account}).GetUser(ctx, l.User)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Admin = u.AdminRole && s.adult() == nil
	return out
}

// Visões que a interface usa (capa já com o endereço do proxy).

// MusicTrack é uma faixa.
type MusicTrack struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Album       string `json:"album"`
	AlbumID     string `json:"albumId"`
	Artist      string `json:"artist"`
	ArtistID    string `json:"artistId"`
	Track       int    `json:"track"`
	Disc        int    `json:"disc"`
	Year        int    `json:"year"`
	Duration    int    `json:"duration"`
	Cover       string `json:"cover"`
	Starred     bool   `json:"starred"`
	Suffix      string `json:"suffix"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	Local       bool   `json:"local"` // tem cópia neste PC
}

// MusicAlbum é um álbum.
type MusicAlbum struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	ArtistID string `json:"artistId"`
	Year     int    `json:"year"`
	Genre    string `json:"genre"`
	Songs    int    `json:"songs"`
	Duration int    `json:"duration"`
	Cover    string `json:"cover"`
	Starred  bool   `json:"starred"`
	Local    bool   `json:"local"`
}

// MusicArtist é um artista.
type MusicArtist struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Albums  int    `json:"albums"`
	Cover   string `json:"cover"`
	Starred bool   `json:"starred"`
}

// MusicPlaylist é uma playlist (Tracks só na página dela).
type MusicPlaylist struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Owner    string       `json:"owner"`
	Mine     bool         `json:"mine"`
	Songs    int          `json:"songs"`
	Duration int          `json:"duration"`
	Cover    string       `json:"cover"`
	Changed  string       `json:"changed"`
	Local    bool         `json:"local"`
	Tracks   []MusicTrack `json:"tracks"`
}

func (s *Service) musicProxy() string {
	base, err := s.music.proxy.URL()
	if err != nil {
		return ""
	}
	return base
}

func cover(base, id string, size int) string {
	if base == "" || id == "" {
		return ""
	}
	return base + "/rest/getCoverArt?" + url.Values{"id": {id}, "size": {strconv.Itoa(size)}}.Encode()
}

func (s *Service) trackView(base, server string, t subsonic.Song) MusicTrack {
	v := MusicTrack{ID: t.ID, Title: t.Title, Album: t.Album, AlbumID: t.AlbumID, Artist: t.Artist, ArtistID: t.ArtistID,
		Track: t.Track, Disc: t.DiscNumber, Year: t.Year, Duration: t.Duration, Cover: cover(base, t.CoverArt, 300),
		Starred: t.Starred != "", Suffix: t.Suffix, ContentType: t.ContentType, Size: t.Size}
	_, v.Local = s.localTrack(server, t.ID)
	return v
}

func (s *Service) tracksView(server string, list []subsonic.Song) []MusicTrack {
	base := s.musicProxy()
	out := make([]MusicTrack, 0, len(list))
	for _, t := range list {
		out = append(out, s.trackView(base, server, t))
	}
	return out
}

func (s *Service) albumsView(server string, list []subsonic.Album) []MusicAlbum {
	base := s.musicProxy()
	out := make([]MusicAlbum, 0, len(list))
	for _, a := range list {
		out = append(out, MusicAlbum{ID: a.ID, Name: a.Name, Artist: a.Artist, ArtistID: a.ArtistID, Year: a.Year, Genre: a.Genre,
			Songs: a.SongCount, Duration: a.Duration, Cover: cover(base, a.CoverArt, 360), Starred: a.Starred != "",
			Local: s.localSet(server, "album", a.ID) != nil})
	}
	return out
}

func (s *Service) artistsView(list []subsonic.Artist) []MusicArtist {
	base := s.musicProxy()
	out := make([]MusicArtist, 0, len(list))
	for _, a := range list {
		out = append(out, MusicArtist{ID: a.ID, Name: a.Name, Albums: a.AlbumCount, Cover: cover(base, a.CoverArt, 300), Starred: a.Starred != ""})
	}
	return out
}

func (s *Service) playlistView(c *subsonic.API, p subsonic.Playlist) MusicPlaylist {
	return MusicPlaylist{ID: p.ID, Name: p.Name, Owner: p.Owner, Mine: strings.EqualFold(p.Owner, c.User) || p.Owner == "",
		Songs: p.SongCount, Duration: p.Duration, Cover: cover(s.musicProxy(), p.CoverArt, 360), Changed: p.Changed,
		Local: s.localSet(c.Base, "playlist", p.ID) != nil, Tracks: s.tracksView(c.Base, p.Entry)}
}

// MusicHome é o início da seção de músicas.
type MusicHome struct {
	Newest   []MusicAlbum `json:"newest"`
	Frequent []MusicAlbum `json:"frequent"`
	Recent   []MusicAlbum `json:"recent"`
	Random   []MusicAlbum `json:"random"`
	Starred  []MusicAlbum `json:"starred"`
}

// MusicHome busca as fileiras do início (em paralelo; uma que falhe fica
// vazia; todas falhando é erro).
func (s *Service) MusicHome() (MusicHome, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicHome{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
		out   MusicHome
	)
	for kind, dst := range map[string]*[]MusicAlbum{"newest": &out.Newest, "frequent": &out.Frequent, "recent": &out.Recent, "random": &out.Random, "starred": &out.Starred} {
		wg.Go(func() {
			list, err := c.AlbumList2(ctx, kind, 20, 0)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				first = cmpErr(first, err)
				*dst = []MusicAlbum{}
				return
			}
			*dst = s.albumsView(c.Base, list)
		})
	}
	wg.Wait()
	if first != nil && len(out.Newest)+len(out.Frequent)+len(out.Recent)+len(out.Random)+len(out.Starred) == 0 {
		return out, first
	}
	return out, nil
}

// MusicIndex é uma letra do índice de artistas.
type MusicIndex struct {
	Name    string        `json:"name"`
	Artists []MusicArtist `json:"artists"`
}

// MusicArtists é o índice de artistas por letra.
func (s *Service) MusicArtists() ([]MusicIndex, error) {
	c, err := s.musicAPI()
	if err != nil {
		return nil, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	idx, err := c.Artists(ctx)
	out := make([]MusicIndex, 0, len(idx))
	for _, i := range idx {
		out = append(out, MusicIndex{Name: i.Name, Artists: s.artistsView(i.Artists)})
	}
	return out, err
}

var musicSorts = map[string]string{"newest": "newest", "name": "alphabeticalByName", "artist": "alphabeticalByArtist",
	"year": "byYear", "frequent": "frequent", "recent": "recent", "random": "random", "starred": "starred"}

// MusicAlbumsPage é uma página de álbuns (o Subsonic não dá o total: More
// diz se pode ter mais).
type MusicAlbumsPage struct {
	Albums []MusicAlbum `json:"albums"`
	More   bool         `json:"more"`
}

// MusicAlbums lista os álbuns: sort é newest, name, artist, year, frequent,
// recent, random ou starred.
func (s *Service) MusicAlbums(sort string, start, limit int) (MusicAlbumsPage, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicAlbumsPage{}, err
	}
	kind, ok := musicSorts[sort]
	if !ok {
		return MusicAlbumsPage{}, errors.New("ordem desconhecida")
	}
	limit = min(max(limit, 1), 500)
	ctx, cancel := musicCtx()
	defer cancel()
	list, err := c.AlbumList2(ctx, kind, limit, max(0, start))
	return MusicAlbumsPage{Albums: s.albumsView(c.Base, list), More: len(list) == limit}, err
}

// MusicArtistPage é a página de um artista.
type MusicArtistPage struct {
	Artist MusicArtist  `json:"artist"`
	Albums []MusicAlbum `json:"albums"`
	Top    []MusicTrack `json:"top"`
}

// MusicArtist busca o artista, os álbuns e as mais tocadas.
func (s *Service) MusicArtist(id string) (MusicArtistPage, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicArtistPage{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	ar, albums, err := c.Artist(ctx, id)
	if err != nil {
		return MusicArtistPage{}, err
	}
	out := MusicArtistPage{Artist: s.artistsView([]subsonic.Artist{ar})[0], Albums: s.albumsView(c.Base, albums), Top: []MusicTrack{}}
	if top, err := c.TopSongs(ctx, ar.Name, 10); err == nil {
		out.Top = s.tracksView(c.Base, top)
	}
	return out, nil
}

// MusicAlbumPage é a página de um álbum.
type MusicAlbumPage struct {
	Album  MusicAlbum   `json:"album"`
	Tracks []MusicTrack `json:"tracks"`
	// Offline: o servidor não respondeu e veio da cópia baixada.
	Offline bool `json:"offline"`
}

// MusicAlbum busca o álbum com as faixas (sem servidor, a cópia baixada).
func (s *Service) MusicAlbum(id string) (MusicAlbumPage, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicAlbumPage{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	al, songs, err := c.Album(ctx, id)
	if err != nil {
		if set := s.localSet(c.Base, "album", id); set != nil && !errors.Is(err, subsonic.ErrAuth) && !isNotFound(err) {
			return MusicAlbumPage{Album: set.album(true), Tracks: s.localTracks(c.Base, set), Offline: true}, nil
		}
		return MusicAlbumPage{}, err
	}
	return MusicAlbumPage{Album: s.albumsView(c.Base, []subsonic.Album{al})[0], Tracks: s.tracksView(c.Base, songs)}, nil
}

func isNotFound(err error) bool {
	var e *subsonic.Error
	return errors.As(err, &e) && e.Code == 70
}

// MusicFound é o que a busca e as favoritas devolvem.
type MusicFound struct {
	Artists []MusicArtist `json:"artists"`
	Albums  []MusicAlbum  `json:"albums"`
	Tracks  []MusicTrack  `json:"tracks"`
}

func (s *Service) found(server string, f subsonic.Found) MusicFound {
	return MusicFound{Artists: s.artistsView(f.Artists), Albums: s.albumsView(server, f.Albums), Tracks: s.tracksView(server, f.Songs)}
}

// MusicSearch busca artistas, álbuns e faixas.
func (s *Service) MusicSearch(text string) (MusicFound, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return MusicFound{Artists: []MusicArtist{}, Albums: []MusicAlbum{}, Tracks: []MusicTrack{}}, nil
	}
	c, err := s.musicAPI()
	if err != nil {
		return MusicFound{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	f, err := c.Search(ctx, text, 20)
	return s.found(c.Base, f), err
}

// MusicStarred são as favoritas.
func (s *Service) MusicStarred() (MusicFound, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicFound{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	f, err := c.Starred(ctx)
	return s.found(c.Base, f), err
}

// MusicStar marca ou desmarca como favorita (kind: track, album ou artist).
func (s *Service) MusicStar(kind, id string, on bool) error {
	c, err := s.musicAPI()
	if err != nil {
		return err
	}
	k := map[string]string{"track": "song", "album": "album", "artist": "artist"}[kind]
	if k == "" || id == "" {
		return errors.New("item inválido")
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.Star(ctx, k, id, on)
}

// MusicStream é como a interface toca uma faixa: URL no proxy local (o
// arquivo original; Fallback pede mp3 ao servidor se o sistema não abrir o
// formato). Local = toca da cópia deste PC (funciona sem internet).
type MusicStream struct {
	URL      string `json:"url"`
	Fallback string `json:"fallback"`
	Local    bool   `json:"local"`
}

// MusicPlay prepara a reprodução de uma faixa.
func (s *Service) MusicPlay(id string) (MusicStream, error) {
	l := s.musicLinkOf(s.cur().id)
	if l == nil {
		return MusicStream{}, errNoMusic
	}
	if id == "" || strings.ContainsAny(id, "/\\?#") {
		return MusicStream{}, errors.New("faixa inválida")
	}
	base, err := s.music.proxy.URL()
	if err != nil {
		return MusicStream{}, err
	}
	q := url.Values{"id": {id}}
	out := MusicStream{URL: base + "/rest/stream?" + q.Encode() + "&format=raw", Fallback: base + "/rest/stream?" + q.Encode() + "&format=mp3&maxBitRate=320"}
	if _, ok := s.localTrack(l.Base, id); ok {
		out.URL, out.Local = base+"/local/"+url.PathEscape(id), true
	}
	return out, nil
}

// MusicScrobble avisa o servidor: submission false = tocando agora (no
// começo); true = conta a execução (depois de metade ou 4 min).
func (s *Service) MusicScrobble(id string, submission bool) error {
	c, err := s.musicAPI()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.Scrobble(ctx, id, submission, time.Now())
}

// MusicSaveQueue guarda a fila no servidor (ao parar e ao sair).
func (s *Service) MusicSaveQueue(ids []string, current string, position float64) error {
	c, err := s.musicAPI()
	if err != nil {
		return err
	}
	if len(ids) > 1000 {
		ids = ids[:1000]
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.SaveQueue(ctx, ids, current, time.Duration(max(0, position)*float64(time.Second)))
}

// MusicQueue é a fila salva no servidor (pra continuar).
type MusicQueue struct {
	Tracks   []MusicTrack `json:"tracks"`
	Current  int          `json:"current"`
	Position float64      `json:"position"`
}

// MusicResume busca a fila salva.
func (s *Service) MusicResume() (MusicQueue, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicQueue{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	q, err := c.Queue(ctx)
	if err != nil {
		return MusicQueue{}, err
	}
	out := MusicQueue{Tracks: s.tracksView(c.Base, q.Entry), Position: float64(q.Position) / 1000}
	out.Current = max(0, slices.IndexFunc(q.Entry, func(t subsonic.Song) bool { return t.ID == q.Current }))
	return out, nil
}

// Playlists.

// MusicPlaylists lista as playlists.
func (s *Service) MusicPlaylists() ([]MusicPlaylist, error) {
	c, err := s.musicAPI()
	if err != nil {
		return nil, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	list, err := c.Playlists(ctx)
	out := make([]MusicPlaylist, 0, len(list))
	for _, p := range list {
		out = append(out, s.playlistView(c, p))
	}
	return out, err
}

// MusicPlaylist é uma playlist com as faixas (sem servidor, a cópia baixada).
func (s *Service) MusicPlaylist(id string) (MusicPlaylist, error) {
	c, err := s.musicAPI()
	if err != nil {
		return MusicPlaylist{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	p, err := c.Playlist(ctx, id)
	if err != nil {
		if set := s.localSet(c.Base, "playlist", id); set != nil && !errors.Is(err, subsonic.ErrAuth) && !isNotFound(err) {
			return MusicPlaylist{ID: id, Name: set.Name, Songs: len(set.Tracks), Local: true, Tracks: s.localTracks(c.Base, set)}, nil
		}
		return MusicPlaylist{}, err
	}
	return s.playlistView(c, p), nil
}

// musicWrite: playlist só muda no perfil adulto (o infantil só ouve).
func (s *Service) musicWrite() (*subsonic.API, error) {
	if err := s.adult(); err != nil {
		return nil, err
	}
	return s.musicAPI()
}

// MusicCreatePlaylist cria uma playlist (com faixas ou vazia).
func (s *Service) MusicCreatePlaylist(name string, tracks []string) (MusicPlaylist, error) {
	c, err := s.musicWrite()
	if err != nil {
		return MusicPlaylist{}, err
	}
	if name = strings.TrimSpace(name); name == "" {
		return MusicPlaylist{}, errors.New("dê um nome à playlist")
	}
	ctx, cancel := musicCtx()
	defer cancel()
	p, err := c.CreatePlaylist(ctx, name, tracks)
	if err != nil {
		return MusicPlaylist{}, err
	}
	if p.ID == "" { // Subsonic antigo não devolve a playlist: acha pelo nome
		list, _ := c.Playlists(ctx)
		for _, x := range list {
			if x.Name == name {
				p = x
			}
		}
	}
	return s.playlistView(c, p), nil
}

// MusicRenamePlaylist troca o nome.
func (s *Service) MusicRenamePlaylist(id, name string) error {
	c, err := s.musicWrite()
	if err != nil {
		return err
	}
	if name = strings.TrimSpace(name); name == "" {
		return errors.New("dê um nome à playlist")
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.UpdatePlaylist(ctx, id, subsonic.PlaylistChange{Name: &name})
}

// MusicDeletePlaylist apaga a playlist (as faixas continuam no servidor).
func (s *Service) MusicDeletePlaylist(id string) error {
	c, err := s.musicWrite()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.DeletePlaylist(ctx, id)
}

// MusicAddToPlaylist põe faixas no fim da playlist.
func (s *Service) MusicAddToPlaylist(id string, tracks []string) error {
	c, err := s.musicWrite()
	if err != nil {
		return err
	}
	if len(tracks) == 0 {
		return errors.New("nenhuma faixa pra pôr")
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.UpdatePlaylist(ctx, id, subsonic.PlaylistChange{Add: tracks})
}

// MusicAddAlbumToPlaylist põe as faixas de um álbum no fim da playlist.
func (s *Service) MusicAddAlbumToPlaylist(id, album string) error {
	c, err := s.musicWrite()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	_, songs, err := c.Album(ctx, album)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(songs))
	for _, t := range songs {
		ids = append(ids, t.ID)
	}
	if len(ids) == 0 {
		return errors.New("álbum vazio")
	}
	return c.UpdatePlaylist(ctx, id, subsonic.PlaylistChange{Add: ids})
}

// MusicRemoveFromPlaylist tira faixas pela posição (de 0).
func (s *Service) MusicRemoveFromPlaylist(id string, positions []int) error {
	c, err := s.musicWrite()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.UpdatePlaylist(ctx, id, subsonic.PlaylistChange{Remove: positions})
}

// MusicReorderPlaylist põe as faixas na ordem dada (a lista inteira).
func (s *Service) MusicReorderPlaylist(id string, tracks []string) error {
	c, err := s.musicWrite()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	return c.SetPlaylistSongs(ctx, id, tracks)
}

// Administração (conta admin, perfil adulto).

// MusicAdminInfo é o estado do servidor de músicas pra quem administra.
type MusicAdminInfo struct {
	Scan  subsonic.ScanStatus `json:"scan"`
	Users []subsonic.User     `json:"users"`
	Error string              `json:"error,omitempty"` // usuários não vieram
}

func (s *Service) musicAdmin() (*subsonic.API, error) {
	if err := s.adult(); err != nil {
		return nil, err
	}
	return s.musicAPI()
}

// MusicAdmin busca o scan e os usuários (só leitura: o Navidrome cuida dos
// usuários na interface dele).
func (s *Service) MusicAdmin() (MusicAdminInfo, error) {
	c, err := s.musicAdmin()
	if err != nil {
		return MusicAdminInfo{}, err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	st, err := c.Scan(ctx)
	if err != nil {
		return MusicAdminInfo{}, err
	}
	out := MusicAdminInfo{Scan: st, Users: []subsonic.User{}}
	if users, err := c.Users(ctx); err != nil {
		out.Error = err.Error()
	} else if users != nil {
		out.Users = users
	}
	return out, nil
}

// musicScanPoll: de quanto em quanto tempo o scan é conferido.
var musicScanPoll = 2 * time.Second

// MusicScan começa o scan e acompanha até o fim (tarefa "musicscan").
func (s *Service) MusicScan() error {
	c, err := s.musicAdmin()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	if _, err := c.StartScan(ctx); err != nil {
		return err
	}
	if !s.runTaskCtx("musicscan", "Escaneando as músicas", func(ctx context.Context, _ func(float64)) (string, error) {
		deadline := time.Now().Add(2 * time.Hour)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(musicScanPoll):
			}
			sctx, scancel := context.WithTimeout(ctx, 30*time.Second)
			st, err := c.Scan(sctx)
			scancel()
			if err != nil {
				return "", err
			}
			if !st.Scanning {
				return strconv.Itoa(int(st.Count)) + " faixas no servidor", nil
			}
		}
		return "", errors.New("o scan não terminou em 2 horas")
	}) {
		return errors.New("as músicas já estão sendo escaneadas")
	}
	return nil
}

func newMusicState(s *Service) *musicState {
	m := &musicState{links: map[string]*musicLink{}}
	m.proxy = &subsonic.Proxy{
		Account: func() (subsonic.Account, bool) {
			l := s.musicLinkOf(s.cur().id)
			if l == nil {
				return subsonic.Account{}, false
			}
			return l.Account, true
		},
		Local: func(id string) (string, bool) {
			l := s.musicLinkOf(s.cur().id)
			if l == nil {
				return "", false
			}
			return s.localTrack(l.Base, id)
		},
	}
	return m
}
