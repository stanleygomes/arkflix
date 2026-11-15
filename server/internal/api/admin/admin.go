// Package admin é a API própria do arkflix-server (/arkflix/api): status,
// scan, usuários, envio de arquivo, esconder item e listas. Contrato em
// API.md. A FAZER.
package admin

import (
	"net/http"

	"github.com/stanleygomes/arkflix/server/internal/deps"
)

// New monta o handler de /arkflix/.
func New(d *deps.Deps) http.Handler {
	return http.NotFoundHandler()
}
