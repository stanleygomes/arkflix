package romm

import (
	"net/http"

	"github.com/stanleygomes/arkflix/server/internal/deps"
)

// New monta o handler da API. A FAZER.
func New(d *deps.Deps) http.Handler {
	return http.NotFoundHandler()
}
