// Package admin é a API própria do arkflix-server (/arkflix/api): status,
// scan, usuários, envio de arquivo, esconder item e listas. Contrato em
// API.md (o app arkflix é construído em cima dele).
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/deps"
	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Prefix é onde a API mora.
const Prefix = "/arkflix/api"

type api struct {
	d       *deps.Deps
	lists   *lists.Lists
	started time.Time
}

// quem pode chamar a rota
type level int

const (
	public level = iota
	user
	admin
)

// handlerFunc recebe o usuário logado (nil em rota pública).
type handlerFunc func(w http.ResponseWriter, r *http.Request, u *auth.User)

// New monta o handler de /arkflix/.
func New(d *deps.Deps) http.Handler {
	a := &api{d: d, lists: lists.New(d.Store), started: time.Now().UTC()}
	mux := http.NewServeMux()
	h := func(pattern string, lv level, fn handlerFunc) {
		method, path, _ := strings.Cut(pattern, " ")
		mux.HandleFunc(method+" "+Prefix+path, a.wrap(lv, fn))
	}
	h("GET /info", public, a.info)
	h("GET /me", user, a.me)

	h("GET /status", admin, a.status)
	h("POST /scan", admin, a.scan)
	h("GET /users", admin, a.users)
	h("POST /users", admin, a.createUser)
	h("PATCH /users/{id}", admin, a.patchUser)
	h("DELETE /users/{id}", admin, a.deleteUser)
	h("GET /platforms", admin, a.platforms)
	h("POST /upload", admin, a.upload)
	h("POST /items/{id}/hide", admin, a.hide)
	h("POST /items/{id}/refresh", admin, a.refresh)
	h("GET /items/{id}/image", user, a.image)

	h("GET /lists", user, a.listLists)
	h("POST /lists", user, a.createList)
	h("GET /lists/{id}", user, a.getList)
	h("PATCH /lists/{id}", user, a.patchList)
	h("DELETE /lists/{id}", user, a.deleteList)
	h("POST /lists/{id}/items", user, a.addListItems)
	h("DELETE /lists/{id}/items/{itemId}", user, a.removeListItem)
	h("PUT /lists/{id}/order", user, a.orderList)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fail(w, http.StatusNotFound, "não encontrado") })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cors(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func cors(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
		h.Set("Access-Control-Allow-Headers", req)
	} else {
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Emby-Token, X-Emby-Authorization, X-MediaBrowser-Token")
	}
}

func (a *api) wrap(lv level, fn handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if lv == public {
			fn(w, r, nil)
			return
		}
		u := a.authenticate(r)
		if u == nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			fail(w, http.StatusUnauthorized, "login necessário")
			return
		}
		if lv == admin && !u.Admin {
			fail(w, http.StatusForbidden, "só administrador")
			return
		}
		fn(w, r, u)
	}
}

// tokens que valem aqui: os que o servidor já emite pros clientes
var tokenKinds = []string{auth.KindJellyfin, auth.KindRomM, auth.KindRomMClient, auth.KindSubsonic}

func (a *api) authenticate(r *http.Request) *auth.User {
	tok := requestToken(r)
	if tok == "" {
		return nil
	}
	u, _, err := a.d.Auth.CheckToken(r.Context(), tok, tokenKinds...)
	if err != nil {
		return nil
	}
	return u
}

// requestToken acha o token: Bearer, MediaBrowser ...Token="..", os
// cabeçalhos do Jellyfin ou ?api_key= (imagem em <img>).
func requestToken(r *http.Request) string {
	for _, hdr := range []string{"Authorization", "X-Emby-Authorization"} {
		v := strings.TrimSpace(r.Header.Get(hdr))
		scheme, rest, _ := strings.Cut(v, " ")
		switch strings.ToLower(scheme) {
		case "bearer":
			return strings.TrimSpace(rest)
		case "mediabrowser", "emby":
			if t := mediaBrowserToken(rest); t != "" {
				return t
			}
		}
	}
	for _, hdr := range []string{"X-Emby-Token", "X-MediaBrowser-Token"} {
		if t := strings.TrimSpace(r.Header.Get(hdr)); t != "" {
			return t
		}
	}
	return r.URL.Query().Get("api_key")
}

// mediaBrowserToken lê o Token="..." de `Client="..", Token=".."`.
func mediaBrowserToken(v string) string {
	for part := range strings.SplitSeq(v, ",") {
		k, val, ok := strings.Cut(part, "=")
		if ok && strings.EqualFold(strings.TrimSpace(k), "token") {
			val = strings.Trim(strings.TrimSpace(val), `"`)
			if u, err := url.PathUnescape(val); err == nil {
				val = u
			}
			return val
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("arkflix api: json", "err", err)
	}
}

// fail responde {"error": "..."}.
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func internal(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "não encontrado")
		return
	}
	slog.Error("arkflix api", "err", err)
	fail(w, http.StatusInternalServerError, "erro interno")
}

// readJSON decodifica o corpo (até 1 MiB).
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil && err != io.EOF {
		fail(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}

// date formata em RFC 3339 UTC; nunca = null (string vazia quebra cliente
// que lê data tipada, como o time.Time do Go).
func date(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// GET /info (sem login)
func (a *api) info(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	writeJSON(w, http.StatusOK, map[string]any{
		"product": "arkflix-server", "name": a.d.Config.Name, "version": a.d.Version, "writable": a.d.Config.Writable,
		"features": []string{"admin", "lists", "upload", "metadata", "import"},
	})
}

// GET /me
func (a *api) me(w http.ResponseWriter, r *http.Request, u *auth.User) {
	writeJSON(w, http.StatusOK, map[string]any{"id": u.ID, "rid": u.RID, "name": u.Name, "admin": u.Admin})
}

// POST /scan
func (a *api) scan(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	a.d.Library.Trigger()
	writeJSON(w, http.StatusAccepted, map[string]any{})
}

// POST /items/{id}/refresh: busca os metadados desse item de novo, já, em
// segundo plano (o que veio do disco continua valendo).
func (a *api) refresh(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	it, err := a.item(r.Context(), r.PathValue("id"))
	if err != nil {
		internal(w, err)
		return
	}
	if a.d.Meta != nil {
		go func() {
			if err := a.d.Meta.Refresh(context.Background(), it.ID); err != nil {
				slog.Warn("metadados", "item", it.ID, "err", err)
			}
		}()
	}
	writeJSON(w, http.StatusAccepted, map[string]any{})
}

// GET /platforms: consoles do acervo (o app usa pra escolher onde vai uma
// ROM ou BIOS enviada).
func (a *api) platforms(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	rows, err := a.d.Store.DB.QueryContext(r.Context(), `SELECT p.platform, p.name,
  (SELECT COUNT(*) FROM items i WHERE i.parent_id = p.id AND i.kind = ?)
FROM items p WHERE p.kind = ? ORDER BY p.sort_name`, store.KindRom, store.KindPlatform)
	if err != nil {
		internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var slug, name string
		var n int
		if err := rows.Scan(&slug, &name, &n); err != nil {
			internal(w, err)
			return
		}
		out = append(out, map[string]any{"slug": slug, "name": name, "count": n})
	}
	writeJSON(w, http.StatusOK, out)
}
