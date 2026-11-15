// Package jellyfin fala a API do Jellyfin (a parte de consumo: login,
// navegar, imagens, tocar, onde parou, favoritos, legendas) em cima do
// acervo único, pra Infuse, Swiftfin, Findroid, jellyfin-web, Finamp e o
// app arkflix funcionarem sem mudar nada. Nada de admin, plugins ou TV.
package jellyfin

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/deps"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Versão que os clientes veem (eles conferem a mínima suportada).
const (
	productName = "Jellyfin Server"
	apiVersion  = "10.10.7"
)

// Server é a API montada.
type Server struct {
	d        *deps.Deps
	serverID string
	routes   []route
	sessions *sessions
	hls      *transcoder
	prefs    *displayPrefs
}

// New monta o handler da API.
func New(d *deps.Deps) http.Handler {
	s := &Server{d: d, serverID: serverID(d), sessions: newSessions(), prefs: &displayPrefs{st: d.Store}}
	s.hls = newTranscoder(d.Config.FFmpeg, filepath.Join(d.Config.Data, "transcodes"))
	s.routes = s.table()
	return s
}

// serverID é estável: sai da chave secreta (ou da pasta de dados).
func serverID(d *deps.Deps) string {
	seed := "arkflix:" + d.Config.Data
	if raw, err := os.ReadFile(filepath.Join(d.Config.Data, "secret.key")); err == nil {
		seed = "arkflix-jellyfin:" + strings.TrimSpace(string(raw))
	}
	sum := md5.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

// ServeHTTP tira o prefixo /emby ou /jellyfin, responde CORS e despacha.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cors(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	path := stripPrefix(r.URL.Path)
	segs := split(path)
	rt, params, ok := s.match(r.Method, segs)
	if !ok {
		notFound(w)
		return
	}
	c := &rc{s: s, w: w, r: r, p: params, q: lowerQuery(r)}
	if !rt.public {
		if !s.authenticate(c) {
			unauthorized(w)
			return
		}
		if !s.resolveTarget(c) {
			return
		}
	} else {
		s.optionalAuth(c)
	}
	rt.h(c)
}

// resolveTarget: userId de outro usuário só pra admin.
func (s *Server) resolveTarget(c *rc) bool {
	want := c.p["userId"]
	if want == "" {
		want = c.qs("userId")
	}
	if want == "" {
		return true
	}
	id := store.NormalizeID(want)
	if id == c.user.ID {
		return true
	}
	if !c.user.Admin {
		http.Error(c.w, "Forbidden", http.StatusForbidden)
		return false
	}
	if _, err := s.d.Auth.User(c.ctx(), id); err != nil {
		notFound(c.w)
		return false
	}
	c.target = id
	return true
}

func stripPrefix(p string) string {
	for _, pre := range []string{"/emby", "/jellyfin"} {
		if len(p) >= len(pre) && strings.EqualFold(p[:len(pre)], pre) && (len(p) == len(pre) || p[len(pre)] == '/') {
			return p[len(pre):]
		}
	}
	return p
}

func cors(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD")
	if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
		h.Set("Access-Control-Allow-Headers", req)
	} else {
		h.Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Authorization, Cache-Control, Content-Type, Origin, Range, X-Emby-Authorization, X-Emby-Token, X-MediaBrowser-Token")
	}
	h.Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges, Content-Disposition")
	h.Set("Access-Control-Max-Age", "86400")
}

func notFound(w http.ResponseWriter) {
	http.Error(w, "não encontrado", http.StatusNotFound)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `MediaBrowser realm="Jellyfin"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
