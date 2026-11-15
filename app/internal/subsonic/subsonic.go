// Package subsonic fala a API Subsonic/OpenSubsonic (Navidrome e o
// arkflix-server): login, artistas, álbuns, faixas, busca, favoritas,
// playlists, scrobble, fila e scan. A senha só é usada no login: o que fica
// guardado é o par salgado do Subsonic (t = md5(senha + s), s) ou, se o
// servidor der, uma chave de API. Depois do login nunca vai p= pro servidor.
package subsonic

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/jellyfin"
)

// Client é o nome do app nos servidores (parâmetro c=).
const Client = "arkflix"

// Version é a versão do protocolo pedida (a do Navidrome).
const Version = "1.16.1"

var (
	ErrAuth = errors.New("o servidor de músicas recusou o acesso: vincule a conta de novo no perfil")
	ErrHost = jellyfin.ErrHost
)

// Account é o que fica guardado depois do login (nunca a senha).
type Account struct {
	Base string `json:"base"`
	User string `json:"user"`
	// Token e Salt: o par do Subsonic (t = md5(senha + s)).
	Token string `json:"token,omitempty"`
	Salt  string `json:"salt,omitempty"`
	// APIKey: chave do OpenSubsonic (apiKeyAuthentication), quando há.
	APIKey string `json:"apiKey,omitempty"`
	// Server: nome do servidor (navidrome, arkflix-server...) e a versão.
	Server        string   `json:"server"`
	ServerVersion string   `json:"serverVersion"`
	Extensions    []string `json:"extensions"`
}

// Has diz se o servidor anuncia a extensão do OpenSubsonic.
func (a Account) Has(ext string) bool { return slices.Contains(a.Extensions, ext) }

// Auth são os parâmetros de autenticação (sem a senha).
func (a Account) Auth() url.Values {
	q := url.Values{"v": {Version}, "c": {Client}}
	if a.APIKey != "" {
		q.Set("apiKey", a.APIKey)
		return q
	}
	q.Set("u", a.User)
	q.Set("t", a.Token)
	q.Set("s", a.Salt)
	return q
}

// Error é o <error code message> do Subsonic.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("servidor de músicas: erro %d", e.Code)
	}
	return "servidor de músicas: " + e.Message
}

// códigos de login: senha/usuário (40), token não suportado (41), chave (44)
func authCode(code int) bool {
	return code == 40 || code == 41 || code == 42 || code == 43 || code == 44
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

// URL monta o endereço de um método com a autenticação (servidor e proxy).
func (a Account) URL(method string, q url.Values) string {
	all := a.Auth()
	for k, v := range q {
		all[k] = v
	}
	return a.Base + "/rest/" + method + ".view?" + all.Encode()
}

// response é o envelope {"subsonic-response": {...}}.
type response struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	Type          string `json:"type"`
	ServerVersion string `json:"serverVersion"`
	OpenSubsonic  bool   `json:"openSubsonic"`
	Error         *Error `json:"error"`
}

func (c *API) call(ctx context.Context, method string, q url.Values, out any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("f", "json")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Account.URL(method, q), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("servidor de músicas fora do ar? %w", stripURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrAuth
	}
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("servidor de músicas respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil {
		return errors.New("isso não parece um servidor Subsonic/Navidrome")
	}
	body, ok := env["subsonic-response"]
	if !ok {
		return errors.New("isso não parece um servidor Subsonic/Navidrome")
	}
	var head response
	if err := json.Unmarshal(body, &head); err != nil {
		return err
	}
	if head.Status != "ok" {
		if head.Error == nil {
			return errors.New("servidor de músicas: falhou sem dizer o porquê")
		}
		if authCode(head.Error.Code) {
			return ErrAuth
		}
		return head.Error
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

// stripURL tira o endereço do erro de rede (ele leva t e s na query).
func stripURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// Salt é um sal novo (o Subsonic pede pelo menos 6 caracteres).
func Salt() string {
	buf := make([]byte, 8)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

// TokenFor é o t do Subsonic: md5(senha + sal), em hex.
func TokenFor(pass, salt string) string {
	sum := md5.Sum([]byte(pass + salt))
	return hex.EncodeToString(sum[:])
}

// Login confere usuário e senha e devolve a conta com o par salgado (a senha
// não fica em lugar nenhum). key, se vier, é uma chave de API do
// OpenSubsonic (aí usuário e senha nem são usados).
func Login(ctx context.Context, base, user, pass, key string) (Account, error) {
	base, err := jellyfin.NormalizeBase(base)
	if err != nil {
		return Account{}, err
	}
	acc := Account{Base: base, User: strings.TrimSpace(user)}
	c := &API{Account: acc}
	// extensões e nome do servidor (sem login no OpenSubsonic)
	var ext struct {
		response
		Extensions []struct {
			Name string `json:"name"`
		} `json:"openSubsonicExtensions"`
	}
	if err := c.call(ctx, "getOpenSubsonicExtensions", nil, &ext); err == nil {
		for _, e := range ext.Extensions {
			acc.Extensions = append(acc.Extensions, e.Name)
		}
	}
	if key = strings.TrimSpace(key); key != "" {
		if !acc.Has("apiKeyAuthentication") {
			return Account{}, errors.New("este servidor não aceita chave de API: entre com usuário e senha")
		}
		acc.APIKey = key
	} else {
		if acc.User == "" {
			return Account{}, errors.New("informe o usuário")
		}
		acc.Salt = Salt()
		acc.Token = TokenFor(pass, acc.Salt)
	}
	c.Account = acc
	var ping response
	err = c.call(ctx, "ping", nil, &ping)
	if errors.Is(err, ErrAuth) {
		if acc.APIKey != "" {
			return Account{}, errors.New("chave de API recusada")
		}
		return Account{}, errors.New("usuário ou senha errados (ou o servidor não aceita login por token)")
	}
	if err != nil {
		return Account{}, err
	}
	acc.Server, acc.ServerVersion = ping.Type, ping.ServerVersion
	if acc.Server == "" {
		acc.Server = "subsonic"
	}
	if acc.APIKey != "" {
		var info struct {
			TokenInfo struct {
				Username string `json:"username"`
			} `json:"tokenInfo"`
		}
		c.Account = acc
		if c.call(ctx, "tokenInfo", nil, &info) == nil && info.TokenInfo.Username != "" {
			acc.User = info.TokenInfo.Username
		}
	}
	return acc, nil
}

// Artist é um artista (ID3).
type Artist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
	CoverArt   string `json:"coverArt"`
	Starred    string `json:"starred,omitempty"`
}

// Album é um álbum (ID3).
type Album struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	ArtistID  string `json:"artistId"`
	CoverArt  string `json:"coverArt"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
	Year      int    `json:"year"`
	Genre     string `json:"genre"`
	Starred   string `json:"starred,omitempty"`
	PlayCount int64  `json:"playCount"`
	Created   string `json:"created"`
}

// Song é uma faixa.
type Song struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Album       string `json:"album"`
	AlbumID     string `json:"albumId"`
	Artist      string `json:"artist"`
	ArtistID    string `json:"artistId"`
	Track       int    `json:"track"`
	DiscNumber  int    `json:"discNumber"`
	Year        int    `json:"year"`
	Genre       string `json:"genre"`
	CoverArt    string `json:"coverArt"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	Suffix      string `json:"suffix"`
	Duration    int    `json:"duration"`
	BitRate     int    `json:"bitRate"`
	Starred     string `json:"starred,omitempty"`
	PlayCount   int64  `json:"playCount"`
}

// Playlist é uma playlist (Entry só no getPlaylist).
type Playlist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Comment   string `json:"comment"`
	Owner     string `json:"owner"`
	Public    bool   `json:"public"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
	Changed   string `json:"changed"`
	CoverArt  string `json:"coverArt"`
	Entry     []Song `json:"entry"`
}

// Index é uma letra do índice de artistas.
type Index struct {
	Name    string   `json:"name"`
	Artists []Artist `json:"artist"`
}

// ScanStatus é o estado do scan da biblioteca.
type ScanStatus struct {
	Scanning    bool   `json:"scanning"`
	Count       int64  `json:"count"`
	FolderCount int64  `json:"folderCount"`
	LastScan    string `json:"lastScan"`
}

// User é um usuário do servidor (só o que o app mostra).
type User struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	AdminRole     bool   `json:"adminRole"`
	StreamRole    bool   `json:"streamRole"`
	DownloadRole  bool   `json:"downloadRole"`
	PlaylistRole  bool   `json:"playlistRole"`
	ScrobblingOn  bool   `json:"scrobblingEnabled"`
	SettingsRole  bool   `json:"settingsRole"`
	UploadRole    bool   `json:"uploadRole"`
	ShareRole     bool   `json:"shareRole"`
	CoverArtRole  bool   `json:"coverArtRole"`
	CommentRole   bool   `json:"commentRole"`
	PodcastRole   bool   `json:"podcastRole"`
	JukeboxRole   bool   `json:"jukeboxRole"`
	VideoConvRole bool   `json:"videoConversionRole"`
}

// Queue é a fila salva (getPlayQueue). Position em milissegundos.
type Queue struct {
	Current  string `json:"current"`
	Position int64  `json:"position"`
	Entry    []Song `json:"entry"`
}

// Ping confere a conta.
func (c *API) Ping(ctx context.Context) error { return c.call(ctx, "ping", nil, nil) }

// GetUser devolve o usuário (pra saber se é admin).
func (c *API) GetUser(ctx context.Context, name string) (User, error) {
	var out struct {
		User User `json:"user"`
	}
	err := c.call(ctx, "getUser", url.Values{"username": {name}}, &out)
	return out.User, err
}

// Users lista os usuários (só admin).
func (c *API) Users(ctx context.Context) ([]User, error) {
	var out struct {
		Users struct {
			User []User `json:"user"`
		} `json:"users"`
	}
	err := c.call(ctx, "getUsers", nil, &out)
	return out.Users.User, err
}

// Artists é o índice de artistas por letra.
func (c *API) Artists(ctx context.Context) ([]Index, error) {
	var out struct {
		Artists struct {
			Index []Index `json:"index"`
		} `json:"artists"`
	}
	err := c.call(ctx, "getArtists", nil, &out)
	return out.Artists.Index, err
}

// Artist devolve o artista e os álbuns dele.
func (c *API) Artist(ctx context.Context, id string) (Artist, []Album, error) {
	var out struct {
		Artist struct {
			Artist
			Album []Album `json:"album"`
		} `json:"artist"`
	}
	err := c.call(ctx, "getArtist", url.Values{"id": {id}}, &out)
	return out.Artist.Artist, out.Artist.Album, err
}

// TopSongs são as mais tocadas do artista (pelo nome, como manda o Subsonic).
func (c *API) TopSongs(ctx context.Context, artist string, count int) ([]Song, error) {
	var out struct {
		TopSongs struct {
			Song []Song `json:"song"`
		} `json:"topSongs"`
	}
	err := c.call(ctx, "getTopSongs", url.Values{"artist": {artist}, "count": {strconv.Itoa(count)}}, &out)
	return out.TopSongs.Song, err
}

// AlbumList2: kind é newest, recent, frequent, random, starred,
// alphabeticalByName, alphabeticalByArtist, byYear...
func (c *API) AlbumList2(ctx context.Context, kind string, size, offset int) ([]Album, error) {
	q := url.Values{"type": {kind}, "size": {strconv.Itoa(size)}, "offset": {strconv.Itoa(offset)}}
	if kind == "byYear" {
		q.Set("fromYear", "9999")
		q.Set("toYear", "0")
	}
	var out struct {
		AlbumList2 struct {
			Album []Album `json:"album"`
		} `json:"albumList2"`
	}
	err := c.call(ctx, "getAlbumList2", q, &out)
	return out.AlbumList2.Album, err
}

// Album devolve o álbum e as faixas.
func (c *API) Album(ctx context.Context, id string) (Album, []Song, error) {
	var out struct {
		Album struct {
			Album
			Song []Song `json:"song"`
		} `json:"album"`
	}
	err := c.call(ctx, "getAlbum", url.Values{"id": {id}}, &out)
	return out.Album.Album, out.Album.Song, err
}

// Song devolve uma faixa.
func (c *API) Song(ctx context.Context, id string) (Song, error) {
	var out struct {
		Song Song `json:"song"`
	}
	err := c.call(ctx, "getSong", url.Values{"id": {id}}, &out)
	return out.Song, err
}

// Found é o resultado de search3 e getStarred2.
type Found struct {
	Artists []Artist `json:"artist"`
	Albums  []Album  `json:"album"`
	Songs   []Song   `json:"song"`
}

// Search busca artistas, álbuns e faixas.
func (c *API) Search(ctx context.Context, query string, count int) (Found, error) {
	n := strconv.Itoa(count)
	var out struct {
		SearchResult3 Found `json:"searchResult3"`
	}
	err := c.call(ctx, "search3", url.Values{"query": {query}, "artistCount": {n}, "albumCount": {n}, "songCount": {n}}, &out)
	return out.SearchResult3, err
}

// Starred são as favoritas (artistas, álbuns e faixas).
func (c *API) Starred(ctx context.Context) (Found, error) {
	var out struct {
		Starred2 Found `json:"starred2"`
	}
	err := c.call(ctx, "getStarred2", nil, &out)
	return out.Starred2, err
}

// Star marca (ou desmarca) como favorita. kind: song, album ou artist.
func (c *API) Star(ctx context.Context, kind, id string, on bool) error {
	param := map[string]string{"song": "id", "album": "albumId", "artist": "artistId"}[kind]
	if param == "" {
		return errors.New("tipo desconhecido")
	}
	method := "star"
	if !on {
		method = "unstar"
	}
	return c.call(ctx, method, url.Values{param: {id}}, nil)
}

// Scrobble avisa que tocou (submission) ou que está tocando agora.
func (c *API) Scrobble(ctx context.Context, id string, submission bool, at time.Time) error {
	q := url.Values{"id": {id}, "submission": {strconv.FormatBool(submission)}}
	if submission && !at.IsZero() {
		q.Set("time", strconv.FormatInt(at.UnixMilli(), 10))
	}
	return c.call(ctx, "scrobble", q, nil)
}

// Playlists lista as playlists que o usuário vê.
func (c *API) Playlists(ctx context.Context) ([]Playlist, error) {
	var out struct {
		Playlists struct {
			Playlist []Playlist `json:"playlist"`
		} `json:"playlists"`
	}
	err := c.call(ctx, "getPlaylists", nil, &out)
	return out.Playlists.Playlist, err
}

// Playlist devolve uma playlist com as faixas.
func (c *API) Playlist(ctx context.Context, id string) (Playlist, error) {
	var out struct {
		Playlist Playlist `json:"playlist"`
	}
	err := c.call(ctx, "getPlaylist", url.Values{"id": {id}}, &out)
	return out.Playlist, err
}

// CreatePlaylist cria uma playlist com as faixas.
func (c *API) CreatePlaylist(ctx context.Context, name string, songs []string) (Playlist, error) {
	var out struct {
		Playlist Playlist `json:"playlist"`
	}
	err := c.call(ctx, "createPlaylist", url.Values{"name": {name}, "songId": songs}, &out)
	return out.Playlist, err
}

// SetPlaylistSongs troca as faixas da playlist (na ordem dada).
func (c *API) SetPlaylistSongs(ctx context.Context, id string, songs []string) error {
	return c.call(ctx, "createPlaylist", url.Values{"playlistId": {id}, "songId": songs}, nil)
}

// PlaylistChange é o que o updatePlaylist muda (nil = deixa como está).
type PlaylistChange struct {
	Name   *string
	Add    []string // faixas pro fim
	Remove []int    // posições (de 0) pra tirar
}

// UpdatePlaylist renomeia, tira e acrescenta faixas.
func (c *API) UpdatePlaylist(ctx context.Context, id string, ch PlaylistChange) error {
	q := url.Values{"playlistId": {id}}
	if ch.Name != nil {
		q.Set("name", *ch.Name)
	}
	if len(ch.Add) > 0 {
		q["songIdToAdd"] = ch.Add
	}
	for _, i := range ch.Remove {
		q.Add("songIndexToRemove", strconv.Itoa(i))
	}
	return c.call(ctx, "updatePlaylist", q, nil)
}

// DeletePlaylist apaga a playlist.
func (c *API) DeletePlaylist(ctx context.Context, id string) error {
	return c.call(ctx, "deletePlaylist", url.Values{"id": {id}}, nil)
}

// StartScan começa o scan da biblioteca (só admin).
func (c *API) StartScan(ctx context.Context) (ScanStatus, error) {
	var out struct {
		ScanStatus ScanStatus `json:"scanStatus"`
	}
	err := c.call(ctx, "startScan", nil, &out)
	return out.ScanStatus, err
}

// Scan é o estado do scan.
func (c *API) Scan(ctx context.Context) (ScanStatus, error) {
	var out struct {
		ScanStatus ScanStatus `json:"scanStatus"`
	}
	err := c.call(ctx, "getScanStatus", nil, &out)
	return out.ScanStatus, err
}

// SaveQueue guarda a fila no servidor (continua em outro aparelho ou depois).
func (c *API) SaveQueue(ctx context.Context, ids []string, current string, position time.Duration) error {
	q := url.Values{"id": ids}
	if current != "" {
		q.Set("current", current)
		q.Set("position", strconv.FormatInt(position.Milliseconds(), 10))
	}
	return c.call(ctx, "savePlayQueue", q, nil)
}

// Queue devolve a fila salva.
func (c *API) Queue(ctx context.Context) (Queue, error) {
	var out struct {
		PlayQueue Queue `json:"playQueue"`
	}
	err := c.call(ctx, "getPlayQueue", nil, &out)
	return out.PlayQueue, err
}

// Download baixa o arquivo original da faixa pra w a partir de offset
// (Range). Devolve o total do arquivo (0 = desconhecido).
func (c *API) Download(ctx context.Context, id string, offset int64, w io.Writer, progress func(n int64)) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Account.URL("download", url.Values{"id": {id}}), nil)
	if err != nil {
		return 0, err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	client := *c.http()
	client.Timeout = 0 // arquivo grande: quem manda é o ctx
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("servidor de músicas fora do ar? %w", stripURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && offset > 0 {
		return offset, nil // já estava inteiro
	}
	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("download recusado (%d)", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "json") || strings.Contains(ct, "xml") {
		return 0, errors.New("o servidor não deixou baixar esta faixa")
	}
	total := resp.ContentLength
	if offset > 0 && resp.StatusCode != http.StatusPartialContent {
		return 0, errResume
	}
	if total > 0 {
		total += offset
	}
	buf := make([]byte, 64<<10)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return total, werr
			}
			if progress != nil {
				progress(int64(n))
			}
		}
		if errors.Is(err, io.EOF) {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

// errResume: o servidor ignorou o Range (quem chama recomeça do zero).
var errResume = errors.New("o servidor não continua downloads")

// ErrResume diz se o servidor não aceitou continuar do meio.
func ErrResume(err error) bool { return errors.Is(err, errResume) }
