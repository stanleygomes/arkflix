package importer

import (
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

func TestMapper(t *testing.T) {
	var m Mapper
	for _, r := range []string{"/music=/media/musicas", "/media=/var/media", "/media/Shows=/srv/series/"} {
		if err := m.Set(r); err != nil {
			t.Fatal(err)
		}
	}
	for in, want := range map[string]string{
		"/music/a/b.mp3":       "/media/musicas/a/b.mp3",
		"/musicas/a.mp3":       "/musicas/a.mp3", // só em fronteira de pasta
		"/media/Movies/x.mkv":  "/var/media/Movies/x.mkv",
		"/media/Shows/S/E.mkv": "/srv/series/S/E.mkv", // o prefixo mais longo ganha
		"/media":               "/var/media",
		"/outro/lugar//x.mkv":  "/outro/lugar/x.mkv",
		"":                     "",
	} {
		if got := m.Apply(in); got != want {
			t.Errorf("%q → %q, queria %q", in, got, want)
		}
	}
	if err := m.Set("sem-igual"); err == nil {
		t.Fatal("regra sem =")
	}
	if !strings.Contains(m.String(), "/music=/media/musicas") {
		t.Fatal(m.String())
	}
}

func TestToTime(t *testing.T) {
	want := time.Date(2025, 3, 1, 20, 15, 0, 123000000, time.UTC)
	for _, v := range []any{
		"2025-03-01 20:15:00.1234567",        // EF Core (Jellyfin)
		"2025-03-01T20:15:00.123Z",           // RFC3339
		"2025-03-01 20:15:00.123+00:00",      // Navidrome
		"2025-03-01 17:15:00.123-03:00",      // com fuso
		[]byte("2025-03-01 20:15:00.123456"), // MariaDB sem parseTime
		time.Date(2025, 3, 1, 20, 15, 0, 123456789, time.UTC),
	} {
		if got := toTime(v); !got.Equal(want) {
			t.Errorf("%v → %v", v, got)
		}
	}
	for _, v := range []any{nil, "", "lixo", "0001-01-01 00:00:00", time.Time{}, 42} {
		if !toTime(v).IsZero() {
			t.Errorf("%v devia dar zero", v)
		}
	}
}

func TestMergeUD(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	old := store.UserData{PlayCount: 2, Position: 30, LastPlayed: t2, Rating: 3}
	in := store.UserData{Played: true, PlayCount: 1, Position: 10, LastPlayed: t1, Favorite: true, FavoriteAt: t1, Rating: 5}
	m := mergeUD(old, in)
	if !m.Played || m.PlayCount != 2 || m.Position != 30 || !m.LastPlayed.Equal(t2) || !m.Favorite || m.Rating != 3 {
		t.Fatalf("%+v", m)
	}
	// juntar de novo não muda (o import pode rodar várias vezes)
	if !sameUD(mergeUD(m, in), m) {
		t.Fatal("não é idempotente")
	}
	// mais recente leva a posição
	m = mergeUD(store.UserData{Position: 30, LastPlayed: t1}, store.UserData{Position: 5, LastPlayed: t2})
	if m.Position != 5 || !m.LastPlayed.Equal(t2) {
		t.Fatalf("%+v", m)
	}
}
