// Package web junta as três APIs numa porta só:
//
//	/rest/...                 Subsonic/OpenSubsonic (Navidrome)
//	/api/..., /assets/romm/... RomM
//	/arkflix/api/...          API própria (admin, listas: api/admin/API.md)
//	/health                   saúde (sem login)
//	todo o resto              Jellyfin
//
// Os três domínios de hoje (filmes, música, jogos) podem apontar pra cá.
package web

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/api/admin"
	"github.com/stanleygomes/arkflix/server/internal/api/jellyfin"
	"github.com/stanleygomes/arkflix/server/internal/api/romm"
	"github.com/stanleygomes/arkflix/server/internal/api/subsonic"
	"github.com/stanleygomes/arkflix/server/internal/deps"
)

// New devolve o handler raiz.
func New(d *deps.Deps) http.Handler {
	sub, rm, jf, adm := subsonic.New(d), romm.New(d), jellyfin.New(d), admin.New(d)
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.ToLower(r.URL.Path)
		switch {
		case p == "/health":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("ok"))
		case strings.HasPrefix(p, "/arkflix/"):
			adm.ServeHTTP(w, r)
		case strings.HasPrefix(p, "/rest/") || p == "/rest":
			sub.ServeHTTP(w, r)
		case strings.HasPrefix(p, "/api/") || p == "/api" || strings.HasPrefix(p, "/assets/romm/"):
			rm.ServeHTTP(w, r)
		default:
			jf.ServeHTTP(w, r)
		}
	})
	return logRequests(recoverPanics(mux))
}

type recorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *recorder) WriteHeader(s int) { r.status = s; r.ResponseWriter.WriteHeader(s) }
func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = 200
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}
func (r *recorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// segredos que clientes mandam na URL
var secretParams = []string{"api_key", "apikey", "api-key", "token", "t", "p", "s", "x-emby-token", "x-mediabrowser-token", "access_token", "password"}

// Redact tira senha e token da query pra ir pro log.
func Redact(u *url.URL) string {
	q := u.Query()
	for k := range q {
		for _, s := range secretParams {
			if strings.EqualFold(k, s) {
				q.Set(k, "***")
			}
		}
	}
	if len(q) == 0 {
		return u.Path
	}
	return u.Path + "?" + q.Encode()
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		slog.Info("http", "method", r.Method, "url", Redact(r.URL), "status", rec.status, "bytes", rec.bytes,
			"ms", time.Since(start).Milliseconds(), "ua", r.UserAgent())
	})
}

func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil && v != http.ErrAbortHandler {
				slog.Error("panic", "url", Redact(r.URL), "err", v)
				http.Error(w, "erro interno", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
