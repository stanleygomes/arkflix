package importer

import (
	"context"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// as listas importadas vão pras listas do servidor e rodar de novo não duplica
func TestStoreListsMerges(t *testing.T) {
	k := testkit.New(t)
	ids := k.Seed(t)
	ctx := context.Background()
	sink := storeLists{lists.New(k.Store)}
	for range 2 {
		if err := sink.AddList(ctx, k.AdminUser.ID, "Viagem", []string{ids.TrackBip, ids.TrackBop}); err != nil {
			t.Fatal(err)
		}
	}
	if err := sink.AddList(ctx, k.AdminUser.ID, " viagem ", []string{ids.TrackSolta}); err != nil {
		t.Fatal(err)
	}
	own, _ := lists.New(k.Store).Own(ctx, k.AdminUser.ID)
	if len(own) != 1 {
		t.Fatalf("listas: %d", len(own))
	}
	es, _ := lists.New(k.Store).Entries(ctx, own[0].ID)
	if len(es) != 3 {
		t.Fatalf("itens: %+v", es)
	}
}
