package core_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/core"
	"github.com/stanleygomes/arkflix/app/internal/fixture"
	"github.com/stanleygomes/arkflix/app/internal/profiles"
)

func musicSetup(t *testing.T) (harness, *fixture.FakeSubsonic, *httptest.Server) {
	t.Helper()
	h := setup(t)
	fake := fixture.NewFakeSubsonic()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	if _, err := h.svc.ConnectMusic(profiles.Main, srv.URL, "stanley", "senha123", ""); err != nil {
		t.Fatal(err)
	}
	return h, fake, srv
}

func TestMusicLink(t *testing.T) {
	h := setup(t)
	fake := fixture.NewFakeSubsonic()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	if _, err := h.svc.MusicHome(); err == nil || !strings.Contains(err.Error(), "vincule") {
		t.Fatalf("sem conta: %v", err)
	}
	if _, err := h.svc.ConnectMusic(profiles.Main, srv.URL, "stanley", "errada", ""); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	info, err := h.svc.ConnectMusic(profiles.Main, srv.URL, "stanley", "senha123", "")
	if err != nil || !info.Connected || info.Server != "navidrome" {
		t.Fatalf("vincular: %+v %v", info, err)
	}
	file := filepath.Join(h.env.Data, "subsonic.json")
	raw, _ := os.ReadFile(file)
	if strings.Contains(string(raw), "senha123") || !strings.Contains(string(raw), `"salt"`) {
		t.Fatalf("guarda o par salgado, nunca a senha: %s", raw)
	}
	if st, _ := os.Stat(file); st.Mode().Perm() != 0o600 {
		t.Fatalf("subsonic.json legível por outros: %v", st.Mode())
	}
	if b, _ := h.svc.Bootstrap(); !b.Music.Connected {
		t.Fatal("o boot conta que o perfil tem conta de músicas")
	}
	if acc, _ := h.svc.ProfileAccounts(profiles.Main); !acc.Music.Connected || acc.Music.User != "stanley" {
		t.Fatalf("contas: %+v", acc)
	}
	srvInfo := h.svc.MusicServerInfo()
	if !srvInfo.Admin || srvInfo.Ark {
		t.Fatalf("servidor: %+v", srvInfo)
	}

	// perfil infantil: ouve com a conta que o adulto vinculou, não mexe
	kid, err := h.svc.CreateProfile("Kid", "kirby", "#3a86d8")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.ConnectMusic(kid.ID, srv.URL, "ana", "senha123", ""); err != nil {
		t.Fatal(err)
	}
	h.svc.SetProfileKids(kid.ID, true)
	if _, err := h.svc.SelectProfile(kid.ID, ""); err != nil {
		t.Fatal(err)
	}
	if home, err := h.svc.MusicHome(); err != nil || len(home.Newest) != 3 {
		t.Fatalf("infantil ouve: %+v %v", home, err)
	}
	for name, fn := range map[string]func() error{
		"criar playlist":  func() error { _, err := h.svc.MusicCreatePlaylist("x", nil); return err },
		"apagar playlist": func() error { return h.svc.MusicDeletePlaylist("pl1") },
		"scan":            h.svc.MusicScan,
		"baixar":          func() error { return h.svc.MusicDownload("album", "al1") },
		"desvincular":     func() error { return h.svc.DisconnectMusic(kid.ID) },
	} {
		if err := fn(); err != core.ErrKids {
			t.Fatalf("%s no infantil: %v", name, err)
		}
	}
	if s := h.svc.MusicServerInfo(); s.Admin {
		t.Fatal("infantil nunca é admin")
	}
	if _, err := h.svc.SelectProfile(profiles.Main, ""); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DisconnectMusic(profiles.Main); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) || h.svc.MusicStatus().Connected {
		t.Fatal("desvincular apaga o arquivo")
	}
}

func TestMusicBrowseAndPlay(t *testing.T) {
	h, fake, _ := musicSetup(t)
	home, err := h.svc.MusicHome()
	if err != nil || len(home.Newest) != 3 || len(home.Frequent) != 2 || len(home.Starred) != 1 || home.Newest[0].Name != "Ao Vivo no Teste" {
		t.Fatalf("início: %+v %v", home, err)
	}
	cv := home.Newest[0].Cover
	if !strings.HasPrefix(cv, "http://127.0.0.1:") || strings.Contains(cv, "t=") {
		t.Fatalf("capa pelo proxy, sem a conta: %q", cv)
	}
	resp, err := http.Get(cv)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("capa: %v", err)
	}
	resp.Body.Close()
	idx, err := h.svc.MusicArtists()
	if err != nil || len(idx) != 2 || idx[1].Artists[0].Name != "Orquestra Pixel" {
		t.Fatalf("artistas: %+v %v", idx, err)
	}
	page, err := h.svc.MusicAlbums("name", 0, 2)
	if err != nil || len(page.Albums) != 2 || !page.More || page.Albums[0].Name != "Ao Vivo no Teste" {
		t.Fatalf("álbuns: %+v %v", page, err)
	}
	if _, err := h.svc.MusicAlbums("cor", 0, 2); err == nil {
		t.Fatal("ordem desconhecida")
	}
	ar, err := h.svc.MusicArtist("ar2")
	if err != nil || len(ar.Albums) != 1 || len(ar.Top) != 2 {
		t.Fatalf("artista: %+v %v", ar, err)
	}
	al, err := h.svc.MusicAlbum("al1")
	if err != nil || len(al.Tracks) != 2 || !al.Tracks[1].Starred || al.Tracks[0].Local {
		t.Fatalf("álbum: %+v %v", al, err)
	}
	if found, err := h.svc.MusicSearch("pixel"); err != nil || len(found.Artists) != 1 {
		t.Fatalf("busca: %+v %v", found, err)
	}
	if err := h.svc.MusicStar("track", "t1", true); err != nil || !fake.Starred("t1") {
		t.Fatalf("estrela: %v", err)
	}
	st, err := h.svc.MusicStarred()
	if err != nil || len(st.Tracks) != 2 || len(st.Albums) != 1 {
		t.Fatalf("favoritas: %+v %v", st, err)
	}

	s, err := h.svc.MusicPlay("t3")
	if err != nil || s.Local || !strings.Contains(s.URL, "format=raw") || !strings.Contains(s.Fallback, "format=mp3") {
		t.Fatalf("tocar: %+v %v", s, err)
	}
	resp, err = http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(data) != len(fixture.ToneFLAC) || resp.Header.Get("Content-Type") != "audio/flac" {
		t.Fatalf("áudio pelo proxy: %d %s", len(data), resp.Header.Get("Content-Type"))
	}
	if err := h.svc.MusicScrobble("t3", false); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MusicScrobble("t3", true); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.Scrobbles(), ","); got != "now t3,play t3" {
		t.Fatalf("scrobbles: %s", got)
	}
	if err := h.svc.MusicSaveQueue([]string{"t1", "t3"}, "t3", 1.2); err != nil {
		t.Fatal(err)
	}
	q, err := h.svc.MusicResume()
	if err != nil || len(q.Tracks) != 2 || q.Current != 1 || q.Position != 1.2 {
		t.Fatalf("fila: %+v %v", q, err)
	}
	for _, v := range fake.Queries() {
		if v.Has("p") {
			t.Fatalf("a senha foi pro servidor depois do login: %v", v)
		}
	}
}

func TestMusicPlaylists(t *testing.T) {
	h, fake, _ := musicSetup(t)
	lists, err := h.svc.MusicPlaylists()
	if err != nil || len(lists) != 1 || !lists[0].Mine || lists[0].Songs != 2 {
		t.Fatalf("playlists: %+v %v", lists, err)
	}
	p, err := h.svc.MusicCreatePlaylist(" Treino ", []string{"t1"})
	if err != nil || p.Name != "Treino" || len(p.Tracks) != 1 {
		t.Fatalf("criar: %+v %v", p, err)
	}
	if _, err := h.svc.MusicCreatePlaylist("  ", nil); err == nil {
		t.Fatal("sem nome")
	}
	if err := h.svc.MusicAddAlbumToPlaylist(p.ID, "al3"); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MusicAddToPlaylist(p.ID, []string{"t3"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.PlaylistSongs(p.ID), ","); got != "t1,t4,t5,t3" {
		t.Fatalf("faixas: %s", got)
	}
	if err := h.svc.MusicRemoveFromPlaylist(p.ID, []int{1}); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MusicReorderPlaylist(p.ID, []string{"t3", "t1", "t5"}); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MusicRenamePlaylist(p.ID, "Corrida"); err != nil {
		t.Fatal(err)
	}
	full, err := h.svc.MusicPlaylist(p.ID)
	if err != nil || full.Name != "Corrida" || len(full.Tracks) != 3 || full.Tracks[0].ID != "t3" {
		t.Fatalf("playlist: %+v %v", full, err)
	}
	if err := h.svc.MusicDeletePlaylist(p.ID); err != nil || fake.PlaylistSongs(p.ID) != nil {
		t.Fatalf("apagar: %v", err)
	}
}

func TestMusicAdmin(t *testing.T) {
	core.SetMusicScanPoll(10 * time.Millisecond)
	h, _, _ := musicSetup(t)
	adm, err := h.svc.MusicAdmin()
	if err != nil || len(adm.Users) != 2 || adm.Scan.Count != 5 {
		t.Fatalf("admin: %+v %v", adm, err)
	}
	if err := h.svc.MusicScan(); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "musicscan"); ev.State != "done" || !strings.Contains(ev.Message, "5 faixas") {
		t.Fatalf("scan: %+v", ev)
	}
}

func TestMusicOffline(t *testing.T) {
	h, _, srv := musicSetup(t)
	dir := filepath.Join(h.env.Home, "Music", "arkflix")
	// metade de uma faixa já baixada: continua do .part
	part := filepath.Join(dir, "Banda Teste", "Disco Teste", "01 - Bip.mp3.part")
	os.MkdirAll(filepath.Dir(part), 0o755)
	os.WriteFile(part, fixture.ToneMP3[:500], 0o644)
	if err := h.svc.MusicDownload("album", "al1"); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "musicdl:album:al1"); ev.State != "done" {
		t.Fatalf("baixar álbum: %+v", ev)
	}
	got, err := os.ReadFile(filepath.Join(dir, "Banda Teste", "Disco Teste", "01 - Bip.mp3"))
	if err != nil || string(got) != string(fixture.ToneMP3) {
		t.Fatalf("faixa inteira (continuou do meio): %d %v", len(got), err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Banda Teste", "Disco Teste", "02 - Solo de Teste.mp3")); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MusicDownload("playlist", "pl1"); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "musicdl:playlist:pl1"); ev.State != "done" {
		t.Fatalf("baixar playlist: %+v", ev)
	}
	m3u, err := os.ReadFile(filepath.Join(dir, "Playlists", "Pra Viagem.m3u8"))
	if err != nil || !strings.Contains(string(m3u), "../Orquestra Pixel/Trilhas 8-bit/01 - Tema do Menu.mp3") {
		t.Fatalf("m3u8: %s %v", m3u, err)
	}
	local, err := h.svc.MusicLocal()
	if err != nil || len(local) != 2 || local[0].Kind != "playlist" {
		t.Fatalf("no PC: %+v %v", local, err)
	}
	al, _ := h.svc.MusicAlbum("al1")
	if !al.Album.Local || !al.Tracks[0].Local {
		t.Fatalf("selo No PC: %+v", al)
	}
	s, err := h.svc.MusicPlay("t1")
	if err != nil || !s.Local || !strings.Contains(s.URL, "/local/t1") {
		t.Fatalf("toca do disco: %+v %v", s, err)
	}

	// sem servidor: o álbum baixado abre e toca do disco
	srv.Close()
	off, err := h.svc.MusicAlbum("al1")
	if err != nil || !off.Offline || len(off.Tracks) != 2 || off.Album.Name != "Disco Teste" {
		t.Fatalf("offline: %+v %v", off, err)
	}
	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || len(data) != len(fixture.ToneMP3) {
		t.Fatalf("cópia local sem servidor: %d", resp.StatusCode)
	}

	// apagar o álbum mantém a faixa que a playlist também usa
	if err := h.svc.MusicRemoveLocal("album", "al1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Banda Teste", "Disco Teste", "01 - Bip.mp3")); err != nil {
		t.Fatal("a playlist ainda usa o Bip")
	}
	if _, err := os.Stat(filepath.Join(dir, "Banda Teste", "Disco Teste", "02 - Solo de Teste.mp3")); !os.IsNotExist(err) {
		t.Fatal("o Solo saiu")
	}
	if err := h.svc.MusicRemoveLocal("playlist", "pl1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Banda Teste")); !os.IsNotExist(err) {
		t.Fatal("pastas vazias somem")
	}
	if err := h.svc.MusicRemoveLocal("playlist", "pl1"); err == nil {
		t.Fatal("já não tem cópia")
	}
}
