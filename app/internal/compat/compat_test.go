//go:build compat

// Package compat roda os clientes do app (Jellyfin e RomM) contra o
// arkflix-server de verdade: compila o servidor, monta uma biblioteca com
// os arquivos de teste dele e faz as mesmas chamadas que o app faz.
//
//	cd app && go test -tags compat ./internal/compat/
package compat

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/jellyfin"
	"github.com/stanleygomes/arkflix/app/internal/romm"
)

// arquivos de teste do servidor
var tree = map[string]string{
	"Movies/O Grande Teste (2021)/O Grande Teste (2021).webm":                 "clip.webm",
	"Movies/O Grande Teste (2021)/poster.png":                                 "cover.png",
	"Movies/O Grande Teste (2021)/O Grande Teste (2021).pt.srt":               "sub.srt",
	"Movies/Curta Pixel (2019).mp4":                                           "clip.mp4",
	"Shows/Série de Teste/Season 01/Série de Teste - S01E01 - O Começo.webm":  "clip.webm",
	"Shows/Série de Teste/Season 01/Série de Teste - S01E02 - O Segundo.webm": "clip.webm",
	"musicas/Banda Teste/Disco Teste/01 - Bip.mp3":                            "tone.mp3",
	"jogos/roms/snes/Super Teste (USA).sfc":                                   "",
	"jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 1).cue":           "",
	"jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 1).bin":           "",
	"jogos/bios/psx/scph1001.bin":                                             "",
}

type server struct {
	base string
}

func start(t *testing.T) server {
	t.Helper()
	root := t.TempDir()
	testdata, _ := filepath.Abs("../../../server/internal/testkit/testdata")
	for rel, src := range tree {
		dst := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(dst), 0o755)
		data := []byte("rom: " + rel)
		if src != "" {
			var err error
			if data, err = os.ReadFile(filepath.Join(testdata, src)); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, "arkflix-server")
	build := exec.Command("go", "build", "-o", bin, "./cmd/arkflix-server")
	build.Dir = "../../../server"
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	args := []string{"--addr", addr, "--data", filepath.Join(root, "data"), "--movies", filepath.Join(root, "Movies"),
		"--shows", filepath.Join(root, "Shows"), "--music", filepath.Join(root, "musicas"), "--games", filepath.Join(root, "jogos")}
	add := exec.Command(bin, append(args, "user", "add", "stanley", "--admin")...)
	add.Stdin = strings.NewReader("senha123\n")
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("user add: %v\n%s", err, out)
	}
	if out, err := exec.Command(bin, append(args, "scan")...).CombinedOutput(); err != nil {
		t.Fatalf("scan: %v\n%s", err, out)
	}
	srv := exec.Command(bin, args...)
	logs := &strings.Builder{}
	srv.Stdout, srv.Stderr = logs, logs
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		srv.Process.Kill()
		srv.Wait()
		if t.Failed() {
			t.Log(logs.String())
		}
	})
	base := "http://" + addr
	for i := 0; ; i++ {
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		if i > 100 {
			t.Fatal("servidor não subiu")
		}
		time.Sleep(50 * time.Millisecond)
	}
	return server{base: base}
}

func TestJellyfinClient(t *testing.T) {
	s := start(t)
	ctx := context.Background()
	if _, err := jellyfin.Login(ctx, s.base, "stanley", "errada", jellyfin.Account{DeviceID: "d"}); err == nil {
		t.Fatal("senha errada entrou")
	}
	acc, err := jellyfin.Login(ctx, s.base, "stanley", "senha123", jellyfin.Account{DeviceID: "compat", Device: "teste", Version: "1"})
	if err != nil || acc.Token == "" || acc.UserID == "" {
		t.Fatalf("login: %+v %v", acc, err)
	}
	c := &jellyfin.API{Account: acc}
	movies, total, err := c.Items(ctx, jellyfin.Query{Types: "Movie", Sort: "name"})
	if err != nil || total != 2 || movies[0].Name != "Curta Pixel" {
		t.Fatalf("filmes: %+v %d %v", movies, total, err)
	}
	series, _, err := c.Items(ctx, jellyfin.Query{Types: "Series"})
	if err != nil || len(series) != 1 {
		t.Fatalf("séries: %+v %v", series, err)
	}
	seasons, err := c.Seasons(ctx, series[0].ID)
	if err != nil || len(seasons) != 1 {
		t.Fatalf("temporadas: %+v %v", seasons, err)
	}
	eps, err := c.Episodes(ctx, series[0].ID, seasons[0].ID)
	if err != nil || len(eps) != 2 || eps[1].Name != "O Segundo" {
		t.Fatalf("episódios: %+v %v", eps, err)
	}
	if latest, err := c.Latest(ctx, "Movie", 10); err != nil || len(latest) != 2 {
		t.Fatalf("recentes: %+v %v", latest, err)
	}

	big := movies[1]
	st, err := c.Play(ctx, big.ID, -1, 0)
	if err != nil || st.Path == "" || st.Session == "" {
		t.Fatalf("play: %+v %v", st, err)
	}
	if len(st.Subtitles) != 1 {
		t.Fatalf("legenda externa: %+v", st.Subtitles)
	}
	get := func(p string) (int, string) {
		req, _ := http.NewRequest("GET", s.base+p, nil)
		req.Header.Set("Authorization", jellyfin.Authorization(acc))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if code, body := get(st.Subtitles[0].URL); code != 200 || !strings.Contains(body, "WEBVTT") {
		t.Fatalf("legenda: %d %q", code, body)
	}
	if code, body := get(st.Path); code != 200 || len(body) == 0 {
		t.Fatalf("vídeo: %d", code)
	}
	if err := c.Report(ctx, "start", big.ID, st.Session, st.Source, 0, false); err != nil {
		t.Fatal(err)
	}
	if err := c.Report(ctx, "stop", big.ID, st.Session, st.Source, 1.0, false); err != nil {
		t.Fatal(err)
	}
	resume, err := c.Resume(ctx, 10)
	if err != nil || len(resume) != 1 || resume[0].ID != big.ID || resume[0].Position < 0.9 {
		t.Fatalf("continuar: %+v %v", resume, err)
	}
	if err := c.SetFavorite(ctx, big.ID, true); err != nil {
		t.Fatal(err)
	}
	if favs, _, _ := c.Items(ctx, jellyfin.Query{Types: "Movie,Series", Favorite: true}); len(favs) != 1 {
		t.Fatalf("favoritos: %+v", favs)
	}
	if err := c.SetPlayed(ctx, eps[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if next, err := c.NextUp(ctx, 10); err != nil || len(next) != 1 || next[0].ID != eps[1].ID {
		t.Fatalf("próximo: %+v %v", next, err)
	}
	if err := c.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Resume(ctx, 1); err != jellyfin.ErrAuth {
		t.Fatalf("token depois do logout: %v", err)
	}
}

func TestRomMClient(t *testing.T) {
	s := start(t)
	ctx := context.Background()
	if _, err := romm.Login(ctx, s.base, "stanley", "errada", "pc"); err == nil {
		t.Fatal("senha errada entrou")
	}
	acc, err := romm.Login(ctx, s.base, "stanley", "senha123", "pc de teste")
	if err != nil || !strings.HasPrefix(acc.Token, "rmm_") {
		t.Fatalf("login: %+v %v", acc, err)
	}
	c := &romm.Client{Base: acc.Base, Token: acc.Token}
	if me, err := c.Me(ctx); err != nil || me != "stanley" {
		t.Fatalf("me: %q %v", me, err)
	}
	roms, err := c.Roms(ctx)
	if err != nil || len(roms) != 2 {
		t.Fatalf("roms: %+v %v", roms, err)
	}
	var single, multi romm.Rom
	for _, r := range roms {
		if r.Multi {
			multi = r
		} else {
			single = r
		}
	}
	if single.Platform != "snes" || single.FsName != "Super Teste (USA).sfc" || multi.Platform != "psx" {
		t.Fatalf("roms: %+v", roms)
	}
	dir := t.TempDir()
	path, err := c.Download(ctx, single, dir, func(int64, int64) {})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "Super Teste") {
		t.Fatalf("conteúdo: %q", b)
	}
	full, err := c.Rom(ctx, multi.ID)
	if err != nil || len(full.Files) != 2 {
		t.Fatalf("rom multi: %+v %v", full, err)
	}
	path, err = c.Download(ctx, full, dir, func(int64, int64) {})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "Jogo Disco (USA) (Disc 1).cue")); err != nil {
		t.Fatalf("multidisco: %v", err)
	}

	save := filepath.Join(dir, "Super Teste (USA).srm")
	os.WriteFile(save, []byte("save 1"), 0o644)
	if err := c.Upload(ctx, romm.Saves, single.ID, "snes9x", save, ""); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(save, []byte("save 2, maior"), 0o644)
	if err := c.Upload(ctx, romm.Saves, single.ID, "snes9x", save, ""); err != nil {
		t.Fatal(err)
	}
	assets, err := c.Assets(ctx, romm.Saves, single.ID)
	if err != nil || len(assets) != 1 || assets[0].Size != int64(len("save 2, maior")) || assets[0].Updated.IsZero() {
		t.Fatalf("saves: %+v %v", assets, err)
	}
	got := filepath.Join(dir, "baixado.srm")
	if err := c.Fetch(ctx, romm.Saves, assets[0], got); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(got); string(b) != "save 2, maior" {
		t.Fatalf("save baixado: %q", b)
	}
	if states, err := c.Assets(ctx, romm.States, single.ID); err != nil || len(states) != 0 {
		t.Fatalf("states: %+v %v", states, err)
	}

	fws, err := c.Firmwares(ctx)
	if err != nil || len(fws) != 1 || fws[0].Platform != "psx" {
		t.Fatalf("firmware: %+v %v", fws, err)
	}
	if err := c.FetchFirmware(ctx, fws[0], filepath.Join(dir, fws[0].FileName)); err != nil {
		t.Fatal(err)
	}
}
