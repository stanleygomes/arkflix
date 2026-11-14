package core_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/app/internal/core"
	"github.com/stanleygomes/arkflix/app/internal/fixture"
	"github.com/stanleygomes/arkflix/app/internal/profiles"
)

func TestWatchPerProfile(t *testing.T) {
	h := setup(t)
	fake := fixture.NewFakeJellyfin()
	srv := httptest.NewServer(fake)
	defer srv.Close()

	if _, err := h.svc.WatchHome(); err == nil || !strings.Contains(err.Error(), "vincule") {
		t.Fatalf("sem conta: %v", err)
	}
	if b, _ := h.svc.Bootstrap(); b.Watch.Connected {
		t.Fatal("começa sem conta")
	}
	if _, err := h.svc.ConnectWatch(profiles.Main, srv.URL, "stanley", "errada"); err == nil || !strings.Contains(err.Error(), "senha") {
		t.Fatalf("senha errada: %v", err)
	}
	info, err := h.svc.ConnectWatch(profiles.Main, srv.URL, "stanley", "senha123")
	if err != nil || !info.Connected || info.User != "stanley" || info.Server != "Jellyfin de Teste" {
		t.Fatalf("vincular: %+v %v", info, err)
	}
	file := filepath.Join(h.env.Data, "jellyfin.json")
	raw, _ := os.ReadFile(file)
	if strings.Contains(string(raw), "senha123") || !strings.Contains(string(raw), "tok-stanley") {
		t.Fatalf("guarda o token, nunca a senha: %s", raw)
	}
	if st, _ := os.Stat(file); st.Mode().Perm() != 0o600 {
		t.Fatalf("jellyfin.json legível por outros: %v", st.Mode())
	}
	if b, _ := h.svc.Bootstrap(); !b.Watch.Connected {
		t.Fatal("o boot conta que o perfil tem conta")
	}

	home, err := h.svc.WatchHome()
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Resume) != 1 || home.Resume[0].Progress != 0.25 || len(home.NextUp) != 1 || len(home.Movies) != 3 || len(home.Series) != 1 || len(home.Favorites) != 1 {
		t.Fatalf("início: %+v", home)
	}
	poster := home.Resume[0].Poster
	if !strings.HasPrefix(poster, "http://127.0.0.1:") || strings.Contains(poster, "tok-stanley") || home.Resume[0].Backdrop == "" {
		t.Fatalf("imagens pelo proxy, sem o token: %q", poster)
	}
	resp, err := http.Get(poster)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("capa pelo proxy: %d", resp.StatusCode)
	}

	page, err := h.svc.WatchList("movies", "name", "", 0, 2)
	if err != nil || page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("lista: %+v %v", page, err)
	}
	if _, err := h.svc.WatchList("músicas", "name", "", 0, 10); err == nil {
		t.Fatal("lista desconhecida")
	}
	if found, err := h.svc.WatchSearch("teste"); err != nil || len(found) != 2 {
		t.Fatalf("busca: %+v %v", found, err)
	}

	d, err := h.svc.WatchDetails("s1")
	if err != nil || len(d.Seasons) != 2 || d.Next == nil || d.Next.ID != "e2" {
		t.Fatalf("série: %+v %v", d, err)
	}
	eps, err := h.svc.WatchEpisodes("s1", d.Seasons[1].ID)
	if err != nil || len(eps) != 1 || eps[0].Name != "Recomeço" {
		t.Fatalf("episódios: %+v %v", eps, err)
	}

	st, err := h.svc.WatchPlay("m1", -1, -1)
	if err != nil || st.Start != 1800 || st.HLS || !strings.HasPrefix(st.URL, "http://127.0.0.1:") || !strings.HasPrefix(st.Subtitles[0].URL, "http://127.0.0.1:") {
		t.Fatalf("tocar de onde parou: %+v %v", st, err)
	}
	if st, err := h.svc.WatchPlay("m1", -1, 0); err != nil || st.Start != 0 {
		t.Fatalf("do começo: %+v %v", st, err)
	}
	if st, err := h.svc.WatchPlay("e2", -1, -1); err != nil || st.Next == nil || st.Next.ID != "e3" {
		t.Fatalf("próximo episódio: %+v %v", st.Next, err)
	}
	if _, err := h.svc.WatchPlay("s1", -1, -1); err == nil {
		t.Fatal("série não toca direto")
	}
	if err := h.svc.WatchReport("stop", "m1", st.Session, st.Source, 7000, false); err != nil {
		t.Fatal(err)
	}
	if it, _ := fake.Item("m1"); !it.Played {
		t.Fatal("parou no fim = assistido")
	}
	if v, err := h.svc.WatchFavorite("m3", true); err != nil || !v.Favorite {
		t.Fatalf("favorito: %+v %v", v, err)
	}
	if v, err := h.svc.WatchPlayed("e3", true); err != nil || !v.Played {
		t.Fatalf("assistido: %+v %v", v, err)
	}

	// outro perfil não herda a conta; o adulto vincula pro infantil
	kid, _ := h.svc.CreateProfile("Nina", "mushroom", profiles.Colors[6])
	h.svc.SetProfileKids(kid.ID, true)
	if acc, _ := h.svc.ProfileAccounts(kid.ID); acc.Jellyfin.Connected {
		t.Fatal("conta é por perfil")
	}
	if _, err := h.svc.SelectProfile(kid.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.WatchHome(); err == nil {
		t.Fatal("perfil sem conta não vê os filmes do outro")
	}
	if resp, err := http.Get(poster); err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			t.Fatal("o proxy usa a conta do perfil em uso")
		}
	}
	if _, err := h.svc.ConnectWatch(kid.ID, srv.URL, "stanley", "senha123"); err != core.ErrKids {
		t.Fatalf("infantil não vincula sozinho: %v", err)
	}
	if err := h.svc.DisconnectWatch(profiles.Main); err != core.ErrKids {
		t.Fatalf("infantil não desvincula: %v", err)
	}
	h.svc.SelectProfile(profiles.Main, "")
	if _, err := h.svc.ConnectWatch(kid.ID, srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	if acc, _ := h.svc.ProfileAccounts(kid.ID); !acc.Jellyfin.Connected {
		t.Fatal("o adulto vincula a conta do infantil")
	}

	if err := h.svc.DisconnectWatch(profiles.Main); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("desvincular apaga o arquivo")
	}
	if fake.Logouts() != 1 || h.svc.WatchStatus().Connected {
		t.Fatalf("desvincular invalida o token: %d", fake.Logouts())
	}
	if acc, _ := h.svc.ProfileAccounts(kid.ID); !acc.Jellyfin.Connected {
		t.Fatal("o outro perfil continua vinculado")
	}
	if _, err := h.svc.ProfileAccounts("nao-existe"); err == nil {
		t.Fatal("perfil desconhecido")
	}
}

func TestLinkRomM(t *testing.T) {
	h := setup(t)
	bia, _ := h.svc.CreateProfile("Bia", "cat", profiles.Colors[2])
	if _, err := h.svc.LinkRomM(bia.ID); err == nil || !strings.Contains(err.Error(), "RomM") {
		t.Fatalf("sem servidor: %v", err)
	}
	srv := httptest.NewServer(fixture.NewFakeRomM())
	defer srv.Close()
	if _, err := h.svc.ConnectRemote(srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	h.rec.waitTask(t, "remote")
	acc, err := h.svc.LinkRomM(bia.ID)
	if err != nil || !acc.RomM.Mine || acc.RomM.Owner != "Bia" {
		t.Fatalf("saves passam pra Bia: %+v %v", acc, err)
	}
	if main, _ := h.svc.ProfileAccounts(profiles.Main); main.RomM.Mine || !main.RomM.Connected {
		t.Fatalf("principal deixa de sincronizar: %+v", main)
	}
	raw, _ := os.ReadFile(filepath.Join(h.env.Data, "romm.json"))
	if !strings.Contains(string(raw), bia.ID) {
		t.Fatalf("dono gravado: %s", raw)
	}
	if s := h.svc.RemoteStatus(); s.Mine || s.Owner != "Bia" || !slices.Contains([]string{"Bia"}, s.Owner) {
		t.Fatalf("status: %+v", s)
	}
}
