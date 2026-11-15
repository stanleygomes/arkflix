package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/arkserver"
	"github.com/stanleygomes/arkflix/app/internal/catalog"
	"github.com/stanleygomes/arkflix/app/internal/jellyfin"
)

// arkflix-server: quando a conta do Jellyfin do perfil (ou o servidor RomM)
// aponta pra um arkflix-server, o app ganha o painel de administração (só
// adulto e admin) e as listas do usuário do servidor. Nada novo é guardado:
// entra com o token que já existe e o resultado do /info fica só na memória.

var errNoArk = errors.New("nenhum servidor arkflix ligado a este perfil")

// arkTTL: quanto tempo vale a resposta do /info (falha vale menos).
const (
	arkTTL     = 10 * time.Minute
	arkFailTTL = time.Minute
)

type arkProbe struct {
	at   time.Time
	info arkserver.Info
	ok   bool
}

type arkState struct {
	mu     sync.Mutex
	probes map[string]arkProbe // por endereço do servidor
	http   *http.Client
}

func newArkState() *arkState {
	return &arkState{probes: map[string]arkProbe{}, http: &http.Client{Timeout: 10 * time.Second}}
}

// probe lê o /info do endereço (em cache; force busca de novo).
func (s *Service) arkProbe(base string, force bool) (arkserver.Info, bool) {
	s.ark.mu.Lock()
	p, cached := s.ark.probes[base]
	s.ark.mu.Unlock()
	ttl := arkFailTTL
	if p.ok {
		ttl = arkTTL
	}
	if cached && !force && time.Since(p.at) < ttl {
		return p.info, p.ok
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := arkserver.GetInfo(ctx, s.ark.http, base)
	p = arkProbe{at: time.Now(), info: info, ok: err == nil}
	s.ark.mu.Lock()
	s.ark.probes[base] = p
	s.ark.mu.Unlock()
	return p.info, p.ok
}

// arkDetect confere um servidor recém-ligado (em segundo plano).
func (s *Service) arkDetect(base string) {
	if base != "" {
		go s.arkProbe(base, true)
	}
}

type arkConn struct {
	client *arkserver.Client
	info   arkserver.Info
	source string // jellyfin ou romm
}

// arkConn: o servidor arkflix do perfil em uso. Primeiro a conta do Jellyfin
// do perfil; sem ela (ou se não for arkflix), o servidor RomM do app.
func (s *Service) arkConn() (arkConn, error) {
	if l := s.watchLinkOf(s.cur().id); l != nil {
		if info, ok := s.arkProbe(l.Base, false); ok {
			return arkConn{client: &arkserver.Client{Base: l.Base, Token: l.Token, Jellyfin: true}, info: info, source: "jellyfin"}, nil
		}
	}
	if conn, _ := s.remote.snapshot(); conn != nil {
		if info, ok := s.arkProbe(conn.Base, false); ok {
			return arkConn{client: &arkserver.Client{Base: conn.Base, Token: conn.Token}, info: info, source: "romm"}, nil
		}
	}
	return arkConn{}, errNoArk
}

// arkRomM: o servidor RomM do app é este arkflix (os jogos têm o mesmo rid).
func (s *Service) arkRomM(c arkConn) bool {
	conn, _ := s.remote.snapshot()
	return conn != nil && strings.TrimRight(conn.Base, "/") == strings.TrimRight(c.client.Base, "/")
}

func arkCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// ArkInfo é o que a interface sabe do servidor arkflix do perfil.
type ArkInfo struct {
	Found    bool     `json:"found"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Base     string   `json:"base"`
	Writable bool     `json:"writable"`
	Features []string `json:"features"`
	Source   string   `json:"source"` // jellyfin ou romm: de qual conta veio
	User     string   `json:"user"`
	// Admin: a conta é admin do servidor e o perfil é adulto (o painel aparece).
	Admin bool `json:"admin"`
	// Lists: o servidor tem listas; Videos/Games: dá pra pôr filmes (conta do
	// Jellyfin no servidor) e jogos (o RomM do app é este servidor).
	Lists  bool   `json:"lists"`
	Videos bool   `json:"videos"`
	Games  bool   `json:"games"`
	Error  string `json:"error,omitempty"`
}

// ArkServer diz se o perfil em uso tem um arkflix-server e o que dá pra fazer.
func (s *Service) ArkServer() ArkInfo {
	c, err := s.arkConn()
	if err != nil {
		return ArkInfo{Features: []string{}}
	}
	if info, ok := s.arkProbe(c.client.Base, true); ok {
		c.info = info // gravável muda sem o app saber
	}
	out := ArkInfo{Found: true, Name: c.info.Name, Version: c.info.Version, Base: c.client.Base, Writable: c.info.Writable,
		Features: c.info.Features, Source: c.source, Lists: c.info.Has("lists"), Videos: c.source == "jellyfin", Games: s.arkRomM(c)}
	if out.Features == nil {
		out.Features = []string{}
	}
	ctx, cancel := arkCtx()
	defer cancel()
	me, err := c.client.Me(ctx)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.User, out.Admin = me.Name, me.Admin && s.adult() == nil && c.info.Has("admin")
	return out
}

// arkAdmin: o servidor do perfil, só pra adulto (o servidor confere o admin).
func (s *Service) arkAdmin() (arkConn, error) {
	if err := s.adult(); err != nil {
		return arkConn{}, err
	}
	return s.arkConn()
}

// ArkStatus é o estado do servidor (bibliotecas, scan, metadados, sessões).
func (s *Service) ArkStatus() (arkserver.Status, error) {
	c, err := s.arkAdmin()
	if err != nil {
		return arkserver.Status{}, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	st, err := c.client.Status(ctx)
	if st.Libraries == nil {
		st.Libraries = []arkserver.Library{}
	}
	if st.Sessions == nil {
		st.Sessions = []arkserver.Session{}
	}
	return st, err
}

// arkScanPoll: de quanto em quanto tempo o scan é conferido.
var arkScanPoll = 2 * time.Second

// ArkScan dispara a varredura e acompanha até o fim (tarefa "arkscan").
func (s *Service) ArkScan() error {
	c, err := s.arkAdmin()
	if err != nil {
		return err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	if err := c.client.Scan(ctx); err != nil {
		return err
	}
	if !s.runTaskCtx("arkscan", "Escaneando o servidor", func(ctx context.Context, _ func(float64)) (string, error) {
		deadline := time.Now().Add(2 * time.Hour)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(arkScanPoll):
			}
			sctx, scancel := context.WithTimeout(ctx, 30*time.Second)
			st, err := c.client.Status(sctx)
			scancel()
			if err != nil {
				return "", err
			}
			if st.Scan.Running {
				continue
			}
			if st.Scan.Error != "" {
				return "", errors.New(st.Scan.Error)
			}
			return fmt.Sprintf("%d itens no servidor", st.Scan.Count), nil
		}
		return "", errors.New("o scan não terminou em 2 horas")
	}) {
		return errors.New("o servidor já está sendo escaneado")
	}
	return nil
}

// ArkUsers lista os usuários do servidor.
func (s *Service) ArkUsers() ([]arkserver.User, error) {
	c, err := s.arkAdmin()
	if err != nil {
		return nil, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	users, err := c.client.Users(ctx)
	if users == nil {
		users = []arkserver.User{}
	}
	return users, err
}

// ArkCreateUser cria um usuário no servidor (a senha vai direto pra lá).
func (s *Service) ArkCreateUser(name, password string, admin bool) (arkserver.User, error) {
	c, err := s.arkAdmin()
	if err != nil {
		return arkserver.User{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || password == "" {
		return arkserver.User{}, errors.New("informe o nome e a senha")
	}
	ctx, cancel := arkCtx()
	defer cancel()
	return c.client.CreateUser(ctx, name, password, admin)
}

// ArkSetPassword troca a senha de um usuário.
func (s *Service) ArkSetPassword(id, password string) (arkserver.User, error) {
	if password == "" {
		return arkserver.User{}, errors.New("informe a senha nova")
	}
	return s.arkPatchUser(id, arkserver.UserPatch{Password: &password})
}

// ArkSetAdmin liga ou desliga o admin de um usuário.
func (s *Service) ArkSetAdmin(id string, admin bool) (arkserver.User, error) {
	return s.arkPatchUser(id, arkserver.UserPatch{Admin: &admin})
}

func (s *Service) arkPatchUser(id string, p arkserver.UserPatch) (arkserver.User, error) {
	c, err := s.arkAdmin()
	if err != nil {
		return arkserver.User{}, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	return c.client.UpdateUser(ctx, id, p)
}

// ArkDeleteUser apaga um usuário do servidor.
func (s *Service) ArkDeleteUser(id string) error {
	c, err := s.arkAdmin()
	if err != nil {
		return err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	return c.client.DeleteUser(ctx, id)
}

// ArkPickFiles abre o seletor do sistema pra escolher o que enviar (desktop).
func (s *Service) ArkPickFiles() ([]string, error) {
	if err := s.adult(); err != nil {
		return nil, err
	}
	return s.opt.Window.PickAnyFiles("Escolha os arquivos pra enviar ao servidor")
}

// ArkPlatform é um console pro envio de ROM e BIOS.
type ArkPlatform struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// ArkPlatforms são os consoles do servidor (servidor sem a lista: os que o
// app conhece).
func (s *Service) ArkPlatforms() ([]ArkPlatform, error) {
	c, err := s.arkAdmin()
	if err != nil {
		return nil, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	out := []ArkPlatform{}
	list, err := c.client.Platforms(ctx)
	var apiErr *arkserver.Error
	switch {
	case err == nil:
		for _, p := range list {
			name := p.Name
			if sys, ok := catalog.Find(p.Slug); ok && name == "" {
				name = sys.Name
			}
			out = append(out, ArkPlatform{Slug: p.Slug, Name: cmp(name, p.Slug)})
		}
	case errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound:
		for _, sys := range catalog.Systems {
			out = append(out, ArkPlatform{Slug: sys.Slug, Name: sys.Name})
		}
	default:
		return nil, err
	}
	return out, nil
}

func cmp(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// campos aceitos por tipo de envio (o resto é ignorado); os obrigatórios
// vêm primeiro na mensagem de erro.
var arkUploadFields = map[string]struct{ required, optional []string }{
	"movie":    {nil, []string{"title", "year"}},
	"episode":  {[]string{"series", "season"}, []string{"episode", "title"}},
	"music":    {nil, []string{"folder"}},
	"rom":      {[]string{"platform"}, nil},
	"bios":     {[]string{"platform"}, nil},
	"subtitle": {[]string{"item"}, []string{"lang"}},
}

var arkFieldNames = map[string]string{"series": "a série", "season": "a temporada", "platform": "o console", "item": "o filme ou episódio"}

// ArkUpload envia arquivos pro servidor em segundo plano (tarefa
// "arkupload", cancelável). kind: movie, episode, music, rom, bios, subtitle.
func (s *Service) ArkUpload(kind string, fields map[string]string, paths []string) error {
	c, err := s.arkAdmin()
	if err != nil {
		return err
	}
	if info, ok := s.arkProbe(c.client.Base, true); ok {
		c.info = info
	}
	if !c.info.Writable {
		return errors.New("o servidor está em modo só leitura")
	}
	spec, ok := arkUploadFields[kind]
	if !ok {
		return errors.New("tipo de arquivo desconhecido")
	}
	clean := map[string]string{"kind": kind}
	for _, k := range slices.Concat(spec.required, spec.optional) {
		if v := strings.TrimSpace(fields[k]); v != "" {
			clean[k] = v
		}
	}
	for _, k := range spec.required {
		if clean[k] == "" {
			return fmt.Errorf("informe %s", arkFieldNames[k])
		}
	}
	for _, k := range []string{"year", "season", "episode"} {
		if v, ok := clean[k]; ok {
			if n, err := strconv.Atoi(v); err != nil || n < 0 {
				return fmt.Errorf("%s precisa ser um número", map[string]string{"year": "o ano", "season": "a temporada", "episode": "o episódio"}[k])
			}
		}
	}
	var files []string
	for _, p := range paths {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		if strings.HasPrefix(p, "~/") {
			home, _ := os.UserHomeDir()
			p = home + p[1:]
		}
		info, err := os.Stat(p)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("arquivo não encontrado: %s", p)
		}
		files = append(files, p)
	}
	if len(files) == 0 {
		return errors.New("escolha pelo menos um arquivo")
	}
	label := fmt.Sprintf("Enviando %d arquivo%s pro servidor", len(files), plural(len(files)))
	if !s.runTaskCtx("arkupload", label, func(ctx context.Context, progress func(float64)) (string, error) {
		last := time.Time{}
		saved, err := c.client.Upload(ctx, clean, files, func(done, total int64) {
			if total > 0 && time.Since(last) > 200*time.Millisecond {
				last = time.Now()
				progress(min(0.99, float64(done)/float64(total)))
			}
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d arquivo%s no servidor", len(saved), plural(len(saved))), nil
	}) {
		return errors.New("já tem um envio em andamento: espere terminar ou cancele")
	}
	return nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// arkRef: o id do item no servidor. kind "watch" = filme/série/episódio (o id
// do Jellyfin é o do servidor); "game" = jogo da biblioteca que está no RomM
// do servidor ("rom:<rid>").
func (s *Service) arkRef(c arkConn, kind, id string) (string, error) {
	switch kind {
	case "watch":
		if c.source != "jellyfin" {
			return "", errNoWatch
		}
		if id == "" {
			return "", errors.New("item inválido")
		}
		return id, nil
	case "game":
		if !s.arkRomM(c) {
			return "", errors.New("o servidor RomM ligado não é este servidor arkflix")
		}
		e, _, err := s.find(id)
		if err != nil {
			return "", err
		}
		if e.romID <= 0 {
			return "", errors.New("este jogo não está no servidor")
		}
		return "rom:" + strconv.Itoa(e.romID), nil
	}
	return "", errors.New("tipo desconhecido")
}

// ArkHide tira um filme, série ou jogo do acervo do servidor (move pra pasta
// de escondidos; desfazer é à mão no servidor).
func (s *Service) ArkHide(kind, id string) error {
	c, err := s.arkAdmin()
	if err != nil {
		return err
	}
	if info, ok := s.arkProbe(c.client.Base, true); ok {
		c.info = info
	}
	if !c.info.Writable {
		return errors.New("o servidor está em modo só leitura")
	}
	ref, err := s.arkRef(c, kind, id)
	if err != nil {
		return err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	if err := c.client.Hide(ctx, ref); err != nil {
		return err
	}
	if kind == "game" {
		s.startRemoteRefresh()
	}
	return nil
}

// ArkList é uma lista do servidor como a interface mostra: os filmes e
// séries viram WatchItem (abrem a página do título) e os jogos, ids da
// biblioteca (o perfil infantil só vê os liberados).
type ArkList struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Count     int         `json:"count"`
	Kinds     []string    `json:"kinds"`
	UpdatedAt time.Time   `json:"updatedAt"`
	Videos    []WatchItem `json:"videos"`
	Games     []string    `json:"games"`
	// Has: o item perguntado (ArkListsFor) está na lista.
	Has bool `json:"has"`
}

var arkImage = regexp.MustCompile(`^/arkflix/api/items/[A-Za-z0-9_-]+/image$`)

var arkVideoTypes = map[string]string{"movie": "Movie", "series": "Series", "episode": "Episode"}

func (s *Service) arkView(c arkConn, l arkserver.List, games map[int]string) ArkList {
	out := ArkList{ID: l.ID, Name: l.Name, Count: l.Count, Kinds: l.Kinds, UpdatedAt: l.UpdatedAt, Videos: []WatchItem{}, Games: []string{}}
	if out.Kinds == nil {
		out.Kinds = []string{}
	}
	base := ""
	if c.source == "jellyfin" {
		base = s.proxyBase()
	}
	for _, it := range l.Items {
		if t, ok := arkVideoTypes[it.Kind]; ok && c.source == "jellyfin" {
			v := WatchItem{Item: jellyfin.Item{ID: it.ID, Type: t, Name: it.Name, Year: it.Year, SeriesName: it.SeriesName, Genres: []string{}}}
			if base != "" && arkImage.MatchString(it.Image) {
				v.Poster = base + it.Image + "?maxWidth=360"
				v.Thumb = v.Poster
			}
			out.Videos = append(out.Videos, v)
		}
		if id, ok := games[it.RID]; ok && it.Kind == "rom" {
			out.Games = append(out.Games, id)
		}
	}
	return out
}

// arkGames: rid do RomM -> id da biblioteca, só o que o perfil pode ver.
func (s *Service) arkGames(c arkConn) map[int]string {
	out := map[int]string{}
	if !s.arkRomM(c) {
		return out
	}
	u := s.cur()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.romID > 0 && u.allows(e) {
			out[e.romID] = e.game.ID
		}
	}
	return out
}

func (s *Service) arkLists() (arkConn, error) {
	c, err := s.arkConn()
	if err == nil && !c.info.Has("lists") {
		err = errors.New("este servidor não tem listas")
	}
	return c, err
}

// arkDetails busca as listas com os itens (em paralelo).
func (s *Service) arkDetails(ctx context.Context, c arkConn) ([]arkserver.List, error) {
	lists, err := c.client.Lists(ctx)
	if err != nil {
		return nil, err
	}
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
	)
	for i := range lists {
		wg.Go(func() {
			full, err := c.client.List(ctx, lists[i].ID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				first = cmpErr(first, err)
				return
			}
			lists[i] = full
		})
	}
	wg.Wait()
	return lists, first
}

func cmpErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// ArkLists são as listas do usuário do servidor, com os itens.
func (s *Service) ArkLists() ([]ArkList, error) {
	c, err := s.arkLists()
	if err != nil {
		return nil, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	lists, err := s.arkDetails(ctx, c)
	if err != nil {
		return nil, err
	}
	games := s.arkGames(c)
	out := make([]ArkList, 0, len(lists))
	for _, l := range lists {
		out = append(out, s.arkView(c, l, games))
	}
	return out, nil
}

// ArkList é uma lista com os itens.
func (s *Service) ArkList(id string) (ArkList, error) {
	c, err := s.arkLists()
	if err != nil {
		return ArkList{}, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	l, err := c.client.List(ctx, id)
	if err != nil {
		return ArkList{}, err
	}
	return s.arkView(c, l, s.arkGames(c)), nil
}

// arkWrite: listas só mudam no perfil adulto (o infantil só vê).
func (s *Service) arkWrite() (arkConn, error) {
	if err := s.adult(); err != nil {
		return arkConn{}, err
	}
	return s.arkLists()
}

// ArkCreateList cria uma lista vazia no servidor.
func (s *Service) ArkCreateList(name string) (ArkList, error) {
	c, err := s.arkWrite()
	if err != nil {
		return ArkList{}, err
	}
	if name = strings.TrimSpace(name); name == "" {
		return ArkList{}, errors.New("dê um nome à lista")
	}
	ctx, cancel := arkCtx()
	defer cancel()
	l, err := c.client.CreateList(ctx, name)
	return s.arkView(c, l, nil), err
}

// ArkRenameList troca o nome de uma lista.
func (s *Service) ArkRenameList(id, name string) (ArkList, error) {
	c, err := s.arkWrite()
	if err != nil {
		return ArkList{}, err
	}
	if name = strings.TrimSpace(name); name == "" {
		return ArkList{}, errors.New("dê um nome à lista")
	}
	ctx, cancel := arkCtx()
	defer cancel()
	l, err := c.client.RenameList(ctx, id, name)
	return s.arkView(c, l, nil), err
}

// ArkDeleteList apaga uma lista (os itens continuam no servidor).
func (s *Service) ArkDeleteList(id string) error {
	c, err := s.arkWrite()
	if err != nil {
		return err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	return c.client.DeleteList(ctx, id)
}

// arkMatch: o item da lista é o pedido (id do servidor ou rom:<rid>).
func arkMatch(it arkserver.Item, ref string) bool {
	if rid, ok := strings.CutPrefix(ref, "rom:"); ok {
		return it.Kind == "rom" && strconv.Itoa(it.RID) == rid
	}
	return it.ID == ref
}

// ArkListsFor são as listas com Has marcado pro item (kind watch ou game).
func (s *Service) ArkListsFor(kind, id string) ([]ArkList, error) {
	c, err := s.arkLists()
	if err != nil {
		return nil, err
	}
	ref, err := s.arkRef(c, kind, id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	lists, err := s.arkDetails(ctx, c)
	if err != nil {
		return nil, err
	}
	out := make([]ArkList, 0, len(lists))
	for _, l := range lists {
		v := s.arkView(c, arkserver.List{ID: l.ID, Name: l.Name, Count: l.Count, Kinds: l.Kinds, UpdatedAt: l.UpdatedAt}, nil)
		v.Has = slices.ContainsFunc(l.Items, func(it arkserver.Item) bool { return arkMatch(it, ref) })
		out = append(out, v)
	}
	return out, nil
}

// ArkSetInList põe ou tira um filme/série/episódio (kind watch) ou jogo (kind
// game) da lista.
func (s *Service) ArkSetInList(list, kind, id string, in bool) error {
	c, err := s.arkWrite()
	if err != nil {
		return err
	}
	ref, err := s.arkRef(c, kind, id)
	if err != nil {
		return err
	}
	ctx, cancel := arkCtx()
	defer cancel()
	if in {
		_, err := c.client.AddToList(ctx, list, ref)
		return err
	}
	l, err := c.client.List(ctx, list)
	if err != nil {
		return err
	}
	for _, it := range l.Items {
		if arkMatch(it, ref) {
			return c.client.RemoveFromList(ctx, list, it.ID)
		}
	}
	return nil
}
