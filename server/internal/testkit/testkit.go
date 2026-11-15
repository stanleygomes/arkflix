// Package testkit monta, pros testes de cada API, um servidor de mentira:
// pastas de mídia de verdade (arquivos pequenos em testdata/), banco em
// memória, dois usuários e o acervo já gravado do jeito que o scanner
// grava (Seed). O teste do scanner confere que escanear essas pastas dá o
// mesmo acervo que Seed.
package testkit

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/deps"
	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Usuários criados por New.
const (
	Admin     = "stanley"
	AdminPass = "senha123"
	User      = "nina"
	UserPass  = "senha456"
)

// Testdata é a pasta com os arquivos de mídia de exemplo.
func Testdata() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// Kit é o servidor de mentira.
type Kit struct {
	*deps.Deps
	Root      string // raiz das pastas de mídia
	AdminUser *auth.User
	PlainUser *auth.User
}

// New cria as pastas, o banco e os usuários (sem gravar o acervo: chame
// Seed, ou rode o scanner).
func New(t testing.TB) *Kit {
	t.Helper()
	root := t.TempDir()
	cfg := &config.Config{
		Addr:   "127.0.0.1:0",
		Data:   filepath.Join(root, "data"),
		Movies: filepath.Join(root, "Movies"),
		Shows:  filepath.Join(root, "Shows"),
		Music:  filepath.Join(root, "musicas"),
		Games:  filepath.Join(root, "jogos"),
		Name:   "arkflix de teste",
	}
	cfg.FFmpeg, _ = lookPath("ffmpeg")
	cfg.FFprobe, _ = lookPath("ffprobe")
	if err := os.MkdirAll(cfg.Data, 0o700); err != nil {
		t.Fatal(err)
	}
	for rel, src := range Files {
		dst := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		var data []byte
		var err error
		if src == "" {
			data = []byte("rom de teste: " + rel)
		} else {
			data, err = os.ReadFile(filepath.Join(Testdata(), src))
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	au := auth.NewWithKey(st, make([]byte, 32))
	ctx := context.Background()
	admin, err := au.CreateUser(ctx, Admin, AdminPass, true)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := au.CreateUser(ctx, User, UserPass, false)
	if err != nil {
		t.Fatal(err)
	}
	d := &deps.Deps{Config: cfg, Store: st, Auth: au, Library: library.New(cfg, st), Media: media.New(cfg), Version: "0.0.0-teste"}
	return &Kit{Deps: d, Root: root, AdminUser: admin, PlainUser: plain}
}

// Files é a árvore de mídia (relativa à raiz) → arquivo de testdata ("" =
// conteúdo de mentira, pras ROMs e BIOS).
var Files = map[string]string{
	"Movies/O Grande Teste (2021)/O Grande Teste (2021).webm":   "clip.webm",
	"Movies/O Grande Teste (2021)/poster.png":                   "cover.png",
	"Movies/O Grande Teste (2021)/O Grande Teste (2021).pt.srt": "sub.srt",
	"Movies/Curta Pixel (2019).mp4":                             "clip.mp4",
	"Movies/Filme Sem Ano.webm":                                 "clip.webm",

	"Shows/Série de Teste/poster.png":                                         "cover.png",
	"Shows/Série de Teste/Season 01/Série de Teste - S01E01 - O Começo.webm":  "clip.webm",
	"Shows/Série de Teste/Season 01/Série de Teste - S01E02 - O Segundo.webm": "clip.webm",
	"Shows/Série de Teste/Season 02/S02E01 - Recomeço.webm":                   "clip.webm",

	"musicas/Banda Teste/Disco Teste/01 - Bip.mp3":  "tone.mp3",
	"musicas/Banda Teste/Disco Teste/02 - Bop.flac": "tone.flac",
	"musicas/Avulsas/Artista X - Solta.mp3":         "solo.mp3",

	"jogos/roms/snes/Super Teste (USA).sfc":                         "",
	"jogos/roms/snes/Outro Jogo (Europe) (En,Fr).sfc":               "",
	"jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 1).cue": "",
	"jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 1).bin": "",
	"jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 2).cue": "",
	"jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 2).bin": "",
	"jogos/bios/psx/scph1001.bin":                                   "",
}
