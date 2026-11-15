package library

import (
	"reflect"
	"testing"
)

func TestParseTitle(t *testing.T) {
	cases := []struct {
		in   string
		name string
		year int
	}{
		{"O Grande Teste (2021)", "O Grande Teste", 2021},
		{"The.Matrix.1999.1080p.BluRay.x264", "The Matrix", 1999},
		{"1917 (2019) [1080p]", "1917", 2019},
		{"Blade Runner 2049 (2017)", "Blade Runner 2049", 2017},
		{"Filme Sem Ano", "Filme Sem Ano", 0},
		{"Duna 2160p HDR", "Duna", 0},
	}
	for _, c := range cases {
		n, y := parseTitle(c.in)
		if n != c.name || y != c.year {
			t.Errorf("%q → %q %d", c.in, n, y)
		}
	}
}

func TestEpisodeInfo(t *testing.T) {
	cases := []struct {
		in         string
		season, ep int
		title      string
	}{
		{"Série de Teste - S01E02 - O Segundo", 1, 2, "O Segundo"},
		{"S02E01 - Recomeço", 2, 1, "Recomeço"},
		{"show.s03e04e05.720p.web", 3, 4, ""},
		{"Show 1x07 Nome", 1, 7, "Nome"},
		{"05 - Solto", -1, 5, "Solto"},
	}
	for _, c := range cases {
		s, e, ti, ok := episodeInfo(c.in)
		if !ok || s != c.season || e != c.ep || ti != c.title {
			t.Errorf("%q → %d %d %q %v", c.in, s, e, ti, ok)
		}
	}
	if _, _, _, ok := episodeInfo("making of"); ok {
		t.Error("sem número não é episódio")
	}
	for in, want := range map[string]int{"Season 01": 1, "Temporada 3": 3, "Specials": 0, "S2": 2} {
		if n, ok := seasonDir(in); !ok || n != want {
			t.Errorf("%q → %d %v", in, n, ok)
		}
	}
}

func TestParseRom(t *testing.T) {
	title, regions, langs, tags := parseRom("Outro Jogo (Europe) (En,Fr) (Rev 1) [!]")
	if title != "Outro Jogo" || !reflect.DeepEqual(regions, []string{"Europe"}) ||
		!reflect.DeepEqual(langs, []string{"En", "Fr"}) || !reflect.DeepEqual(tags, []string{"Rev 1", "!"}) {
		t.Fatalf("%q %v %v %v", title, regions, langs, tags)
	}
	_, regions, _, _ = parseRom("Velho (UE)")
	if !reflect.DeepEqual(regions, []string{"USA", "Europe"}) {
		t.Fatalf("%v", regions)
	}
}

func TestSubStream(t *testing.T) {
	st := subStream(".pt-BR.forced.srt", "/x/f.pt-BR.forced.srt")
	if st.Language != "por" || !st.Forced || st.Codec != "srt" || st.External == "" {
		t.Fatalf("%+v", st)
	}
	if st := subStream(".srt", "/x/f.srt"); st.Language != "" || st.Forced {
		t.Fatalf("%+v", st)
	}
}

func TestPlatformName(t *testing.T) {
	for in, want := range map[string]string{"snes": "Super Nintendo Entertainment System", "megadrive": "Sega Mega Drive/Genesis",
		"gamecube": "Nintendo GameCube", "xyz": "xyz"} {
		if got := platformName(in); got != want {
			t.Errorf("%s → %s", in, got)
		}
	}
}
