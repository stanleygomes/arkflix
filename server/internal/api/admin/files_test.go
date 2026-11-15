package admin_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

type savedT struct{ Saved []string }

func (e *env) upload(auth string, fs ...field) (int, []byte) {
	e.t.Helper()
	body, ct := multipartBody(e.t, fs...)
	return e.req("POST", "/arkflix/api/upload", auth, body, ct)
}

func (e *env) uploadOK(want string, fs ...field) {
	e.t.Helper()
	code, raw := e.upload(e.admin, fs...)
	if code != 201 || !strings.Contains(string(raw), `"`+want+`"`) {
		e.t.Fatalf("upload %s: %d %s", want, code, raw)
	}
	got, err := os.ReadFile(filepath.Join(e.k.Root, filepath.FromSlash(want)))
	if err != nil {
		e.t.Fatalf("não gravou %s: %v", want, err)
	}
	for _, f := range fs {
		if f.file != "" && string(got) == f.value {
			return
		}
	}
	e.t.Fatalf("conteúdo de %s: %q", want, got)
}

// files lista a árvore toda (relativo à raiz), pra achar arquivo fora do lugar.
func (e *env) files() []string {
	var out []string
	filepath.WalkDir(e.k.Root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(e.k.Root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
}

func TestUploadReadOnly(t *testing.T) {
	e := setup(t)
	code, raw := e.upload(e.admin, field{"kind", "movie", ""}, field{"file", "x", "a.mkv"})
	if code != 403 || errMsg(t, raw) != "servidor em modo só leitura" {
		t.Fatalf("só leitura: %d %s", code, raw)
	}
	e.k.Config.Writable = true
	if code, _ := e.upload(e.nina, field{"kind", "movie", ""}, field{"file", "x", "a.mkv"}); code != 403 {
		t.Fatalf("nina: %d", code)
	}
}

func TestUpload(t *testing.T) {
	e := setup(t)
	e.k.Config.Writable = true
	defer e.waitScan()

	e.uploadOK("Movies/Novo Filme (2020)/Novo Filme (2020).mkv",
		field{"kind", "movie", ""}, field{"title", "Novo Filme", ""}, field{"year", "2020", ""}, field{"file", "filme!", "original.MKV"})
	// sem título: nome do arquivo
	e.uploadOK("Movies/Outro/Outro.mp4", field{"kind", "movie", ""}, field{"file", "outro", "Outro.mp4"})
	// nome repetido: 409, e o original fica
	code, raw := e.upload(e.admin, field{"kind", "movie", ""}, field{"title", "Novo Filme", ""}, field{"year", "2020", ""},
		field{"file", "outro conteúdo", "x.mkv"})
	if code != 409 || errMsg(t, raw) == "" {
		t.Fatalf("repetido: %d %s", code, raw)
	}
	if b, _ := os.ReadFile(filepath.Join(e.k.Root, "Movies/Novo Filme (2020)/Novo Filme (2020).mkv")); string(b) != "filme!" {
		t.Fatal("sobrescreveu")
	}

	e.uploadOK("Shows/Série Nova/Season 02/Série Nova - S02E05 - Piloto.mkv", field{"kind", "episode", ""},
		field{"series", "Série Nova", ""}, field{"season", "2", ""}, field{"episode", "5", ""}, field{"title", "Piloto", ""},
		field{"file", "ep", "qualquer.mkv"})
	e.uploadOK("Shows/Série Nova/Season 01/S01E01.mkv", field{"kind", "episode", ""},
		field{"series", "Série Nova", ""}, field{"season", "1", ""}, field{"file", "ep1", "S01E01.mkv"})
	e.uploadOK("musicas/Banda/Disco/03 - Faixa.flac", field{"kind", "music", ""}, field{"folder", "Banda/Disco", ""},
		field{"file", "flac", "03 - Faixa.flac"})
	e.uploadOK("musicas/Avulsas/solta.mp3", field{"kind", "music", ""}, field{"file", "mp3", "solta.mp3"})
	e.uploadOK("jogos/roms/gba/Jogo.gba", field{"kind", "rom", ""}, field{"platform", "gba", ""}, field{"file", "rom", "Jogo.gba"})
	e.uploadOK("jogos/bios/gba/gba_bios.bin", field{"kind", "bios", ""}, field{"platform", "GBA", ""}, field{"file", "bios", "gba_bios.bin"})
	e.uploadOK("Movies/O Grande Teste (2021)/O Grande Teste (2021).en.srt", field{"kind", "subtitle", ""},
		field{"item", e.ids.MovieBig, ""}, field{"lang", "en", ""}, field{"file", "1\n", "legenda.srt"})
	e.uploadOK("Shows/Série de Teste/Season 01/Série de Teste - S01E01 - O Começo.srt", field{"kind", "subtitle", ""},
		field{"item", e.ids.Ep1, ""}, field{"file", "2\n", "x.SRT"})

	// arquivo antes dos campos: espera em <data>/uploads e vai pro lugar no fim
	e.uploadOK("jogos/roms/snes/Antes.sfc", field{"file", "antes", "Antes.sfc"}, field{"kind", "rom", ""}, field{"platform", "snes", ""})
	if es, _ := os.ReadDir(filepath.Join(e.k.Config.Data, "uploads")); len(es) != 0 {
		t.Fatalf("sobrou na espera: %v", es)
	}
	// vários arquivos de uma vez
	code, raw = e.upload(e.admin, field{"kind", "rom", ""}, field{"platform", "psx", ""},
		field{"file", "1", "a.bin"}, field{"file", "2", "b.bin"})
	if code != 201 || !strings.Contains(string(raw), "jogos/roms/psx/a.bin") || !strings.Contains(string(raw), "jogos/roms/psx/b.bin") {
		t.Fatalf("vários: %d %s", code, raw)
	}

	for _, c := range []struct {
		what string
		fs   []field
	}{
		{"sem kind", []field{{"file", "x", "a.mkv"}}},
		{"kind ruim", []field{{"kind", "foto", ""}, {"file", "x", "a.jpg"}}},
		{"sem arquivo", []field{{"kind", "movie", ""}}},
		{"filme não vídeo", []field{{"kind", "movie", ""}, {"file", "x", "a.txt"}}},
		{"ano ruim", []field{{"kind", "movie", ""}, {"year", "20x0", ""}, {"file", "x", "a.mkv"}}},
		{"sem série", []field{{"kind", "episode", ""}, {"season", "1", ""}, {"file", "x", "a.mkv"}}},
		{"série ..", []field{{"kind", "episode", ""}, {"series", "..", ""}, {"season", "1", ""}, {"file", "x", "a.mkv"}}},
		{"console com ..", []field{{"kind", "rom", ""}, {"platform", "../snes", ""}, {"file", "x", "a.sfc"}}},
		{"console vazio", []field{{"kind", "bios", ""}, {"file", "x", "a.bin"}}},
		{"legenda sem item", []field{{"kind", "subtitle", ""}, {"file", "x", "a.srt"}}},
		{"legenda de rom", []field{{"kind", "subtitle", ""}, {"item", e.ids.RomSuper, ""}, {"file", "x", "a.srt"}}},
		{"legenda não srt", []field{{"kind", "subtitle", ""}, {"item", e.ids.MovieBig, ""}, {"file", "x", "a.mkv"}}},
		{"lang ruim", []field{{"kind", "subtitle", ""}, {"item", e.ids.MovieBig, ""}, {"lang", "../x", ""}, {"file", "x", "a.srt"}}},
		{"nome vazio", []field{{"kind", "music", ""}, {"file", "x", "..."}}},
	} {
		if code, raw := e.upload(e.admin, c.fs...); code != 400 || errMsg(t, raw) == "" {
			t.Fatalf("%s: %d %s", c.what, code, raw)
		}
	}

	// tentativa de sair da pasta: tudo é limpo e fica dentro da biblioteca
	before := e.files()
	e.uploadOK("Movies/x/x.mkv", field{"kind", "movie", ""}, field{"title", "../../x", ""}, field{"file", "t", "a.mkv"})
	e.uploadOK("musicas/etc/passwd", field{"kind", "music", ""}, field{"folder", "../../../etc", ""}, field{"file", "t", "../../passwd"})
	e.uploadOK("Shows/a b/Season 01/a b - S01E01 - c d.mkv", field{"kind", "episode", ""}, field{"series", "a/b", ""},
		field{"season", "1", ""}, field{"episode", "1", ""}, field{"title", "c\\d\x00", ""}, field{"file", "t", "a.mkv"})
	for _, f := range e.files() {
		if !slices.Contains(before, f) && !strings.HasPrefix(f, "Movies/") && !strings.HasPrefix(f, "musicas/") &&
			!strings.HasPrefix(f, "Shows/") && !strings.HasPrefix(f, "data/") {
			t.Fatalf("arquivo fora da biblioteca: %s", f)
		}
		if strings.HasSuffix(f, ".part") {
			t.Fatalf("sobrou .part: %s", f)
		}
	}
}

func TestHide(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	hidden := filepath.Join(e.k.Root, "escondidos")
	e.k.Config.Hidden = hidden
	hide := func(id string) (int, []byte) {
		return e.req("POST", "/arkflix/api/items/"+id+"/hide", e.admin, nil, "")
	}

	if code, raw := hide(e.ids.MovieBig); code != 403 || errMsg(t, raw) != "servidor em modo só leitura" {
		t.Fatalf("só leitura: %d %s", code, raw)
	}
	e.k.Config.Writable = true
	defer e.waitScan()
	if code, _ := e.req("POST", "/arkflix/api/items/"+e.ids.MovieBig+"/hide", e.nina, nil, ""); code != 403 {
		t.Fatalf("nina: %d", code)
	}
	if code, _ := hide("nada"); code != 404 {
		t.Fatalf("inexistente: %d", code)
	}
	if code, _ := hide(e.ids.Artist); code != 400 {
		t.Fatalf("artista: %d", code)
	}

	gone := func(id string) {
		t.Helper()
		if _, err := e.k.Store.Item(ctx, id); err != store.ErrNotFound {
			t.Fatalf("%s ainda no acervo: %v", id, err)
		}
	}
	there := func(rel string, want bool) {
		t.Helper()
		if _, err := os.Stat(filepath.Join(e.k.Root, filepath.FromSlash(rel))); (err == nil) != want {
			t.Fatalf("%s: existe=%v", rel, err == nil)
		}
	}

	// filme na pasta: a pasta inteira (pôster e legenda junto)
	if code, raw := hide(e.ids.MovieBig); code != 204 {
		t.Fatalf("filme: %d %s", code, raw)
	}
	there("Movies/O Grande Teste (2021)", false)
	there("escondidos/Movies/O Grande Teste (2021)/O Grande Teste (2021).webm", true)
	there("escondidos/Movies/O Grande Teste (2021)/poster.png", true)
	gone(e.ids.MovieBig)

	// filme solto na raiz: só o arquivo
	if code, _ := hide(e.ids.MoviePixel); code != 204 {
		t.Fatalf("filme solto: %d", code)
	}
	there("escondidos/Movies/Curta Pixel (2019).mp4", true)
	there("Movies/Filme Sem Ano.webm", true)

	// ROM multidisco: a pasta; por "rom:<rid>"
	disco, _ := e.k.Store.Item(ctx, e.ids.RomDisco)
	if code, _ := hide("rom:" + itoa(disco.RID)); code != 204 {
		t.Fatalf("multidisco: %d", code)
	}
	there("escondidos/jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 2).bin", true)
	there("jogos/roms/psx/Jogo Disco (USA)", false)

	// temporada: a pasta, e os episódios saem do acervo
	if code, _ := hide(e.ids.Season1); code != 204 {
		t.Fatalf("temporada: %d", code)
	}
	there("escondidos/Shows/Série de Teste/Season 01/Série de Teste - S01E02 - O Segundo.webm", true)
	there("Shows/Série de Teste/Season 02", true)
	gone(e.ids.Season1)
	gone(e.ids.Ep1)

	// já existe em escondidos: 409 e nada se mexe
	dst := filepath.Join(hidden, "Shows", "Série de Teste", "Season 02", "S02E01 - Recomeço.webm")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("x"), 0o644)
	if code, _ := hide(e.ids.Ep3); code != 409 {
		t.Fatalf("conflito: %d", code)
	}
	there("Shows/Série de Teste/Season 02/S02E01 - Recomeço.webm", true)

	// álbum: as faixas
	if code, _ := hide(e.ids.Album); code != 204 {
		t.Fatalf("álbum: %d", code)
	}
	there("escondidos/musicas/Banda Teste/Disco Teste/02 - Bop.flac", true)
	gone(e.ids.TrackBip)

	// sem pasta de escondidos
	e.k.Config.Hidden = ""
	if code, _ := hide(e.ids.RomSuper); code != 409 {
		t.Fatalf("sem --hidden: %d", code)
	}
}

func itoa(n int64) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
