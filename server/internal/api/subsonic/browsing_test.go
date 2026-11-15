package subsonic

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIndexes(t *testing.T) {
	e := setup(t)
	r := e.ok("getIndexes", q())
	ix := r.Indexes
	if ix == nil || ix.IgnoredArticles != ignoredArticles || ix.LastModified == 0 || len(ix.Index) != 2 {
		t.Fatalf("indexes: %+v", ix)
	}
	if ix.Index[0].Name != "A" || ix.Index[0].Artists[0].Name != "Artista X" || ix.Index[1].Name != "B" ||
		ix.Index[1].Artists[0].ID != e.ids.Artist || ix.Index[1].Artists[0].CoverArt != e.ids.Artist {
		t.Fatalf("index: %+v", ix.Index)
	}
	// nada mudou desde então: sem índice
	future := strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10)
	if r := e.ok("getIndexes", q("ifModifiedSince", future)); len(r.Indexes.Index) != 0 {
		t.Fatalf("ifModifiedSince: %+v", r.Indexes)
	}
	e.failed("getIndexes", q("musicFolderId", "2"), 70)

	r = e.ok("getArtists", q("musicFolderId", "1"))
	if a := r.Artists; a == nil || len(a.Index) != 2 || a.Index[1].Artists[0].AlbumCount != 1 || a.Index[1].Artists[0].Name != "Banda Teste" {
		t.Fatalf("artists: %+v", r.Artists)
	}
	xml := e.xmlBody("getArtists", q())
	if !strings.Contains(xml, `<artists lastModified="`) || !strings.Contains(xml, `ignoredArticles="The El La Los Las Le Les Os As O A"><index name="A"><artist id="`+e.ids.ArtistX+`" name="Artista X" albumCount="1" sortName="artista x"><roles>albumartist</roles></artist></index>`) {
		t.Fatalf("artists xml: %s", xml)
	}
}

func TestArtistAlbumSong(t *testing.T) {
	e := setup(t)
	r := e.ok("getArtist", q("id", e.ids.Artist))
	if a := r.Artist; a == nil || a.Name != "Banda Teste" || a.AlbumCount != 1 || len(a.Album) != 1 || a.Album[0].ID != e.ids.Album {
		t.Fatalf("artist: %+v", r.Artist)
	}
	e.failed("getArtist", q("id", "nada"), 70)
	e.failed("getArtist", q(), 10)

	r = e.ok("getAlbum", q("id", e.ids.Album))
	al := r.Album
	if al == nil || al.Name != "Disco Teste" || al.SongCount != 2 || al.Duration != 4 || al.Year != 2020 || al.Genre != "Rock" ||
		al.ArtistID != e.ids.Artist || al.CoverArt != e.ids.Album || al.Created.IsZero() || al.ReleaseDate.Year != 2020 {
		t.Fatalf("album: %+v", al)
	}
	eq(t, "faixas", titles(al.Song), []string{"Bip", "Bop"})
	// prefixo do Navidrome é aceito
	if r := e.ok("getAlbum", q("id", "al-"+e.ids.Album)); r.Album.ID != e.ids.Album {
		t.Fatal("prefixo al-")
	}

	r = e.ok("getSong", q("id", e.ids.TrackBip))
	s := r.Song
	if s.Path != "Banda Teste/Disco Teste/01-01 - Bip.mp3" || s.ContentType != "audio/mpeg" || s.Suffix != "mp3" ||
		s.BitRate != 64 || s.Duration != 2 || s.Parent != e.ids.Album || s.AlbumID != e.ids.Album || s.ArtistID != e.ids.Artist ||
		s.Type != "music" || s.IsDir || s.CoverArt != e.ids.TrackBip || s.Track != 1 || s.DiscNumber != 1 || s.Year != 2020 ||
		s.MediaType != "song" || s.ChannelCount != 1 || s.SamplingRate != 44100 || s.Size == 0 || s.Created == nil ||
		len(s.Artists) != 1 || s.Artists[0].ID != e.ids.Artist || len(s.AlbumArtists) != 1 || s.DisplayAlbumArtist != "Banda Teste" {
		t.Fatalf("song: %+v", s)
	}
	if r := e.ok("getSong", q("id", e.ids.TrackBop)); r.Song.CoverArt != e.ids.Album || r.Song.ContentType != "audio/flac" || r.Song.BitRate != 136 {
		t.Fatalf("bop: %+v", r.Song)
	}
	if r := e.ok("getSong", q("id", e.ids.TrackSolta)); r.Song.CoverArt != "" || r.Song.Path != "Artista X/Avulsas/Solta.mp3" {
		t.Fatalf("solta: %+v", r.Song)
	}
	e.failed("getSong", q("id", e.ids.Album), 70)

	// campos do OpenSubsonic: JSON sempre, XML só com valor
	w := e.raw("GET", "/rest/getSong", login(q("f", "json", "id", e.ids.TrackSolta)))
	for _, want := range []string{`"bpm":0`, `"comment":""`, `"isrc":[]`, `"genres":[]`, `"replayGain":{}`, `"moods":[]`,
		`"contributors":[]`, `"explicitStatus":""`, `"musicBrainzId":""`, `"mediaType":"song"`, `"isDir":false`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("falta %s em %s", want, w.Body.String())
		}
	}
	xml := e.xmlBody("getSong", q("id", e.ids.TrackBip))
	if !strings.Contains(xml, `<song id="`+e.ids.TrackBip+`" parent="`+e.ids.Album+`" isDir="false" title="Bip" album="Disco Teste" artist="Banda Teste" track="1" year="2020" genre="Rock"`) ||
		!strings.Contains(xml, `<genres name="Rock"></genres>`) || strings.Contains(xml, "replayGain") || strings.Contains(xml, `bpm=`) ||
		!strings.Contains(xml, `mediaType="song"`) || !strings.Contains(xml, `<artists id="`+e.ids.Artist+`" name="Banda Teste"></artists>`) {
		t.Fatalf("song xml: %s", xml)
	}
	w = e.raw("GET", "/rest/getAlbum", login(q("f", "json", "id", e.ids.AlbumSolo)))
	for _, want := range []string{`"recordLabels":[]`, `"releaseTypes":[]`, `"discTitles":[]`, `"originalReleaseDate":{}`,
		`"releaseDate":{}`, `"isCompilation":false`, `"userRating":0`, `"displayArtist":"Artista X"`, `"songCount":1`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("falta %s em %s", want, w.Body.String())
		}
	}
}

func TestMusicDirectory(t *testing.T) {
	e := setup(t)
	r := e.ok("getMusicDirectory", q("id", e.ids.Artist))
	d := r.Directory
	if d.ID != e.ids.Artist || d.Name != "Banda Teste" || len(d.Child) != 1 || !d.Child[0].IsDir || d.Child[0].Title != "Disco Teste" ||
		d.Child[0].Parent != e.ids.Artist || d.Child[0].SongCount != 2 || d.Child[0].MediaType != "album" {
		t.Fatalf("dir artista: %+v", d)
	}
	r = e.ok("getMusicDirectory", q("id", d.Child[0].ID))
	if d := r.Directory; d.Parent != e.ids.Artist || len(d.Child) != 2 || d.Child[1].Title != "Bop" || d.Child[1].IsDir {
		t.Fatalf("dir álbum: %+v", d)
	}
	e.failed("getMusicDirectory", q("id", e.ids.TrackBip), 70)
	e.failed("getMusicDirectory", q("id", e.ids.MovieBig), 70)
}

func TestGenresInfoSimilar(t *testing.T) {
	e := setup(t)
	r := e.ok("getGenres", q())
	if g := r.Genres; g == nil || len(g.Genre) != 1 || g.Genre[0].Name != "Rock" || g.Genre[0].SongCount != 2 || g.Genre[0].AlbumCount != 1 {
		t.Fatalf("genres: %+v", r.Genres)
	}
	if xml := e.xmlBody("getGenres", q()); !strings.Contains(xml, `<genres><genre songCount="2" albumCount="1">Rock</genre></genres>`) {
		t.Fatalf("genres xml: %s", xml)
	}
	if r := e.ok("getArtistInfo2", q("id", e.ids.Artist)); r.ArtistInfo2 == nil {
		t.Fatal("artistInfo2")
	}
	e.ok("getArtistInfo", q("id", e.ids.Artist))
	e.failed("getArtistInfo", q("id", "x"), 70)
	if r := e.ok("getAlbumInfo2", q("id", e.ids.Album)); r.AlbumInfo == nil {
		t.Fatal("albumInfo")
	}
	r = e.ok("getSimilarSongs", q("id", e.ids.TrackBip))
	eq(t, "similares", titles(r.SimilarSongs.Songs), []string{"Bop"})
	r = e.ok("getSimilarSongs2", q("id", e.ids.Artist, "count", "5"))
	if len(r.SimilarSongs2.Songs) != 2 {
		t.Fatalf("similares2: %+v", r.SimilarSongs2)
	}
	// top: mais tocadas primeiro
	e.ok("scrobble", q("id", e.ids.TrackBop))
	r = e.ok("getTopSongs", q("artist", "banda teste"))
	eq(t, "top", titles(r.TopSongs.Songs), []string{"Bop", "Bip"})
	r = e.ok("getTopSongs", q("id", e.ids.ArtistX, "count", "1"))
	eq(t, "top id", titles(r.TopSongs.Songs), []string{"Solta"})
	e.failed("getTopSongs", q(), 10)
}
