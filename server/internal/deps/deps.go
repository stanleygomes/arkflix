// Package deps junta o que cada API recebe do servidor.
package deps

import (
	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Deps é passado pro New de cada API.
type Deps struct {
	Config  *config.Config
	Store   *store.Store
	Auth    *auth.Auth
	Library *library.Library
	Media   *media.Media
	Version string // versão do arkflix-server
}
