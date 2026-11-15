package fixture

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// FakeArk imita a API própria do arkflix-server (/arkflix/api): info, me,
// status, scan, usuários, envio, esconder e listas. Aceita o token do
// Jellyfin falso (X-Emby-Token ou MediaBrowser Token=) e os do RomM falso
// (Bearer): todos são do usuário "stanley", admin.
type FakeArk struct {
	mu       sync.Mutex
	writable bool
	admin    bool
	users    []fakeArkUser
	lists    []*fakeArkList
	items    map[string]fakeArkItem
	uploads  []string // "kind:campo=valor,...:arquivo:tamanho"
	hidden   []string
	auths    []string // como cada pedido entrou ("jellyfin" ou "romm")
	scanned  time.Time
	scans    int
	next     int
}

type fakeArkUser struct {
	ID, Name, Password string
	RID                int
	Admin              bool
	Created            time.Time
}

type fakeArkItem struct {
	ID, Kind, Name, Platform, SeriesName string
	RID, Year                            int
}

type fakeArkList struct {
	ID, Name string
	RID      int
	Items    []string
	Updated  time.Time
}

// NewFakeArk monta o servidor: gravável, stanley admin e ana comum, e a
// lista "Sessão de sábado" com um filme, a série e dois jogos do RomM falso.
func NewFakeArk() *FakeArk {
	now := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	f := &FakeArk{writable: true, admin: true, items: map[string]fakeArkItem{}, next: 10}
	for _, it := range []fakeArkItem{
		{ID: "m1", Kind: "movie", Name: "O Grande Teste", Year: 2024},
		{ID: "m2", Kind: "movie", Name: "Filme Sem Fim", Year: 2019},
		{ID: "m3", Kind: "movie", Name: "Curta Pixel", Year: 2022},
		{ID: "s1", Kind: "series", Name: "Série de Teste", Year: 2021},
		{ID: "e2", Kind: "episode", Name: "O Segundo", SeriesName: "Série de Teste"},
		{ID: "r1", RID: 1, Kind: "rom", Name: "Donkey Kong Country", Platform: "snes"},
		{ID: "r2", RID: 2, Kind: "rom", Name: "Spyro the Dragon", Platform: "psx"},
		{ID: "r3", RID: 3, Kind: "rom", Name: "Super Mario World", Platform: "snes"},
		{ID: "t1", RID: 9, Kind: "track", Name: "Tema de Abertura"},
	} {
		f.items[it.ID] = it
	}
	f.users = []fakeArkUser{
		{ID: "u1", RID: 1, Name: "stanley", Admin: true, Created: now},
		{ID: "u2", RID: 2, Name: "ana", Password: "ana12345", Created: now},
	}
	f.lists = []*fakeArkList{{ID: "l1", RID: 1, Name: "Sessão de sábado", Items: []string{"m1", "s1", "r1", "r3", "t1"}, Updated: now}}
	return f
}

// SetWritable liga ou desliga o modo gravável.
func (f *FakeArk) SetWritable(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writable = on
}

// SetAdmin troca o admin do usuário do token (stanley).
func (f *FakeArk) SetAdmin(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.admin = on
}

// Uploads: envios recebidos.
func (f *FakeArk) Uploads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.uploads)
}

// Hidden: ids escondidos.
func (f *FakeArk) Hidden() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.hidden)
}

// Auths: por qual token cada pedido entrou ("jellyfin" ou "romm").
func (f *FakeArk) Auths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.auths)
}

// ListItems devolve os ids de uma lista.
func (f *FakeArk) ListItems(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.lists {
		if l.ID == id {
			return slices.Clone(l.Items)
		}
	}
	return nil
}

func arkError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (f *FakeArk) who(r *http.Request) string {
	if r.Header.Get("X-Emby-Token") == fakeJFToken || strings.Contains(r.Header.Get("Authorization"), `Token="`+fakeJFToken+`"`) {
		return "jellyfin"
	}
	if b := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); b == fakeDevice || b == fakeAccess {
		return "romm"
	}
	return ""
}

func (f *FakeArk) itemView(id string) map[string]any {
	it := f.items[id]
	out := map[string]any{"id": it.ID, "rid": it.RID, "kind": it.Kind, "name": it.Name, "year": it.Year, "image": "/arkflix/api/items/" + it.ID + "/image"}
	if it.Platform != "" {
		out["platform"] = it.Platform
	}
	if it.SeriesName != "" {
		out["seriesName"] = it.SeriesName
	}
	return out
}

func (f *FakeArk) listView(l *fakeArkList, items bool) map[string]any {
	kinds := []string{}
	for _, id := range l.Items {
		if k := f.items[id].Kind; !slices.Contains(kinds, k) {
			kinds = append(kinds, k)
		}
	}
	cover := ""
	if len(l.Items) > 0 {
		cover = l.Items[0]
	}
	out := map[string]any{"id": l.ID, "rid": l.RID, "name": l.Name, "count": len(l.Items), "kinds": kinds, "updatedAt": l.Updated, "cover": cover}
	if items {
		list := []map[string]any{}
		for _, id := range l.Items {
			list = append(list, f.itemView(id))
		}
		out["items"] = list
	}
	return out
}

func (f *FakeArk) userView(u fakeArkUser) map[string]any {
	return map[string]any{"id": u.ID, "rid": u.RID, "name": u.Name, "admin": u.Admin, "createdAt": u.Created, "lastLogin": u.Created}
}

func (f *FakeArk) findList(id string) *fakeArkList {
	for _, l := range f.lists {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func (f *FakeArk) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/arkflix/api")
	if p == "/info" {
		f.mu.Lock()
		writeJSON(w, map[string]any{"product": "arkflix-server", "name": "arkflix de Teste", "version": "1.2.3", "writable": f.writable,
			"features": []string{"admin", "lists", "upload", "metadata", "import"}})
		f.mu.Unlock()
		return
	}
	who := f.who(r)
	if who == "" {
		arkError(w, http.StatusUnauthorized, "faça login")
		return
	}
	if p == "/upload" {
		f.upload(w, r) // lê o corpo sem segurar o lock
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auths = append(f.auths, who)
	parts := strings.Split(strings.Trim(p, "/"), "/")
	var body struct {
		Name     string   `json:"name"`
		Password *string  `json:"password"`
		Admin    *bool    `json:"admin"`
		IDs      []string `json:"ids"`
	}
	if r.Body != nil {
		json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body)
	}
	admin := func() bool {
		if !f.admin {
			arkError(w, http.StatusForbidden, "só administradores")
			return false
		}
		return true
	}
	switch {
	case p == "/me":
		writeJSON(w, map[string]any{"id": "u1", "rid": 1, "name": "stanley", "admin": f.admin})
	case p == "/status" && admin():
		running := !f.scanned.IsZero() && time.Since(f.scanned) < 600*time.Millisecond
		scan := map[string]any{"running": running, "startedAt": "2026-11-01T20:00:00Z", "finishedAt": "2026-11-01T20:01:00Z", "count": 1500 + f.scans, "error": ""}
		writeJSON(w, map[string]any{
			"version": "1.2.3", "writable": f.writable, "startedAt": "2026-11-01T19:00:00Z",
			"libraries": []map[string]any{
				{"id": "movies", "name": "Filmes", "path": "/var/media/Movies", "count": 120, "online": true},
				{"id": "shows", "name": "Séries", "path": "/var/media/Shows", "count": 14, "episodes": 420, "online": true},
				{"id": "music", "name": "Músicas", "path": "/var/media/Music", "count": 900, "albums": 80, "artists": 60, "online": false},
				{"id": "games", "name": "Jogos", "path": "/var/media/Games", "count": 300, "platforms": 12, "online": true},
			},
			"scan":     scan,
			"metadata": map[string]any{"enabled": true, "tmdb": false, "running": false, "pending": 12, "error": ""},
			"sessions": []map[string]any{{"user": "stanley", "client": "Infuse", "device": "Apple TV", "lastSeen": "2026-11-01T20:05:00Z"}},
		})
	case p == "/scan" && r.Method == http.MethodPost && admin():
		f.scanned = time.Now()
		f.scans++
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("{}"))
	case p == "/platforms" && admin():
		writeJSON(w, []map[string]any{{"slug": "snes", "name": "Super Nintendo", "count": 2}, {"slug": "psx", "name": "PlayStation", "count": 1}, {"slug": "gba", "name": "Game Boy Advance", "count": 0}})
	case parts[0] == "users" && admin():
		f.serveUsers(w, r, parts, body.Name, body.Password, body.Admin)
	case len(parts) == 3 && parts[0] == "items" && parts[2] == "hide" && r.Method == http.MethodPost && admin():
		if !f.writable {
			arkError(w, http.StatusForbidden, "servidor em modo só leitura")
			return
		}
		f.hidden = append(f.hidden, parts[1])
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 3 && parts[0] == "items" && parts[2] == "image" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "image/png")
		w.Write(CoverPNG(color.RGBA{R: 160, G: 40, B: 120, A: 255}))
	case parts[0] == "lists":
		f.serveLists(w, r, parts, body.Name, body.IDs)
	default:
		if !strings.HasPrefix(p, "/status") && !strings.HasPrefix(p, "/scan") && !strings.HasPrefix(p, "/platforms") && !strings.HasPrefix(p, "/users") && !strings.HasPrefix(p, "/items") {
			arkError(w, http.StatusNotFound, "rota desconhecida")
		}
	}
}

func (f *FakeArk) serveUsers(w http.ResponseWriter, r *http.Request, parts []string, name string, pass *string, adm *bool) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			out := []map[string]any{}
			for _, u := range f.users {
				out = append(out, f.userView(u))
			}
			writeJSON(w, out)
		case http.MethodPost:
			name = strings.TrimSpace(name)
			if name == "" || pass == nil || len(*pass) < 4 {
				arkError(w, http.StatusBadRequest, "nome e senha (4+) obrigatórios")
				return
			}
			for _, u := range f.users {
				if strings.EqualFold(u.Name, name) {
					arkError(w, http.StatusConflict, "já existe um usuário com esse nome")
					return
				}
			}
			f.next++
			u := fakeArkUser{ID: fmt.Sprintf("u%d", f.next), RID: f.next, Name: name, Password: *pass, Admin: adm != nil && *adm, Created: time.Now().UTC()}
			f.users = append(f.users, u)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(f.userView(u))
		}
		return
	}
	i := slices.IndexFunc(f.users, func(u fakeArkUser) bool { return u.ID == parts[1] })
	if i < 0 {
		arkError(w, http.StatusNotFound, "usuário não existe")
		return
	}
	switch r.Method {
	case http.MethodPatch:
		if pass != nil {
			f.users[i].Password = *pass
		}
		if adm != nil {
			f.users[i].Admin = *adm
		}
		writeJSON(w, f.userView(f.users[i]))
	case http.MethodDelete:
		if f.users[i].ID == "u1" {
			arkError(w, http.StatusConflict, "você não pode apagar a si mesmo")
			return
		}
		f.users = slices.Delete(f.users, i, i+1)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (f *FakeArk) serveLists(w http.ResponseWriter, r *http.Request, parts []string, name string, ids []string) {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			out := []map[string]any{}
			for _, l := range f.lists {
				out = append(out, f.listView(l, false))
			}
			writeJSON(w, out)
		case http.MethodPost:
			if strings.TrimSpace(name) == "" {
				arkError(w, http.StatusBadRequest, "dê um nome à lista")
				return
			}
			f.next++
			l := &fakeArkList{ID: fmt.Sprintf("l%d", f.next), RID: f.next, Name: strings.TrimSpace(name), Items: []string{}, Updated: time.Now().UTC()}
			f.lists = append(f.lists, l)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(f.listView(l, false))
		}
		return
	}
	l := f.findList(parts[1])
	if l == nil {
		arkError(w, http.StatusNotFound, "lista não existe")
		return
	}
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		writeJSON(w, f.listView(l, true))
	case len(parts) == 2 && r.Method == http.MethodPatch:
		l.Name = strings.TrimSpace(name)
		writeJSON(w, f.listView(l, false))
	case len(parts) == 2 && r.Method == http.MethodDelete:
		f.lists = slices.DeleteFunc(f.lists, func(x *fakeArkList) bool { return x == l })
		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 3 && parts[2] == "items" && r.Method == http.MethodPost:
		for _, id := range ids {
			if rid, ok := strings.CutPrefix(id, "rom:"); ok {
				for _, it := range f.items {
					if it.Kind == "rom" && fmt.Sprint(it.RID) == rid {
						id = it.ID
					}
				}
			}
			if _, ok := f.items[id]; !ok {
				arkError(w, http.StatusBadRequest, "item não existe: "+id)
				return
			}
			if !slices.Contains(l.Items, id) {
				l.Items = append(l.Items, id)
			}
		}
		l.Updated = time.Now().UTC()
		writeJSON(w, f.listView(l, false))
	case len(parts) == 4 && parts[2] == "items" && r.Method == http.MethodDelete:
		l.Items = slices.DeleteFunc(l.Items, func(x string) bool { return x == parts[3] })
		w.WriteHeader(http.StatusNoContent)
	default:
		arkError(w, http.StatusNotFound, "rota desconhecida")
	}
}

func (f *FakeArk) upload(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	writable, admin := f.writable, f.admin
	f.mu.Unlock()
	if !admin {
		arkError(w, http.StatusForbidden, "só administradores")
		return
	}
	if !writable {
		arkError(w, http.StatusForbidden, "servidor em modo só leitura")
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		arkError(w, http.StatusBadRequest, "envio inválido")
		return
	}
	var fields []string
	kind := ""
	saved := []string{}
	var names []string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			arkError(w, http.StatusBadRequest, "envio interrompido")
			return
		}
		if part.FormName() != "file" {
			v, _ := io.ReadAll(io.LimitReader(part, 1024))
			if part.FormName() == "kind" {
				kind = string(v)
			} else {
				fields = append(fields, part.FormName()+"="+string(v))
			}
			continue
		}
		n, _ := io.Copy(io.Discard, part)
		names = append(names, fmt.Sprintf("%s:%d", part.FileName(), n))
		saved = append(saved, kind+"/"+part.FileName())
	}
	if kind == "" || len(names) == 0 {
		arkError(w, http.StatusBadRequest, "faltou o tipo ou o arquivo")
		return
	}
	slices.Sort(fields)
	f.mu.Lock()
	for _, n := range names {
		f.uploads = append(f.uploads, kind+":"+strings.Join(fields, ",")+":"+n)
	}
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"saved": saved})
}

// FakeArkServer é um arkflix-server falso inteiro: a API própria
// (/arkflix/api), a do Jellyfin e a do RomM no mesmo endereço.
type FakeArkServer struct {
	Ark      *FakeArk
	Jellyfin *FakeJellyfin
	RomM     *FakeRomM
}

// NewFakeArkServer junta os três falsos.
func NewFakeArkServer() *FakeArkServer {
	return &FakeArkServer{Ark: NewFakeArk(), Jellyfin: NewFakeJellyfin(), RomM: NewFakeRomM()}
}

func (f *FakeArkServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch p := r.URL.Path; {
	case strings.HasPrefix(p, "/arkflix/"):
		f.Ark.ServeHTTP(w, r)
	case strings.HasPrefix(p, "/api/"), strings.HasPrefix(p, "/assets/"):
		f.RomM.ServeHTTP(w, r)
	default:
		f.Jellyfin.ServeHTTP(w, r)
	}
}
