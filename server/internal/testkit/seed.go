package testkit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

var lookPath = exec.LookPath

// IDs fixos do acervo semeado (store.ID(tipo, chave)), pros testes
// acharem os itens sem consulta.
type SeedIDs struct {
	MovieBig, MoviePixel, MovieNoYear string
	Series, Season1, Season2          string
	Ep1, Ep2, Ep3                     string // S01E01, S01E02, S02E01
	Artist, ArtistX, Album, AlbumSolo string
	TrackBip, TrackBop, TrackSolta    string
	SNES, PSX                         string
	RomSuper, RomOutro, RomDisco      string
	BiosPSX                           string
}

// Seed grava o acervo das pastas de New exatamente como o scanner deve
// gravar (é o contrato do scanner). Devolve os ids.
func (k *Kit) Seed(t testing.TB) SeedIDs {
	t.Helper()
	ctx := context.Background()
	c := k.Config
	p := func(base string, rel ...string) string { return filepath.Join(append([]string{base}, rel...)...) }
	put := func(it *store.Item) *store.Item {
		t.Helper()
		if st, err := os.Stat(it.Path); err == nil {
			if !st.IsDir() {
				it.Size, it.MTime = st.Size(), st.ModTime().Unix()
			}
		}
		if err := k.Store.UpsertItem(ctx, it); err != nil {
			t.Fatal(err)
		}
		return it
	}
	webm := []store.Stream{
		{Index: 0, Type: "video", Codec: "vp8", Width: 96, Height: 64, FPS: 10, Range: "SDR", Default: true},
		{Index: 1, Type: "audio", Codec: "vorbis", Language: "por", Channels: 1, Layout: "mono", Rate: 44100, Default: true},
	}
	mp4 := []store.Stream{
		{Index: 0, Type: "video", Codec: "h264", Profile: "High", Level: 10, Width: 96, Height: 64, FPS: 10, PixFmt: "yuv420p", Range: "SDR", Bitrate: 15188, Default: true},
		{Index: 1, Type: "audio", Codec: "aac", Profile: "LC", Language: "eng", Channels: 1, Layout: "mono", Rate: 44100, Bitrate: 65017, Default: true},
	}
	var ids SeedIDs

	// filmes: "Nome (Ano)" no arquivo ou na pasta; poster.png ao lado; .srt
	// ao lado vira faixa externa (idioma pelo sufixo)
	big := p(c.Movies, "O Grande Teste (2021)", "O Grande Teste (2021).webm")
	ids.MovieBig = put(&store.Item{Kind: store.KindMovie, Library: "movies", Path: big, RelPath: "O Grande Teste (2021)/O Grande Teste (2021).webm",
		Name: "O Grande Teste", Year: 2021, Container: "webm", Duration: 2.003, Bitrate: 66372, Width: 96, Height: 64,
		Streams: append(append([]store.Stream{}, webm...), store.Stream{Index: 2, Type: "subtitle", Codec: "srt", Language: "por",
			External: p(c.Movies, "O Grande Teste (2021)", "O Grande Teste (2021).pt.srt")}),
		Image: p(c.Movies, "O Grande Teste (2021)", "poster.png")}).ID
	ids.MoviePixel = put(&store.Item{Kind: store.KindMovie, Library: "movies", Path: p(c.Movies, "Curta Pixel (2019).mp4"), RelPath: "Curta Pixel (2019).mp4",
		Name: "Curta Pixel", Year: 2019, Container: "mp4", Duration: 2, Bitrate: 90280, Width: 96, Height: 64, Streams: mp4}).ID
	ids.MovieNoYear = put(&store.Item{Kind: store.KindMovie, Library: "movies", Path: p(c.Movies, "Filme Sem Ano.webm"), RelPath: "Filme Sem Ano.webm",
		Name: "Filme Sem Ano", Container: "webm", Duration: 2.003, Bitrate: 66372, Width: 96, Height: 64, Streams: webm}).ID

	// séries: Série/Season NN/[Série - ]SxxEyy[ - Título].ext
	series := put(&store.Item{Kind: store.KindSeries, Library: "shows", Path: p(c.Shows, "Série de Teste"), RelPath: "Série de Teste",
		Name: "Série de Teste", Image: p(c.Shows, "Série de Teste", "poster.png")})
	ids.Series = series.ID
	season := func(n int, dir string) *store.Item {
		return put(&store.Item{Kind: store.KindSeason, Library: "shows", Path: p(c.Shows, "Série de Teste", dir), RelPath: "Série de Teste/" + dir,
			Name: "Temporada " + itoa(n), Index: n, ParentID: series.ID, SeriesID: series.ID})
	}
	s1, s2 := season(1, "Season 01"), season(2, "Season 02")
	ids.Season1, ids.Season2 = s1.ID, s2.ID
	ep := func(s *store.Item, dir, file, name string, n int) string {
		return put(&store.Item{Kind: store.KindEpisode, Library: "shows", Path: p(c.Shows, "Série de Teste", dir, file), RelPath: "Série de Teste/" + dir + "/" + file,
			Name: name, Index: n, ParentIndex: s.Index, ParentID: s.ID, SeriesID: series.ID, SeasonID: s.ID,
			Container: "webm", Duration: 2.003, Bitrate: 66372, Width: 96, Height: 64, Streams: webm}).ID
	}
	ids.Ep1 = ep(s1, "Season 01", "Série de Teste - S01E01 - O Começo.webm", "O Começo", 1)
	ids.Ep2 = ep(s1, "Season 01", "Série de Teste - S01E02 - O Segundo.webm", "O Segundo", 2)
	ids.Ep3 = ep(s2, "Season 02", "S02E01 - Recomeço.webm", "Recomeço", 1)

	// música: tags mandam; sem tag, "Artista - Título" no nome e a pasta
	// vira o álbum. Capa: embutida na faixa ("embedded:<arquivo>").
	artist := put(&store.Item{Kind: store.KindArtist, Library: "music", Path: "artist:banda teste", Name: "Banda Teste"})
	artistX := put(&store.Item{Kind: store.KindArtist, Library: "music", Path: "artist:artista x", Name: "Artista X"})
	ids.Artist, ids.ArtistX = artist.ID, artistX.ID
	bip := p(c.Music, "Banda Teste", "Disco Teste", "01 - Bip.mp3")
	album := put(&store.Item{Kind: store.KindAlbum, Library: "music", Path: "album:banda teste/disco teste", Name: "Disco Teste",
		Year: 2020, Genres: []string{"Rock"}, Artist: "Banda Teste", AlbumArtist: "Banda Teste", ParentID: artist.ID, ArtistID: artist.ID,
		Duration: 4, Image: "embedded:" + bip, RelPath: "Banda Teste/Disco Teste"})
	solo := put(&store.Item{Kind: store.KindAlbum, Library: "music", Path: "album:artista x/avulsas", Name: "Avulsas",
		Artist: "Artista X", AlbumArtist: "Artista X", ParentID: artistX.ID, ArtistID: artistX.ID, Duration: 2, RelPath: "Avulsas"})
	ids.Album, ids.AlbumSolo = album.ID, solo.ID
	track := func(al *store.Item, artistID, file, rel, name, artistName string, n, disc int, container string, bitrate int, image string, genres []string, year int) string {
		return put(&store.Item{Kind: store.KindTrack, Library: "music", Path: file, RelPath: rel, Name: name, Index: n, ParentIndex: disc,
			ParentID: al.ID, AlbumID: al.ID, ArtistID: artistID, Artist: artistName, AlbumArtist: al.AlbumArtist, Album: al.Name,
			Year: year, Genres: genres, Container: container, Duration: 2, Bitrate: bitrate, Image: image,
			Streams: []store.Stream{{Index: 0, Type: "audio", Codec: container, Channels: 1, Layout: "mono", Rate: 44100, Default: true}}}).ID
	}
	ids.TrackBip = track(album, artist.ID, bip, "Banda Teste/Disco Teste/01 - Bip.mp3", "Bip", "Banda Teste", 1, 1, "mp3", 64000, "embedded:"+bip, []string{"Rock"}, 2020)
	ids.TrackBop = track(album, artist.ID, p(c.Music, "Banda Teste", "Disco Teste", "02 - Bop.flac"), "Banda Teste/Disco Teste/02 - Bop.flac", "Bop", "Banda Teste", 2, 1, "flac", 136080, "", []string{"Rock"}, 2020)
	ids.TrackSolta = track(solo, artistX.ID, p(c.Music, "Avulsas", "Artista X - Solta.mp3"), "Avulsas/Artista X - Solta.mp3", "Solta", "Artista X", 0, 0, "mp3", 64000, "", nil, 0)

	// jogos: roms/<slug>/<arquivo> ou roms/<slug>/<pasta>/ (multidisco);
	// nome sem as tags "(USA)"; tags viram regions/languages/tags no Extra
	plat := func(slug, name string) *store.Item {
		return put(&store.Item{Kind: store.KindPlatform, Library: "games", Path: p(c.Games, "roms", slug), RelPath: "roms/" + slug,
			Name: name, Platform: slug})
	}
	snes, psx := plat("snes", "Super Nintendo Entertainment System"), plat("psx", "PlayStation")
	ids.SNES, ids.PSX = snes.ID, psx.ID
	rom := func(pl *store.Item, fsName, name string, files []store.RomFile, regions, langs []string) string {
		multi := len(files) > 1
		var size int64
		for i, f := range files {
			st, err := os.Stat(p(c.Games, "roms", pl.Platform, f.Rel))
			if err != nil {
				t.Fatal(err)
			}
			files[i].Size = st.Size()
			size += st.Size()
		}
		it := put(&store.Item{Kind: store.KindRom, Library: "games", Path: p(c.Games, "roms", pl.Platform, fsName), RelPath: "roms/" + pl.Platform + "/" + fsName,
			Name: name, ParentID: pl.ID, Platform: pl.Platform,
			Extra: map[string]any{"fs_name": fsName, "multi": multi, "files": files, "regions": regions, "languages": langs, "tags": []string{}}})
		if multi {
			// pasta: tamanho é a soma; mtime fica o da pasta
			it.Size = size
			if err := k.Store.UpsertItem(ctx, it); err != nil {
				t.Fatal(err)
			}
		}
		return it.ID
	}
	ids.RomSuper = rom(snes, "Super Teste (USA).sfc", "Super Teste", []store.RomFile{{Name: "Super Teste (USA).sfc", Rel: "Super Teste (USA).sfc"}}, []string{"USA"}, []string{})
	ids.RomOutro = rom(snes, "Outro Jogo (Europe) (En,Fr).sfc", "Outro Jogo", []store.RomFile{{Name: "Outro Jogo (Europe) (En,Fr).sfc", Rel: "Outro Jogo (Europe) (En,Fr).sfc"}}, []string{"Europe"}, []string{"En", "Fr"})
	disco := "Jogo Disco (USA)"
	ids.RomDisco = rom(psx, disco, "Jogo Disco", []store.RomFile{
		{Name: disco + " (Disc 1).bin", Rel: disco + "/" + disco + " (Disc 1).bin"},
		{Name: disco + " (Disc 1).cue", Rel: disco + "/" + disco + " (Disc 1).cue"},
		{Name: disco + " (Disc 2).bin", Rel: disco + "/" + disco + " (Disc 2).bin"},
		{Name: disco + " (Disc 2).cue", Rel: disco + "/" + disco + " (Disc 2).cue"},
	}, []string{"USA"}, []string{})
	ids.BiosPSX = put(&store.Item{Kind: store.KindFirmware, Library: "games", Path: p(c.Games, "bios", "psx", "scph1001.bin"), RelPath: "bios/psx/scph1001.bin",
		Name: "scph1001.bin", ParentID: psx.ID, Platform: "psx"}).ID
	return ids
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
