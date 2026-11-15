package lists_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func ids(t *testing.T, ls *lists.Lists, id string) string {
	t.Helper()
	es, err := ls.Entries(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Kind+":"+e.ItemID[:4])
	}
	return strings.Join(out, ",")
}

func TestLists(t *testing.T) {
	ctx := context.Background()
	k := testkit.New(t)
	s := k.Seed(t)
	ls := lists.New(k.Store)
	short := func(kind, id string) string { return kind + ":" + id[:4] }

	if _, err := ls.Create(ctx, k.AdminUser.ID, "  "); err != lists.ErrName {
		t.Fatalf("nome vazio: %v", err)
	}
	l, err := ls.Create(ctx, k.AdminUser.ID, "Minha")
	if err != nil || l.Owner != "stanley" || l.RID == 0 || l.Public {
		t.Fatalf("criar: %+v %v", l, err)
	}
	ls.Add(ctx, l.ID, []string{s.TrackBip, s.MovieBig, s.RomSuper, s.TrackBip, "nao-existe"})
	want := strings.Join([]string{short("track", s.TrackBip), short("movie", s.MovieBig), short("rom", s.RomSuper), ":naoe"}, ",")
	if got := ids(t, ls, l.ID); got != want {
		t.Fatalf("add: %s", got)
	}
	es, _ := ls.Entries(ctx, l.ID)
	first := es[0].ID

	// SetKinds troca só as faixas, no lugar delas
	ls.SetKinds(ctx, l.ID, []string{store.KindTrack}, []string{s.TrackBip, s.TrackBop, s.TrackBip})
	want = strings.Join([]string{short("track", s.TrackBip), short("movie", s.MovieBig), short("rom", s.RomSuper), ":naoe",
		short("track", s.TrackBop), short("track", s.TrackBip)}, ",")
	if got := ids(t, ls, l.ID); got != want {
		t.Fatalf("setKinds: %s", got)
	}
	if es, _ := ls.Entries(ctx, l.ID); es[0].ID != first {
		t.Fatal("entrada igual perdeu o id")
	}
	if !lists.HasKinds(es, store.KindRom) || lists.HasKinds(es, store.KindEpisode) || !lists.HasKinds(nil, store.KindEpisode) {
		t.Fatal("HasKinds")
	}
	if got := lists.ItemIDs(es, store.KindTrack, store.KindMovie); len(got) != 2 {
		t.Fatalf("ItemIDs: %v", got)
	}

	// Remove por item tira todas as repetições; Reorder e Move
	ls.Remove(ctx, l.ID, []string{s.TrackBip, "nao-existe"})
	ls.Reorder(ctx, l.ID, []string{s.RomSuper, s.TrackBop})
	want = strings.Join([]string{short("rom", s.RomSuper), short("track", s.TrackBop), short("movie", s.MovieBig)}, ",")
	if got := ids(t, ls, l.ID); got != want {
		t.Fatalf("reorder: %s", got)
	}
	ls.Move(ctx, l.ID, s.MovieBig, 0)
	if got := ids(t, ls, l.ID); !strings.HasPrefix(got, "movie:") {
		t.Fatalf("move: %s", got)
	}
	if err := ls.Move(ctx, l.ID, "nada", 0); err != store.ErrNotFound {
		t.Fatalf("move inexistente: %v", err)
	}

	// visibilidade e atualização
	pub := true
	name := "Nova"
	if err := ls.Update(ctx, l.ID, lists.Changes{Name: &name, Public: &pub}); err != nil {
		t.Fatal(err)
	}
	if v, _ := ls.Visible(ctx, k.PlainUser.ID); len(v) != 1 || v[0].Name != "Nova" || !v[0].CanSee(k.PlainUser.ID) {
		t.Fatalf("pública: %+v", v)
	}
	if o, _ := ls.Own(ctx, k.PlainUser.ID); len(o) != 0 {
		t.Fatalf("own: %+v", o)
	}
	if err := ls.Update(ctx, "nada", lists.Changes{Name: &name}); err != store.ErrNotFound {
		t.Fatalf("update inexistente: %v", err)
	}
	if err := ls.Add(ctx, "nada", nil); err != store.ErrNotFound {
		t.Fatalf("add inexistente: %v", err)
	}
	// apagar o usuário leva as listas dele
	k.Auth.DeleteUser(ctx, k.AdminUser.ID)
	if _, err := ls.Get(ctx, l.ID); err != store.ErrNotFound {
		t.Fatalf("lista do usuário apagado: %v", err)
	}
}
