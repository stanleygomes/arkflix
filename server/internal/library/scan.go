package library

import (
	"context"

	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// scanAll: A FAZER (agente do scanner).
func scanAll(ctx context.Context, cfg *config.Config, st *store.Store) (int, error) {
	return 0, nil
}
