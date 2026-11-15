package library_test

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/store"
	"github.com/stanleygomes/arkflix/server/internal/testkit"
)

// scanned roda o scanner nas pastas do kit num banco separado.
func scanned(t *testing.T, k *testkit.Kit) (*store.Store, *library.Library) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	lib := library.New(k.Config, st)
	if err := lib.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	return st, lib
}

func all(t *testing.T, st *store.Store) map[string]*store.Item {
	t.Helper()
	items, err := st.Items(context.Background(), `SELECT `+store.ItemColumns+` FROM items i`)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*store.Item{}
	for _, it := range items {
		out[it.ID] = it
	}
	return out
}

// TestScanMatchesSeed: escanear as pastas do testkit dá o mesmo acervo que
// Seed grava (o contrato).
func TestScanMatchesSeed(t *testing.T) {
	k := testkit.New(t)
	if k.Config.FFprobe == "" {
		t.Skip("sem ffprobe")
	}
	got, lib := scanned(t, k)
	k.Seed(t)
	want := all(t, k.Store)
	have := all(t, got)
	if n := lib.Status().Count; n != len(want) {
		t.Errorf("Count = %d, quer %d", n, len(want))
	}
	for id, w := range want {
		h := have[id]
		if h == nil {
			t.Errorf("faltou %s %s", w.Kind, w.Path)
			continue
		}
		compare(t, w, h)
	}
	for id, h := range have {
		if want[id] == nil {
			t.Errorf("sobrou %s %s", h.Kind, h.Path)
		}
	}
}

func compare(t *testing.T, w, h *store.Item) {
	t.Helper()
	if math.Abs(w.Duration-h.Duration) > 0.05 {
		t.Errorf("%s: Duration %v != %v", w.Path, h.Duration, w.Duration)
	}
	for _, it := range []*store.Item{w, h} {
		it.RID, it.MTime, it.Duration = 0, 0, 0
		it.AddedAt, it.UpdatedAt = time.Time{}, time.Time{}
	}
	wj, _ := json.MarshalIndent(w, "", " ")
	hj, _ := json.MarshalIndent(h, "", " ")
	if !reflect.DeepEqual(wj, hj) {
		t.Errorf("%s difere:\nscan: %s\nseed: %s", w.Path, hj, wj)
	}
}

// TestRescan: segundo scan não regrava nem chama o ffprobe; arquivo que
// some leva junto o que ficou órfão.
func TestRescan(t *testing.T) {
	k := testkit.New(t)
	if k.Config.FFprobe == "" {
		t.Skip("sem ffprobe")
	}
	st, lib := scanned(t, k)
	ctx := context.Background()
	before := all(t, st)

	k.Config.FFprobe = "/nao/existe/ffprobe" // arquivo igual não pode passar no ffprobe
	if err := lib.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	after := all(t, st)
	for id, b := range before {
		a := after[id]
		if a == nil || a.RID != b.RID || !a.UpdatedAt.Equal(b.UpdatedAt) || len(a.Streams) != len(b.Streams) || a.Duration != b.Duration {
			t.Fatalf("rescan mexeu em %s:\n%+v\n%+v", b.Path, b, a)
		}
	}

	ids := k.Seed(t) // só pelos ids (grava no banco do kit, não no do scan)
	for _, rel := range []string{"musicas/Avulsas/Artista X - Solta.mp3", "Shows/Série de Teste/Season 02/S02E01 - Recomeço.webm",
		"jogos/roms/snes/Outro Jogo (Europe) (En,Fr).sfc"} {
		if err := os.Remove(filepath.Join(k.Root, rel)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(filepath.Join(k.Root, "jogos/roms/psx")); err != nil {
		t.Fatal(err)
	}
	if err := lib.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	left := all(t, st)
	for name, id := range map[string]string{"faixa": ids.TrackSolta, "álbum": ids.AlbumSolo, "artista": ids.ArtistX,
		"episódio": ids.Ep3, "temporada": ids.Season2, "rom": ids.RomOutro, "console": ids.PSX, "disco": ids.RomDisco, "bios": ids.BiosPSX} {
		if left[id] != nil {
			t.Errorf("%s devia ter sumido", name)
		}
	}
	for name, id := range map[string]string{"série": ids.Series, "temporada 1": ids.Season1, "snes": ids.SNES, "rom": ids.RomSuper, "álbum": ids.Album} {
		if left[id] == nil {
			t.Errorf("%s sumiu", name)
		}
	}
}

// TestMissingRoot: pasta configurada que não existe (disco desmontado) não
// apaga o que já estava no banco.
func TestMissingRoot(t *testing.T) {
	k := testkit.New(t)
	st, lib := scanned(t, k)
	n := len(all(t, st))
	k.Config.Movies = filepath.Join(k.Root, "sumiu")
	if err := lib.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := len(all(t, st)); got != n {
		t.Fatalf("itens: %d -> %d", n, got)
	}
}

func TestScanCanceled(t *testing.T) {
	k := testkit.New(t)
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := library.New(k.Config, st).Scan(ctx); err == nil {
		t.Fatal("scan cancelado devia dar erro")
	}
}

// TestScanNoFFprobe: sem ffprobe o acervo sai igual, só sem duração/faixas.
func TestScanNoFFprobe(t *testing.T) {
	k := testkit.New(t)
	k.Config.FFprobe = ""
	st, _ := scanned(t, k)
	ids := k.Seed(t)
	have := all(t, st)
	if len(have) != len(all(t, k.Store)) {
		t.Fatalf("itens: %d", len(have))
	}
	bip := have[ids.TrackBip]
	if bip.Name != "Bip" || bip.Duration != 0 || len(bip.Streams) != 1 || bip.Streams[0].Codec != "mp3" {
		t.Fatalf("%+v", bip)
	}
	if mv := have[ids.MovieBig]; len(mv.Streams) != 1 || mv.Streams[0].External == "" {
		t.Fatalf("%+v", mv.Streams)
	}
}
