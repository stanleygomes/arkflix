package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/jellyfin"
)

// Filmes e séries: cada perfil vincula a própria conta do Jellyfin (o que ele
// assistiu, favoritos e o controle dos pais vêm de lá). A conta fica em
// <pasta do perfil>/jellyfin.json (0600), com o token, nunca a senha.

var errNoWatch = errors.New("vincule uma conta do Jellyfin ao perfil pra ver filmes e séries")

type watchLink struct {
	jellyfin.Account
	Linked int64 `json:"linked"`
}

type watchState struct {
	mu    sync.Mutex
	links map[string]*watchLink // por perfil (nil = sem conta)
	proxy *jellyfin.Proxy
}

func (s *Service) watchFile(profile string) string {
	return filepath.Join(s.profiles.DataDir(profile), "jellyfin.json")
}

// watchLinkOf lê (uma vez) a conta do perfil.
func (s *Service) watchLinkOf(profile string) *watchLink {
	s.watch.mu.Lock()
	defer s.watch.mu.Unlock()
	if l, ok := s.watch.links[profile]; ok {
		return l
	}
	var l *watchLink
	if raw, err := os.ReadFile(s.watchFile(profile)); err == nil {
		var w watchLink
		if json.Unmarshal(raw, &w) == nil && w.Token != "" {
			l = &w
		}
	}
	s.watch.links[profile] = l
	return l
}

func (s *Service) watchAPI() (*jellyfin.API, error) {
	l := s.watchLinkOf(s.cur().id)
	if l == nil {
		return nil, errNoWatch
	}
	return &jellyfin.API{Account: l.Account}, nil
}

func watchCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// WatchInfo é o estado da conta do Jellyfin de um perfil.
type WatchInfo struct {
	Connected bool   `json:"connected"`
	Base      string `json:"base"`
	User      string `json:"user"`
	Server    string `json:"server"`
}

func watchInfo(l *watchLink) WatchInfo {
	if l == nil {
		return WatchInfo{}
	}
	return WatchInfo{Connected: true, Base: l.Base, User: l.User, Server: l.Server}
}

// WatchStatus diz se o perfil em uso tem conta do Jellyfin.
func (s *Service) WatchStatus() WatchInfo { return watchInfo(s.watchLinkOf(s.cur().id)) }

// Accounts são as contas online vinculadas a um perfil.
type Accounts struct {
	Jellyfin WatchInfo `json:"jellyfin"`
	// RomM: um servidor pro app todo; Mine = é este perfil que sincroniza os saves.
	RomM struct {
		Connected bool   `json:"connected"`
		Base      string `json:"base"`
		User      string `json:"user"`
		Mine      bool   `json:"mine"`
		Owner     string `json:"owner"`
	} `json:"romm"`
}

// ProfileAccounts mostra as contas de um perfil (Jellyfin e RomM).
func (s *Service) ProfileAccounts(profile string) (Accounts, error) {
	if _, err := s.profiles.Get(profile); err != nil {
		return Accounts{}, err
	}
	var out Accounts
	out.Jellyfin = watchInfo(s.watchLinkOf(profile))
	if conn, _ := s.remote.snapshot(); conn != nil {
		out.RomM.Connected, out.RomM.Base, out.RomM.User, out.RomM.Mine = true, conn.Base, conn.User, conn.Owner == profile
		if p, err := s.profiles.Get(conn.Owner); err == nil {
			out.RomM.Owner = p.Name
		}
	}
	return out, nil
}

// LinkRomM passa a sincronização de saves do RomM pra este perfil.
func (s *Service) LinkRomM(profile string) (Accounts, error) {
	if err := s.adult(); err != nil {
		return Accounts{}, err
	}
	if _, err := s.profiles.Get(profile); err != nil {
		return Accounts{}, err
	}
	s.remote.mu.Lock()
	conn := s.remote.conn
	if conn == nil {
		s.remote.mu.Unlock()
		return Accounts{}, errors.New("nenhum servidor RomM ligado: conecte em Configurações > Servidor")
	}
	next := *conn
	next.Owner = profile
	if err := writePrivate(s.remote.file, &next); err != nil {
		s.remote.mu.Unlock()
		return Accounts{}, err
	}
	s.remote.conn = &next
	s.remote.mu.Unlock()
	return s.ProfileAccounts(profile)
}

// ConnectWatch vincula uma conta do Jellyfin a um perfil. A senha vira um
// token e é descartada. Só adulto vincula (inclusive pro perfil infantil:
// o controle dos pais do Jellyfin vale pra conta escolhida).
func (s *Service) ConnectWatch(profile, host, user, pass string) (WatchInfo, error) {
	if err := s.adult(); err != nil {
		return WatchInfo{}, err
	}
	if _, err := s.profiles.Get(profile); err != nil {
		return WatchInfo{}, err
	}
	if strings.TrimSpace(user) == "" {
		return WatchInfo{}, errors.New("informe o usuário")
	}
	buf := make([]byte, 8)
	rand.Read(buf)
	device, _ := os.Hostname()
	if device == "" {
		device = "arkflix"
	}
	ctx, cancel := watchCtx()
	defer cancel()
	acc, err := jellyfin.Login(ctx, host, strings.TrimSpace(user), pass, jellyfin.Account{
		DeviceID: "arkflix-" + hex.EncodeToString(buf), Device: device, Version: s.opt.Version,
	})
	if err != nil {
		return WatchInfo{}, err
	}
	l := &watchLink{Account: acc, Linked: time.Now().Unix()}
	if err := writePrivate(s.watchFile(profile), l); err != nil {
		return WatchInfo{}, err
	}
	s.watch.mu.Lock()
	old := s.watch.links[profile]
	s.watch.links[profile] = l
	s.watch.mu.Unlock()
	if old != nil {
		go s.watchLogout(old.Account)
	}
	s.arkDetect(acc.Base)
	return watchInfo(l), nil
}

// DisconnectWatch desvincula a conta do perfil e invalida o token no servidor.
func (s *Service) DisconnectWatch(profile string) error {
	if err := s.adult(); err != nil {
		return err
	}
	l := s.watchLinkOf(profile)
	if err := os.Remove(s.watchFile(profile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s.watch.mu.Lock()
	s.watch.links[profile] = nil
	s.watch.mu.Unlock()
	if l != nil {
		s.watchLogout(l.Account)
	}
	return nil
}

// watchLogout tenta invalidar o token (servidor fora do ar não impede nada).
func (s *Service) watchLogout(acc jellyfin.Account) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	(&jellyfin.API{Account: acc}).Logout(ctx)
}

// forgetWatch esquece a conta de um perfil apagado.
func (s *Service) forgetWatch(profile string) {
	s.watch.mu.Lock()
	delete(s.watch.links, profile)
	s.watch.mu.Unlock()
}

// WatchItem é um filme, série, temporada ou episódio como a interface mostra.
type WatchItem struct {
	jellyfin.Item
	Poster   string  `json:"poster"`
	Backdrop string  `json:"backdrop"`
	Logo     string  `json:"logo"`
	Thumb    string  `json:"thumb"`
	Progress float64 `json:"progress"` // 0-1
}

func (s *Service) proxyBase() string {
	base, err := s.watch.proxy.URL()
	if err != nil {
		return ""
	}
	return base
}

func image(base, item, kind, tag string, height int) string {
	if base == "" || tag == "" || item == "" {
		return ""
	}
	q := url.Values{"tag": {tag}, "quality": {"90"}}
	if height > 0 {
		q.Set("maxHeight", strconv.Itoa(height))
	}
	return base + "/Items/" + url.PathEscape(item) + "/Images/" + kind + "?" + q.Encode()
}

func (s *Service) watchView(it jellyfin.Item) WatchItem {
	base := s.proxyBase()
	v := WatchItem{
		Item:     it,
		Poster:   image(base, it.PrimaryItem, "Primary", it.Primary, 540),
		Backdrop: image(base, it.BackdropItem, "Backdrop", it.Backdrop, 1080),
		Logo:     image(base, it.LogoItem, "Logo", it.Logo, 240),
		Thumb:    image(base, it.ThumbItem, "Thumb", it.Thumb, 360),
	}
	if it.Runtime > 0 && it.Position > 0 {
		v.Progress = min(1, it.Position/it.Runtime)
	}
	return v
}

func (s *Service) watchViews(list []jellyfin.Item) []WatchItem {
	out := make([]WatchItem, 0, len(list))
	for _, it := range list {
		out = append(out, s.watchView(it))
	}
	return out
}

// WatchHome é a tela inicial de filmes e séries.
type WatchHome struct {
	Resume    []WatchItem `json:"resume"`
	NextUp    []WatchItem `json:"nextUp"`
	Movies    []WatchItem `json:"movies"`
	Series    []WatchItem `json:"series"`
	Favorites []WatchItem `json:"favorites"`
}

// WatchHome busca as fileiras do início (em paralelo; uma que falhe fica vazia).
func (s *Service) WatchHome() (WatchHome, error) {
	c, err := s.watchAPI()
	if err != nil {
		return WatchHome{}, err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
		out   WatchHome
	)
	fetch := func(dst *[]WatchItem, fn func() ([]jellyfin.Item, error)) {
		wg.Go(func() {
			list, err := fn()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if first == nil {
					first = err
				}
				*dst = []WatchItem{}
				return
			}
			*dst = s.watchViews(list)
		})
	}
	fetch(&out.Resume, func() ([]jellyfin.Item, error) { return c.Resume(ctx, 16) })
	fetch(&out.NextUp, func() ([]jellyfin.Item, error) { return c.NextUp(ctx, 16) })
	fetch(&out.Movies, func() ([]jellyfin.Item, error) { return c.Latest(ctx, "Movie", 24) })
	fetch(&out.Series, func() ([]jellyfin.Item, error) { return c.Latest(ctx, "Series", 24) })
	fetch(&out.Favorites, func() ([]jellyfin.Item, error) {
		l, _, err := c.Items(ctx, jellyfin.Query{Types: "Movie,Series", Favorite: true, Sort: "added", Limit: 24})
		return l, err
	})
	wg.Wait()
	// tudo falhou do mesmo jeito (servidor fora, token revogado): avisa
	if first != nil && len(out.Resume)+len(out.NextUp)+len(out.Movies)+len(out.Series)+len(out.Favorites) == 0 {
		return out, first
	}
	return out, nil
}

// WatchPage é uma página da lista completa.
type WatchPage struct {
	Items []WatchItem `json:"items"`
	Total int         `json:"total"`
}

var watchKinds = map[string]string{"movies": "Movie", "series": "Series", "all": "Movie,Series"}

// WatchList lista filmes ou séries: kind é movies, series, all ou favorites;
// sort é name, added, year ou score.
func (s *Service) WatchList(kind, sort, genre string, start, limit int) (WatchPage, error) {
	c, err := s.watchAPI()
	if err != nil {
		return WatchPage{}, err
	}
	q := jellyfin.Query{Types: watchKinds[kind], Sort: sort, Genre: genre, Start: max(0, start), Limit: min(max(limit, 1), 200)}
	if kind == "favorites" {
		q.Types, q.Favorite = "Movie,Series", true
	}
	if q.Types == "" {
		return WatchPage{}, errors.New("lista desconhecida")
	}
	ctx, cancel := watchCtx()
	defer cancel()
	items, total, err := c.Items(ctx, q)
	return WatchPage{Items: s.watchViews(items), Total: total}, err
}

// WatchSearch busca filmes e séries pelo nome.
func (s *Service) WatchSearch(text string) ([]WatchItem, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return []WatchItem{}, nil
	}
	c, err := s.watchAPI()
	if err != nil {
		return nil, err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	items, _, err := c.Items(ctx, jellyfin.Query{Types: "Movie,Series", Search: text, Limit: 40})
	return s.watchViews(items), err
}

// WatchDetails é a página de um filme ou série (com temporadas).
type WatchDetails struct {
	Item    WatchItem   `json:"item"`
	Seasons []WatchItem `json:"seasons"`
	// Next: o episódio pra continuar ou começar (séries).
	Next *WatchItem `json:"next"`
}

// WatchDetails busca um item com o que a página dele precisa.
func (s *Service) WatchDetails(id string) (WatchDetails, error) {
	c, err := s.watchAPI()
	if err != nil {
		return WatchDetails{}, err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	it, err := c.Item(ctx, id)
	if err != nil {
		return WatchDetails{}, err
	}
	out := WatchDetails{Item: s.watchView(it), Seasons: []WatchItem{}}
	if it.Type != "Series" {
		return out, nil
	}
	seasons, err := c.Seasons(ctx, id)
	if err != nil {
		return out, err
	}
	out.Seasons = s.watchViews(seasons)
	eps, err := c.Episodes(ctx, id, "")
	if err != nil {
		return out, err
	}
	// o primeiro começado e não terminado; se não, o primeiro não visto
	for _, pass := range []func(jellyfin.Item) bool{
		func(e jellyfin.Item) bool { return e.Position > 0 && !e.Played },
		func(e jellyfin.Item) bool { return !e.Played },
	} {
		for _, e := range eps {
			if pass(e) {
				v := s.watchView(e)
				out.Next = &v
				return out, nil
			}
		}
	}
	if len(eps) > 0 {
		v := s.watchView(eps[0])
		out.Next = &v
	}
	return out, nil
}

// WatchEpisodes lista os episódios de uma temporada.
func (s *Service) WatchEpisodes(series, season string) ([]WatchItem, error) {
	c, err := s.watchAPI()
	if err != nil {
		return nil, err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	eps, err := c.Episodes(ctx, series, season)
	return s.watchViews(eps), err
}

// WatchStream é como a interface toca: URL (no proxy local), faixas e de onde
// começar.
type WatchStream struct {
	jellyfin.Stream
	URL   string     `json:"url"`
	Start float64    `json:"start"`
	Item  WatchItem  `json:"item"`
	Next  *WatchItem `json:"next"` // próximo episódio (séries)
}

// WatchPlay prepara a reprodução. start < 0 = de onde parou; audio < 0 = a
// faixa padrão (outra faixa reinicia o vídeo no mesmo ponto).
func (s *Service) WatchPlay(id string, audio int, start float64) (WatchStream, error) {
	c, err := s.watchAPI()
	if err != nil {
		return WatchStream{}, err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	it, err := c.Item(ctx, id)
	if err != nil {
		return WatchStream{}, err
	}
	if it.Type != "Movie" && it.Type != "Episode" {
		return WatchStream{}, errors.New("escolha um filme ou episódio")
	}
	if start < 0 {
		start = 0
		if !it.Played {
			start = it.Position
		}
	}
	st, err := c.Play(ctx, id, audio, start)
	if err != nil {
		return WatchStream{}, err
	}
	base, err := s.watch.proxy.URL()
	if err != nil {
		return WatchStream{}, err
	}
	out := WatchStream{Stream: st, URL: base + st.Path, Start: start, Item: s.watchView(it)}
	for i := range out.Subtitles {
		out.Subtitles[i].URL = base + out.Subtitles[i].URL
	}
	if it.Type == "Episode" && it.SeriesID != "" {
		if eps, err := c.Episodes(ctx, it.SeriesID, ""); err == nil {
			for i, e := range eps {
				if e.ID == id && i+1 < len(eps) {
					v := s.watchView(eps[i+1])
					out.Next = &v
				}
			}
		}
	}
	return out, nil
}

// WatchReport avisa o servidor do progresso (event: start, progress, stop).
func (s *Service) WatchReport(event, id, session, source string, position float64, paused bool) error {
	c, err := s.watchAPI()
	if err != nil {
		return err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	return c.Report(ctx, event, id, session, source, max(0, position), paused)
}

// WatchFavorite marca ou desmarca como favorito.
func (s *Service) WatchFavorite(id string, fav bool) (WatchItem, error) {
	return s.watchMark(id, func(ctx context.Context, c *jellyfin.API) error { return c.SetFavorite(ctx, id, fav) })
}

// WatchPlayed marca como assistido ou não.
func (s *Service) WatchPlayed(id string, played bool) (WatchItem, error) {
	return s.watchMark(id, func(ctx context.Context, c *jellyfin.API) error { return c.SetPlayed(ctx, id, played) })
}

func (s *Service) watchMark(id string, fn func(context.Context, *jellyfin.API) error) (WatchItem, error) {
	c, err := s.watchAPI()
	if err != nil {
		return WatchItem{}, err
	}
	ctx, cancel := watchCtx()
	defer cancel()
	if err := fn(ctx, c); err != nil {
		return WatchItem{}, err
	}
	it, err := c.Item(ctx, id)
	return s.watchView(it), err
}

func newWatchState(s *Service) *watchState {
	w := &watchState{links: map[string]*watchLink{}}
	w.proxy = &jellyfin.Proxy{Account: func() (jellyfin.Account, bool) {
		l := s.watchLinkOf(s.cur().id)
		if l == nil {
			return jellyfin.Account{}, false
		}
		return l.Account, true
	}}
	return w
}
