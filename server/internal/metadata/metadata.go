// Package metadata completa o acervo com dados da internet (TMDB pra
// filmes e séries, libretro pra capas de jogos, MusicBrainz/Cover Art
// Archive pra música). Roda depois de cada scan, só pro que falta. A FAZER.
package metadata

import (
	"context"

	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Status é o andamento (o painel de admin mostra).
type Status struct {
	Running bool
	Pending int    // itens ainda sem metadados
	Done    int    // completados no último ciclo
	Error   string // último erro
}

// Fetcher busca e grava os metadados.
type Fetcher struct {
	cfg *config.Config
	st  *store.Store
}

// New cria o Fetcher.
func New(cfg *config.Config, st *store.Store) *Fetcher { return &Fetcher{cfg: cfg, st: st} }

// Run completa o que falta e volta quando acabar (ou ctx cancelar).
func (f *Fetcher) Run(ctx context.Context) error { return nil }

// Status devolve o andamento.
func (f *Fetcher) Status() Status { return Status{} }
