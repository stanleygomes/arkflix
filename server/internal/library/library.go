// Package library escaneia as pastas de mídia (só leitura) e grava o
// acervo em store.items. O scan é incremental: arquivo com mesmo tamanho
// e mtime não é relido (nem passa no ffprobe de novo).
package library

import (
	"context"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Status é o estado do scan (startScan/getScanStatus do Subsonic, tarefas
// do Jellyfin e do RomM leem daqui).
type Status struct {
	Running  bool
	Started  time.Time
	Finished time.Time
	Count    int // itens vistos no scan atual/último
	Error    string
}

// Library dispara e acompanha scans. Um por vez.
type Library struct {
	cfg *config.Config
	st  *store.Store

	mu     sync.Mutex
	status Status
	// OnScanned roda depois de cada scan que terminou bem (ex.: avisar
	// clientes pelo websocket). Opcional.
	OnScanned func()
}

// New cria a biblioteca.
func New(cfg *config.Config, st *store.Store) *Library {
	return &Library{cfg: cfg, st: st}
}

// Status devolve uma cópia do estado.
func (l *Library) Status() Status {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.status
}

// Trigger começa um scan em segundo plano (não faz nada se já tiver um
// rodando) e devolve na hora.
func (l *Library) Trigger() {
	go l.Scan(context.Background())
}

// Scan roda um scan completo e espera acabar. Se já houver um rodando,
// devolve nil sem fazer nada.
func (l *Library) Scan(ctx context.Context) error {
	l.mu.Lock()
	if l.status.Running {
		l.mu.Unlock()
		return nil
	}
	l.status = Status{Running: true, Started: time.Now()}
	l.mu.Unlock()

	n, err := l.scan(ctx)

	l.mu.Lock()
	l.status.Running, l.status.Finished, l.status.Count = false, time.Now(), n
	if err != nil {
		l.status.Error = err.Error()
	}
	cb := l.OnScanned
	l.mu.Unlock()
	if err == nil && cb != nil {
		cb()
	}
	return err
}

// scan percorre as bibliotecas configuradas. Implementado em scan.go.
func (l *Library) scan(ctx context.Context) (int, error) {
	return scanAll(ctx, l.cfg, l.st)
}
