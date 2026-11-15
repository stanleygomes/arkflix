//go:build compat

// Package compat roda os clientes do app (Jellyfin e RomM) contra o
// arkflix-server de verdade: compila o servidor, monta uma biblioteca com
// os arquivos de teste dele e faz as mesmas chamadas que o app faz.
//
//	cd app && go test -tags compat ./internal/compat/
package compat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/arkserver"
	"github.com/stanleygomes/arkflix/app/internal/jellyfin"
	"github.com/stanleygomes/arkflix/app/internal/romm"
	"github.com/stanleygomes/arkflix/app/internal/subsonic"
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

func start(t *testing.T, extra ...string) server {
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
	srv := exec.Command(bin, append(args, extra...)...)
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

func TestSubsonic(t *testing.T) {
	s := start(t)
	get := func(method string, q string) (int, string) {
		salt := "c19b2d"
		sum := md5.Sum([]byte("senha123" + salt))
		u := s.base + "/rest/" + method + ".view?u=stanley&s=" + salt + "&t=" + hex.EncodeToString(sum[:]) + "&v=1.16.1&c=compat&f=json&" + q
		resp, err := http.Get(u)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if _, body := get("ping", ""); !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, `"openSubsonic":true`) {
		t.Fatalf("ping: %s", body)
	}
	_, body := get("search3", "query=")
	if !strings.Contains(body, `"title":"Bip"`) {
		t.Fatalf("search3: %s", body)
	}
	id := body[strings.Index(body, `"song":[{"id":"`)+15:]
	id = id[:strings.Index(id, `"`)]
	if code, audio := get("stream", "id="+id+"&format=raw"); code != 200 || len(audio) < 1000 {
		t.Fatalf("stream: %d %d", code, len(audio))
	}
}

// TestSubsonicClient: o cliente de músicas do app (o mesmo que fala com o
// Navidrome) contra a API Subsonic do servidor.
func TestSubsonicClient(t *testing.T) {
	s := start(t)
	ctx := context.Background()
	if _, err := subsonic.Login(ctx, s.base, "stanley", "errada", ""); err == nil {
		t.Fatal("senha errada devia falhar")
	}
	acc, err := subsonic.Login(ctx, s.base, "stanley", "senha123", "")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Server != "arkflix" || acc.Token == "" || !acc.Has("apiKeyAuthentication") {
		t.Fatalf("conta: %+v", acc)
	}
	c := &subsonic.API{Account: acc}
	if u, err := c.GetUser(ctx, "stanley"); err != nil || !u.AdminRole {
		t.Fatalf("usuário: %+v %v", u, err)
	}
	idx, err := c.Artists(ctx)
	if err != nil || len(idx) != 1 || idx[0].Artists[0].Name != "Banda Teste" {
		t.Fatalf("artistas: %+v %v", idx, err)
	}
	ar, albums, err := c.Artist(ctx, idx[0].Artists[0].ID)
	if err != nil || ar.Name != "Banda Teste" || len(albums) != 1 || albums[0].Name != "Disco Teste" {
		t.Fatalf("artista: %+v %+v %v", ar, albums, err)
	}
	al, songs, err := c.Album(ctx, albums[0].ID)
	if err != nil || al.SongCount != 1 || len(songs) != 1 || songs[0].Title != "Bip" || songs[0].Suffix != "mp3" {
		t.Fatalf("álbum: %+v %+v %v", al, songs, err)
	}
	song := songs[0]
	if list, err := c.AlbumList2(ctx, "newest", 10, 0); err != nil || len(list) != 1 {
		t.Fatalf("novos: %+v %v", list, err)
	}
	if found, err := c.Search(ctx, "bip", 10); err != nil || len(found.Songs) != 1 {
		t.Fatalf("busca: %+v %v", found, err)
	}
	if _, err := c.TopSongs(ctx, ar.Name, 5); err != nil {
		t.Fatal(err)
	}

	// áudio pelo proxy do app, com Range (o <audio> pede assim)
	p := &subsonic.Proxy{Account: func() (subsonic.Account, bool) { return acc, true }}
	defer p.Close()
	base, err := p.URL()
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, base+"/rest/stream?id="+song.ID+"&format=raw", nil)
	req.Header.Set("Range", "bytes=0-")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	audio, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode/100 != 2 || len(audio) < 1000 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "audio/") {
		t.Fatalf("stream: %d %d %s", resp.StatusCode, len(audio), resp.Header.Get("Content-Type"))
	}
	if r, err := http.Get(base + "/rest/getCoverArt?id=" + al.CoverArt + "&size=300"); err != nil {
		t.Fatal(err)
	} else {
		r.Body.Close()
	}
	var buf strings.Builder
	if n, err := c.Download(ctx, song.ID, 0, &buf, nil); err != nil || buf.Len() != len(audio) || (n > 0 && n != int64(len(audio))) {
		t.Fatalf("download: %d %d %v", n, buf.Len(), err)
	}

	// estrela, scrobble e fila
	if err := c.Star(ctx, "song", song.ID, true); err != nil {
		t.Fatal(err)
	}
	if st, err := c.Starred(ctx); err != nil || len(st.Songs) != 1 {
		t.Fatalf("favoritas: %+v %v", st, err)
	}
	if err := c.Star(ctx, "album", al.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := c.Scrobble(ctx, song.ID, false, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Scrobble(ctx, song.ID, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Song(ctx, song.ID); got.PlayCount != 1 {
		t.Fatalf("o scrobble conta a execução: %+v", got)
	}
	if err := c.SaveQueue(ctx, []string{song.ID}, song.ID, 1500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if q, err := c.Queue(ctx); err != nil || q.Current != song.ID || q.Position != 1500 {
		t.Fatalf("fila: %+v %v", q, err)
	}

	// playlist: criar, renomear, pôr, tirar, reordenar e apagar
	pl, err := c.CreatePlaylist(ctx, "Compat", []string{song.ID})
	if err != nil || pl.ID == "" || len(pl.Entry) != 1 {
		t.Fatalf("criar playlist: %+v %v", pl, err)
	}
	name := "Compat 2"
	if err := c.UpdatePlaylist(ctx, pl.ID, subsonic.PlaylistChange{Name: &name, Add: []string{song.ID}}); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Playlist(ctx, pl.ID); err != nil || got.Name != "Compat 2" || len(got.Entry) != 2 {
		t.Fatalf("playlist depois de mexer: %+v %v", got, err)
	}
	if err := c.UpdatePlaylist(ctx, pl.ID, subsonic.PlaylistChange{Remove: []int{0, 1}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Playlist(ctx, pl.ID); len(got.Entry) != 0 {
		t.Fatalf("tirar: %+v", got)
	}
	if err := c.SetPlaylistSongs(ctx, pl.ID, []string{song.ID}); err != nil {
		t.Fatal(err)
	}
	if lists, err := c.Playlists(ctx); err != nil || len(lists) != 1 || lists[0].SongCount != 1 {
		t.Fatalf("playlists: %+v %v", lists, err)
	}
	if err := c.DeletePlaylist(ctx, pl.ID); err != nil {
		t.Fatal(err)
	}

	// admin: scan e usuários
	if _, err := c.StartScan(ctx); err != nil {
		t.Fatal(err)
	}
	if st, err := c.Scan(ctx); err != nil || st.LastScan == "" {
		t.Fatalf("scan: %+v %v", st, err)
	}
	if users, err := c.Users(ctx); err != nil || len(users) != 1 {
		t.Fatalf("usuários: %+v %v", users, err)
	}
	if strings.Contains(fmt.Sprintf("%+v", acc), "senha123") {
		t.Fatal("a conta guardada tem a senha")
	}
}

func TestArkServerClient(t *testing.T) {
	s := start(t, "--writable")
	ctx := context.Background()
	info, err := arkserver.GetInfo(ctx, nil, s.base)
	if err != nil || info.Product != "arkflix-server" || !info.Writable || !info.Has("lists") {
		t.Fatalf("info: %+v %v", info, err)
	}
	acc, err := jellyfin.Login(ctx, s.base, "stanley", "senha123", jellyfin.Account{DeviceID: "compat-ark", Device: "teste", Version: "1"})
	if err != nil {
		t.Fatal(err)
	}
	c := &arkserver.Client{Base: s.base, Token: acc.Token, Jellyfin: true}
	me, err := c.Me(ctx)
	if err != nil || !me.Admin || me.Name != "stanley" {
		t.Fatalf("me: %+v %v", me, err)
	}
	st, err := c.Status(ctx)
	if err != nil || len(st.Libraries) != 4 || !st.Writable {
		t.Fatalf("status: %+v %v", st, err)
	}
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	plats, err := c.Platforms(ctx)
	if err != nil || len(plats) != 2 {
		t.Fatalf("consoles: %+v %v", plats, err)
	}
	users, err := c.Users(ctx)
	if err != nil || len(users) != 1 {
		t.Fatalf("usuários: %+v %v", users, err)
	}
	bia, err := c.CreateUser(ctx, "bia", "senha-bia", false)
	if err != nil || bia.Admin {
		t.Fatalf("criar: %+v %v", bia, err)
	}
	if _, err := c.UpdateUser(ctx, bia.ID, arkserver.UserPatch{}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteUser(ctx, bia.ID); err != nil {
		t.Fatal(err)
	}

	// listas: filme pelo id do Jellyfin e jogo pelo id do RomM
	jf := &jellyfin.API{Account: acc}
	movies, _, err := jf.Items(ctx, jellyfin.Query{Types: "Movie", Sort: "name"})
	if err != nil || len(movies) == 0 {
		t.Fatal(err)
	}
	racc, err := romm.Login(ctx, s.base, "stanley", "senha123", "pc")
	if err != nil {
		t.Fatal(err)
	}
	roms, err := (&romm.Client{Base: racc.Base, Token: racc.Token}).Roms(ctx)
	if err != nil || len(roms) == 0 {
		t.Fatal(err)
	}
	l, err := c.CreateList(ctx, "Fim de semana")
	if err != nil {
		t.Fatal(err)
	}
	l, err = c.AddToList(ctx, l.ID, movies[0].ID, fmt.Sprintf("rom:%d", roms[0].ID))
	if err != nil || l.Count != 2 {
		t.Fatalf("adicionar: %+v %v", l, err)
	}
	full, err := c.List(ctx, l.ID)
	if err != nil || len(full.Items) != 2 || full.Items[0].ID != movies[0].ID || full.Items[1].RID != roms[0].ID {
		t.Fatalf("lista: %+v %v", full, err)
	}
	if all, err := c.Lists(ctx); err != nil || len(all) != 1 {
		t.Fatalf("listas: %+v %v", all, err)
	}
	if err := c.RemoveFromList(ctx, l.ID, full.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RenameList(ctx, l.ID, "Sábado"); err != nil {
		t.Fatal(err)
	}

	// envio e esconder: o arquivo vai pra pasta certa e some do acervo
	dir := t.TempDir()
	rom := filepath.Join(dir, "Novo Jogo (USA).sfc")
	os.WriteFile(rom, []byte("rom nova"), 0o644)
	saved, err := c.Upload(ctx, map[string]string{"kind": "rom", "platform": "snes"}, []string{rom}, func(int64, int64) {})
	if err != nil || len(saved) != 1 || !strings.Contains(saved[0], "roms/snes/Novo Jogo (USA).sfc") {
		t.Fatalf("envio: %v %v", saved, err)
	}
	if err := c.Hide(ctx, fmt.Sprintf("rom:%d", roms[0].ID)); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteList(ctx, l.ID); err != nil {
		t.Fatal(err)
	}
}
