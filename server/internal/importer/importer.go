// Package importer traz o que já existe nos servidores antigos (Jellyfin,
// Navidrome, RomM) pro arkflix-server: usuários, onde parou, favoritos,
// playlists e saves. Roda uma vez, pela linha de comando. A FAZER.
package importer

import (
	"context"
	"errors"

	"github.com/stanleygomes/arkflix/server/internal/deps"
)

// Run executa `arkflix-server import <fonte> [flags]`.
func Run(ctx context.Context, d *deps.Deps, args []string) error {
	return errors.New("import ainda não implementado")
}
