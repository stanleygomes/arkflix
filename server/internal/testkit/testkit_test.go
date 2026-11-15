package testkit_test

import (
	"context"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/store"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

func TestSeed(t *testing.T) {
	k := testkit.New(t)
	ids := k.Seed(t)
	ctx := context.Background()
	ep, err := k.Store.Item(ctx, ids.Ep2)
	if err != nil || ep.SeriesID != ids.Series || ep.ParentIndex != 1 || ep.Index != 2 || ep.Size == 0 {
		t.Fatalf("%+v %v", ep, err)
	}
	disco, _ := k.Store.Item(ctx, ids.RomDisco)
	if len(disco.Files()) != 4 || disco.Extra["multi"] != true || disco.Size == 0 || disco.RID == 0 {
		t.Fatalf("%+v", disco)
	}
	// rescan (upsert de novo) mantém o rid
	rid := disco.RID
	k.Seed(t)
	again, _ := k.Store.Item(ctx, ids.RomDisco)
	if again.RID != rid {
		t.Fatalf("rid mudou: %d -> %d", rid, again.RID)
	}
	d, err := k.Store.Progress(ctx, k.AdminUser.ID, ep, 1.0)
	if err != nil || d.Position != 1.0 || d.Played {
		t.Fatalf("%+v", d)
	}
	d, _ = k.Store.Progress(ctx, k.AdminUser.ID, ep, 1.95)
	if !d.Played || d.Position != 0 || d.PlayCount != 1 {
		t.Fatalf("fim = assistido: %+v", d)
	}
	if n, _ := k.Store.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE kind = ?`, store.KindTrack); len(n) != 3 {
		t.Fatalf("faixas: %d", len(n))
	}
}
