package arkserver_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/app/internal/arkserver"
	"github.com/stanleygomes/arkflix/app/internal/fixture"
)

func fake(t *testing.T) (*fixture.FakeArk, *arkserver.Client) {
	t.Helper()
	f := fixture.NewFakeArk()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, &arkserver.Client{Base: srv.URL, Token: "tok-stanley", Jellyfin: true}
}

func TestInfo(t *testing.T) {
	_, c := fake(t)
	info, err := arkserver.GetInfo(context.Background(), nil, c.Base+"/")
	if err != nil || info.Product != arkserver.Product || !info.Has("lists") || info.Has("nada") {
		t.Fatalf("info: %+v %v", info, err)
	}
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"product":"jellyfin"}`))
	}))
	defer other.Close()
	if _, err := arkserver.GetInfo(context.Background(), nil, other.URL); err == nil {
		t.Fatal("outro produto não é arkflix-server")
	}
	none := httptest.NewServer(http.NotFoundHandler())
	defer none.Close()
	if _, err := arkserver.GetInfo(context.Background(), nil, none.URL); err == nil {
		t.Fatal("sem a rota não é arkflix-server")
	}
}

func TestAuth(t *testing.T) {
	f, c := fake(t)
	ctx := context.Background()
	if me, err := c.Me(ctx); err != nil || me.Name != "stanley" || !me.Admin {
		t.Fatalf("me pelo token do Jellyfin: %+v %v", me, err)
	}
	romm := &arkserver.Client{Base: c.Base, Token: "token-do-app"}
	if _, err := romm.Me(ctx); err != nil {
		t.Fatalf("me pelo token do RomM: %v", err)
	}
	if got := f.Auths(); !slices.Equal(got, []string{"jellyfin", "romm"}) {
		t.Fatalf("cabeçalhos: %v", got)
	}
	bad := &arkserver.Client{Base: c.Base, Token: "x", Jellyfin: true}
	if _, err := bad.Me(ctx); !errors.Is(err, arkserver.ErrAuth) {
		t.Fatalf("token errado: %v", err)
	}
	f.SetAdmin(false)
	_, err := c.Status(ctx)
	var e *arkserver.Error
	if !errors.As(err, &e) || e.Status != http.StatusForbidden || e.Message != "só administradores" {
		t.Fatalf("usuário comum no admin: %v", err)
	}
}

func TestAdmin(t *testing.T) {
	_, c := fake(t)
	ctx := context.Background()
	st, err := c.Status(ctx)
	if err != nil || len(st.Libraries) != 4 || st.Libraries[1].Episodes != 420 || st.Libraries[2].Online || st.Metadata.Pending != 12 || len(st.Sessions) != 1 {
		t.Fatalf("status: %+v %v", st, err)
	}
	if err := c.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if st, _ := c.Status(ctx); !st.Scan.Running {
		t.Fatal("scan devia estar rodando")
	}
	if p, err := c.Platforms(ctx); err != nil || len(p) != 3 || p[0].Slug != "snes" {
		t.Fatalf("consoles: %+v %v", p, err)
	}
	u, err := c.CreateUser(ctx, "bia", "segredo1", true)
	if err != nil || u.Name != "bia" || !u.Admin {
		t.Fatalf("criar: %+v %v", u, err)
	}
	if _, err := c.CreateUser(ctx, "Bia", "segredo1", false); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Fatalf("repetido: %v", err)
	}
	no := false
	if u, err = c.UpdateUser(ctx, u.ID, arkserver.UserPatch{Admin: &no}); err != nil || u.Admin {
		t.Fatalf("tirar admin: %+v %v", u, err)
	}
	if err := c.DeleteUser(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteUser(ctx, "u1"); err == nil || !strings.Contains(err.Error(), "si mesmo") {
		t.Fatalf("apagar a si mesmo: %v", err)
	}
	if users, _ := c.Users(ctx); len(users) != 2 {
		t.Fatalf("usuários: %+v", users)
	}
}

func TestUpload(t *testing.T) {
	f, c := fake(t)
	dir := t.TempDir()
	a, b := filepath.Join(dir, "Chrono Trigger.sfc"), filepath.Join(dir, "Earthbound.sfc")
	os.WriteFile(a, []byte("0123456789"), 0o644)
	os.WriteFile(b, []byte("abc"), 0o644)
	var last, total int64
	saved, err := c.Upload(context.Background(), map[string]string{"kind": "rom", "platform": "snes"}, []string{a, b}, func(d, t int64) { last, total = d, t })
	if err != nil || len(saved) != 2 {
		t.Fatalf("envio: %v %v", saved, err)
	}
	if last != 13 || total != 13 {
		t.Fatalf("progresso: %d de %d", last, total)
	}
	want := []string{"rom:platform=snes:Chrono Trigger.sfc:10", "rom:platform=snes:Earthbound.sfc:3"}
	if got := f.Uploads(); !slices.Equal(got, want) {
		t.Fatalf("recebido: %v", got)
	}
	f.SetWritable(false)
	if _, err := c.Upload(context.Background(), map[string]string{"kind": "rom", "platform": "snes"}, []string{a}, nil); err == nil || !strings.Contains(err.Error(), "só leitura") {
		t.Fatalf("só leitura: %v", err)
	}
	if _, err := c.Upload(context.Background(), map[string]string{"kind": "rom"}, []string{dir}, nil); err == nil {
		t.Fatal("pasta não é arquivo")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f.SetWritable(true)
	if _, err := c.Upload(ctx, map[string]string{"kind": "rom", "platform": "snes"}, []string{a}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelado: %v", err)
	}
}

func TestLists(t *testing.T) {
	f, c := fake(t)
	ctx := context.Background()
	lists, err := c.Lists(ctx)
	if err != nil || len(lists) != 1 || lists[0].Count != 5 || lists[0].Items != nil {
		t.Fatalf("listas: %+v %v", lists, err)
	}
	l, err := c.List(ctx, lists[0].ID)
	if err != nil || len(l.Items) != 5 || l.Items[2].Kind != "rom" || l.Items[2].RID != 1 || l.Items[0].Image != "/arkflix/api/items/m1/image" {
		t.Fatalf("itens: %+v %v", l, err)
	}
	n, err := c.CreateList(ctx, "Pra Nina")
	if err != nil || n.Name != "Pra Nina" {
		t.Fatalf("criar: %+v %v", n, err)
	}
	if _, err := c.AddToList(ctx, n.ID, "m2", "rom:3", "m2"); err != nil {
		t.Fatal(err)
	}
	if got := f.ListItems(n.ID); !slices.Equal(got, []string{"m2", "r3"}) {
		t.Fatalf("itens novos: %v", got)
	}
	if err := c.RemoveFromList(ctx, n.ID, "r3"); err != nil {
		t.Fatal(err)
	}
	if n, err = c.RenameList(ctx, n.ID, "Da Nina"); err != nil || n.Name != "Da Nina" {
		t.Fatalf("renomear: %+v %v", n, err)
	}
	if err := c.DeleteList(ctx, n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(ctx, n.ID); err == nil {
		t.Fatal("lista apagada")
	}
	if err := c.Hide(ctx, "m3"); err != nil || !slices.Equal(f.Hidden(), []string{"m3"}) {
		t.Fatalf("esconder: %v %v", err, f.Hidden())
	}
}
