package core_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/core"
	"github.com/stanleygomes/arkflix/app/internal/fixture"
	"github.com/stanleygomes/arkflix/app/internal/meta"
	"github.com/stanleygomes/arkflix/app/internal/profiles"
)

// arkSetup liga o perfil principal (Jellyfin) e o app (RomM) num arkflix-server falso.
func arkSetup(t *testing.T) (harness, *fixture.FakeArkServer) {
	t.Helper()
	h := setup(t)
	fake := fixture.NewFakeArkServer()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	if info := h.svc.ArkServer(); info.Found {
		t.Fatalf("sem conta não há servidor: %+v", info)
	}
	if _, err := h.svc.ConnectWatch(profiles.Main, srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.ConnectRemote(srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "remote"); ev.State != "done" {
		t.Fatalf("catálogo: %+v", ev)
	}
	return h, fake
}

func TestArkDetectAndAdmin(t *testing.T) {
	core.SetArkScanPoll(50 * time.Millisecond)
	h, fake := arkSetup(t)
	info := h.svc.ArkServer()
	if !info.Found || !info.Admin || !info.Writable || !info.Lists || !info.Videos || !info.Games || info.Source != "jellyfin" || info.User != "stanley" || info.Name != "arkflix de Teste" {
		t.Fatalf("detecção: %+v", info)
	}
	st, err := h.svc.ArkStatus()
	if err != nil || len(st.Libraries) != 4 {
		t.Fatalf("status: %+v %v", st, err)
	}
	if err := h.svc.ArkScan(); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "arkscan"); ev.State != "done" || ev.Message != "1501 itens no servidor" {
		t.Fatalf("scan: %+v", ev)
	}

	u, err := h.svc.ArkCreateUser(" bia ", "segredo1", false)
	if err != nil || u.Name != "bia" {
		t.Fatalf("criar: %+v %v", u, err)
	}
	if _, err := h.svc.ArkCreateUser("ze", "", false); err == nil {
		t.Fatal("sem senha")
	}
	if u, err = h.svc.ArkSetAdmin(u.ID, true); err != nil || !u.Admin {
		t.Fatalf("admin: %+v %v", u, err)
	}
	if _, err := h.svc.ArkSetPassword(u.ID, "outra123"); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.ArkDeleteUser("u1"); err == nil || !strings.Contains(err.Error(), "si mesmo") {
		t.Fatalf("mensagem do servidor: %v", err)
	}
	if err := h.svc.ArkDeleteUser(u.ID); err != nil {
		t.Fatal(err)
	}
	if users, _ := h.svc.ArkUsers(); len(users) != 2 {
		t.Fatalf("usuários: %+v", users)
	}
	if p, err := h.svc.ArkPlatforms(); err != nil || len(p) != 3 || p[1].Name != "PlayStation" {
		t.Fatalf("consoles: %+v %v", p, err)
	}

	// usuário deixou de ser admin: o painel some
	fake.Ark.SetAdmin(false)
	if info := h.svc.ArkServer(); info.Admin {
		t.Fatal("comum não é admin")
	}
	if _, err := h.svc.ArkStatus(); err == nil {
		t.Fatal("status só pra admin")
	}
}

func TestArkUploadAndHide(t *testing.T) {
	h, fake := arkSetup(t)
	file := filepath.Join(t.TempDir(), "Chrono Trigger.sfc")
	os.WriteFile(file, []byte("SNES-CT"), 0o644)
	if err := h.svc.ArkUpload("rom", map[string]string{}, []string{file}); err == nil || !strings.Contains(err.Error(), "console") {
		t.Fatalf("faltou o console: %v", err)
	}
	if err := h.svc.ArkUpload("episode", map[string]string{"series": "Série", "season": "um"}, []string{file}); err == nil || !strings.Contains(err.Error(), "número") {
		t.Fatalf("temporada não numérica: %v", err)
	}
	if err := h.svc.ArkUpload("rom", map[string]string{"platform": "snes"}, []string{file + ".nao"}); err == nil {
		t.Fatal("arquivo que não existe")
	}
	if err := h.svc.ArkUpload("rom", map[string]string{"platform": "snes", "series": "ignorado"}, []string{file}); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "arkupload"); ev.State != "done" || ev.Message != "1 arquivo no servidor" {
		t.Fatalf("envio: %+v", ev)
	}
	if got := fake.Ark.Uploads(); !slices.Equal(got, []string{"rom:platform=snes:Chrono Trigger.sfc:7"}) {
		t.Fatalf("recebido: %v", got)
	}

	if err := h.svc.ArkHide("watch", "m3"); err != nil {
		t.Fatal(err)
	}
	dkc := h.game(t, "Donkey Kong Country")
	if !dkc.OnServer {
		t.Fatal("jogo do servidor")
	}
	if err := h.svc.ArkHide("game", dkc.ID); err != nil {
		t.Fatal(err)
	}
	if got := fake.Ark.Hidden(); !slices.Equal(got, []string{"m3", "rom:1"}) {
		t.Fatalf("escondidos: %v", got)
	}
	if err := h.svc.ArkHide("game", h.game(t, "Mega Man X").ID); err == nil {
		t.Fatal("jogo que só está aqui não tem como esconder no servidor")
	}

	fake.Ark.SetWritable(false)
	if err := h.svc.ArkUpload("rom", map[string]string{"platform": "snes"}, []string{file}); err == nil || !strings.Contains(err.Error(), "só leitura") {
		t.Fatalf("só leitura: %v", err)
	}
	if err := h.svc.ArkHide("watch", "m2"); err == nil {
		t.Fatal("só leitura não esconde")
	}
}

func TestArkLists(t *testing.T) {
	h, fake := arkSetup(t)
	lists, err := h.svc.ArkLists()
	if err != nil || len(lists) != 1 {
		t.Fatalf("listas: %+v %v", lists, err)
	}
	l := lists[0]
	smw, dkc := h.game(t, "Super Mario World"), h.game(t, "Donkey Kong Country")
	if len(l.Videos) != 2 || l.Videos[1].Type != "Series" || !slices.Equal(l.Games, []string{dkc.ID, smw.ID}) {
		t.Fatalf("itens: %+v", l)
	}
	// a capa vem pelo proxy, com o token no cabeçalho
	poster := l.Videos[0].Poster
	if !strings.Contains(poster, "/arkflix/api/items/m1/image?maxWidth=") || strings.Contains(poster, "tok-stanley") {
		t.Fatalf("capa: %s", poster)
	}
	resp, err := http.Get(poster)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(string(body), "\x89PNG") {
		t.Fatalf("capa pelo proxy: %d", resp.StatusCode)
	}
	base := poster[:strings.Index(poster, "/arkflix/")]
	for _, p := range []string{"/arkflix/api/users", "/arkflix/api/items/m1/hide", "/arkflix/api/items/../users/image"} {
		if r, err := http.Get(base + p); err == nil {
			r.Body.Close()
			if r.StatusCode != http.StatusNotFound {
				t.Fatalf("proxy deixou passar %s: %d", p, r.StatusCode)
			}
		}
	}

	marks, err := h.svc.ArkListsFor("watch", "m2")
	if err != nil || len(marks) != 1 || marks[0].Has {
		t.Fatalf("m2 fora: %+v %v", marks, err)
	}
	if err := h.svc.ArkSetInList(l.ID, "watch", "m2", true); err != nil {
		t.Fatal(err)
	}
	if marks, _ = h.svc.ArkListsFor("watch", "m2"); !marks[0].Has {
		t.Fatal("m2 entrou")
	}
	if marks, _ = h.svc.ArkListsFor("game", smw.ID); !marks[0].Has {
		t.Fatal("Super Mario World está na lista")
	}
	if err := h.svc.ArkSetInList(l.ID, "game", smw.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := fake.Ark.ListItems(l.ID); !slices.Equal(got, []string{"m1", "s1", "r1", "t1", "m2"}) {
		t.Fatalf("depois: %v", got)
	}
	n, err := h.svc.ArkCreateList("  Pra Nina ")
	if err != nil || n.Name != "Pra Nina" {
		t.Fatalf("criar: %+v %v", n, err)
	}
	if err := h.svc.ArkSetInList(n.ID, "game", smw.ID, true); err != nil {
		t.Fatal(err)
	}
	if n, err = h.svc.ArkRenameList(n.ID, "Da Nina"); err != nil || n.Name != "Da Nina" {
		t.Fatalf("renomear: %+v %v", n, err)
	}
	if got, err := h.svc.ArkList(n.ID); err != nil || !slices.Equal(got.Games, []string{smw.ID}) {
		t.Fatalf("lista nova: %+v %v", got, err)
	}

	// infantil: vê as listas (só os jogos liberados), não mexe nem administra
	if _, err := h.svc.SaveMeta(smw.ID, meta.Override{KidsOK: true}); err != nil {
		t.Fatal(err)
	}
	kid, _ := h.svc.CreateProfile("Nina", "cat", profiles.Colors[2])
	h.svc.SetProfileKids(kid.ID, true)
	if _, err := h.svc.SelectProfile(kid.ID, ""); err != nil {
		t.Fatal(err)
	}
	info := h.svc.ArkServer()
	if !info.Found || info.Admin || info.Source != "romm" || info.Videos {
		t.Fatalf("infantil: %+v", info)
	}
	got, err := h.svc.ArkList(n.ID)
	if err != nil || !slices.Equal(got.Games, []string{smw.ID}) || len(got.Videos) != 0 {
		t.Fatalf("lista no infantil: %+v %v", got, err)
	}
	if got, _ := h.svc.ArkList(l.ID); len(got.Games) != 0 {
		t.Fatalf("jogo não liberado some: %+v", got)
	}
	for name, err := range map[string]error{
		"status": second(h.svc.ArkStatus()),
		"criar":  second(h.svc.ArkCreateList("x")),
		"pôr":    h.svc.ArkSetInList(n.ID, "game", smw.ID, false),
		"apagar": h.svc.ArkDeleteList(n.ID),
		"enviar": h.svc.ArkUpload("rom", map[string]string{"platform": "snes"}, []string{"/etc/hostname"}),
		"hide":   h.svc.ArkHide("game", smw.ID),
	} {
		if !errors.Is(err, core.ErrKids) {
			t.Fatalf("%s no infantil: %v", name, err)
		}
	}
}

func second[T any](_ T, err error) error { return err }
