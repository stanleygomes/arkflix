package subsonic

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Uma pasta de música só (a do --music).
const (
	folderID   = 1
	folderName = "Músicas"
)

// Artigos ignorados no índice (padrão do Navidrome + português).
const ignoredArticles = "The El La Los Las Le Les Os As O A"

// Tipos MIME por extensão, os mesmos do Navidrome.
var mimeTypes = map[string]string{
	"mp3": "audio/mpeg", "ogg": "audio/ogg", "oga": "audio/ogg", "opus": "audio/ogg", "aac": "audio/aac",
	"alac": "audio/mp4", "m4a": "audio/mp4", "m4b": "audio/mp4", "flac": "audio/flac", "wav": "audio/x-wav",
	"wma": "audio/x-ms-wma", "ape": "audio/x-monkeys-audio", "mpc": "audio/x-musepack", "shn": "audio/x-shn",
	"aif": "audio/x-aiff", "aiff": "audio/x-aiff", "dsf": "audio/x-dsf", "wv": "audio/x-wavpack",
	"wvp": "audio/x-wavpack", "tak": "audio/tak", "mka": "audio/x-matroska", "webm": "audio/webm",
}

func suffix(path string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
}

func contentType(path string) string {
	if t, ok := mimeTypes[suffix(path)]; ok {
		return t
	}
	return "application/octet-stream"
}

func firstGenre(g []string) string {
	if len(g) == 0 {
		return ""
	}
	return g[0]
}

func itemGenres(g []string) Array[itemGenre] {
	out := make(Array[itemGenre], 0, len(g))
	for _, n := range g {
		out = append(out, itemGenre{Name: n})
	}
	return out
}

func ref(id, name string) Array[artistRef] {
	if id == "" && name == "" {
		return Array[artistRef]{}
	}
	return Array[artistRef]{{ID: id, Name: name}}
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func noSlash(s string) string { return strings.ReplaceAll(s, "/", "_") }

// fakePath é o caminho que o Navidrome mostra: Artista/Álbum/DD-NN - Título.ext.
func fakePath(t *track) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s/%s/", noSlash(orStr(t.AlbumArtist, t.Artist)), noSlash(t.Album))
	if t.ParentIndex != 0 {
		fmt.Fprintf(&b, "%02d-", t.ParentIndex)
	}
	if t.Index != 0 {
		fmt.Fprintf(&b, "%02d - ", t.Index)
	}
	fmt.Fprintf(&b, "%s.%s", noSlash(t.Name), suffix(t.Path))
	return b.String()
}

func audioStream(it *store.Item) store.Stream {
	for _, s := range it.Streams {
		if s.Type == "audio" {
			return s
		}
	}
	return store.Stream{}
}

// cover é o id de capa da faixa (ela mesma ou o álbum); "" sem capa.
func (t *track) cover() string {
	switch {
	case t.Image != "":
		return t.ID
	case t.albumImage != "":
		return t.AlbumID
	}
	return ""
}

func (t *track) child() child {
	s := audioStream(t.Item)
	c := child{
		ID: t.ID, Parent: t.AlbumID, Title: t.Name, Album: t.Album, Artist: t.Artist,
		Track: t.Index, Year: t.Year, Genre: firstGenre(t.Genres), CoverArt: t.cover(), Size: t.Size,
		ContentType: contentType(t.Path), Suffix: suffix(t.Path), Starred: starred(t.ud),
		Duration: int(t.Duration), BitRate: t.Bitrate / 1000, Path: fakePath(t), PlayCount: t.plays,
		DiscNumber: t.ParentIndex, Created: tp(t.AddedAt), AlbumID: t.AlbumID, ArtistID: t.ArtistID,
		Type: "music", UserRating: t.rating, BookmarkPosition: t.bookmark,

		SortName: t.SortName, MediaType: "song", Isrc: Array[string]{}, Genres: itemGenres(t.Genres),
		ChannelCount: s.Channels, SamplingRate: s.Rate, Moods: Array[string]{},
		Artists: ref(t.ArtistID, t.Artist), DisplayArtist: t.Artist,
		AlbumArtists:       ref(t.albumArtistID, orStr(t.AlbumArtist, t.Artist)),
		DisplayAlbumArtist: orStr(t.AlbumArtist, t.Artist), Contributors: Array[contributor]{},
	}
	if t.plays > 0 {
		c.Played = tp(t.last)
	}
	return c
}

func starred(u ud) *time.Time {
	if !u.fav {
		return nil
	}
	if u.favAt.IsZero() {
		return tp(time.Now())
	}
	return tp(u.favAt)
}

func (al *album) coverID() string {
	if al.cover != "" {
		return al.ID
	}
	return ""
}

func (al *album) artistName() string { return orStr(al.AlbumArtist, al.Artist) }

// child é o álbum visto como pasta (getMusicDirectory, getAlbumList).
func (al *album) child() child {
	c := child{
		ID: al.ID, Parent: al.ArtistID, IsDir: true, Title: al.Name, Name: al.Name, Album: al.Name,
		Artist: al.artistName(), Year: al.Year, Genre: firstGenre(al.Genres), CoverArt: al.coverID(),
		Starred: starred(al.ud), Duration: int(al.dur), PlayCount: al.plays, Created: tp(al.AddedAt),
		ArtistID: al.ArtistID, UserRating: al.rating, SongCount: al.songs,

		SortName: al.SortName, MediaType: "album", Isrc: Array[string]{}, Genres: itemGenres(al.Genres),
		Moods: Array[string]{}, Artists: ref(al.ArtistID, al.artistName()), DisplayArtist: al.artistName(),
		AlbumArtists: ref(al.ArtistID, al.artistName()), DisplayAlbumArtist: al.artistName(),
		Contributors: Array[contributor]{},
	}
	if al.plays > 0 {
		c.Played = tp(al.last)
	}
	return c
}

func releaseDate(it *store.Item) itemDate {
	var d itemDate
	if it.Premiere != "" {
		fmt.Sscanf(it.Premiere, "%d-%d-%d", &d.Year, &d.Month, &d.Day)
		return d
	}
	d.Year = it.Year
	return d
}

func (al *album) id3() albumID3 {
	x := albumID3{
		ID: al.ID, Name: al.Name, Artist: al.artistName(), ArtistID: al.ArtistID, CoverArt: al.coverID(),
		SongCount: al.songs, Duration: int(al.dur), PlayCount: al.plays, Created: ms(al.AddedAt),
		Starred: starred(al.ud), Year: al.Year, Genre: firstGenre(al.Genres),

		UserRating: al.rating, Genres: itemGenres(al.Genres), SortName: al.SortName,
		DiscTitles: Array[discTitle]{}, ReleaseDate: releaseDate(al.Item), ReleaseTypes: Array[string]{},
		RecordLabels: Array[recordLabel]{}, Moods: Array[string]{}, Artists: ref(al.ArtistID, al.artistName()),
		DisplayArtist: al.artistName(),
	}
	if al.plays > 0 {
		x.Played = tp(al.last)
	}
	return x
}

func (ar *artistRow) coverID() string {
	if ar.coverAlbum != "" {
		return ar.ID
	}
	return ""
}

func (ar *artistRow) plain() artist {
	return artist{ID: ar.ID, Name: ar.Name, Starred: starred(ar.ud), UserRating: ar.rating, CoverArt: ar.coverID()}
}

func (ar *artistRow) id3() artistID3 {
	roles := Array[string]{}
	if ar.albums > 0 {
		roles = append(roles, "albumartist")
	}
	return artistID3{
		ID: ar.ID, Name: ar.Name, CoverArt: ar.coverID(), AlbumCount: ar.albums, Starred: starred(ar.ud),
		UserRating: ar.rating, SortName: ar.SortName, Roles: roles,
	}
}

func trackChildren(ts []*track) []child {
	out := make([]child, len(ts))
	for i, t := range ts {
		out[i] = t.child()
	}
	return out
}

func albumChildren(as []*album) []child {
	out := make([]child, len(as))
	for i, a := range as {
		out[i] = a.child()
	}
	return out
}

func albumID3s(as []*album) []albumID3 {
	out := make([]albumID3, len(as))
	for i, a := range as {
		out[i] = a.id3()
	}
	return out
}

// indexName é a letra do índice: sem artigo, sem acento, "#" se não for letra.
func indexName(name string) string {
	n := stripArticle(name)
	for _, r := range n {
		r = unicode.ToUpper(fold(r))
		if unicode.IsLetter(r) {
			return string(r)
		}
		return "#"
	}
	return "#"
}

func stripArticle(name string) string {
	n := strings.TrimSpace(name)
	low := strings.ToLower(n)
	for _, a := range strings.Fields(ignoredArticles) {
		p := strings.ToLower(a) + " "
		if strings.HasPrefix(low, p) && len(n) > len(p) {
			return strings.TrimSpace(n[len(p):])
		}
	}
	return n
}

// fold tira o acento das letras latinas mais comuns.
func fold(r rune) rune {
	if f, ok := accents[r]; ok {
		return f
	}
	return r
}

var accents = func() map[rune]rune {
	m := map[rune]rune{}
	for base, list := range map[rune]string{
		'a': "áàâãäåā", 'e': "éèêëē", 'i': "íìîïī", 'o': "óòôõöøō", 'u': "úùûüū", 'c': "ç", 'n': "ñ", 'y': "ýÿ",
		'A': "ÁÀÂÃÄÅĀ", 'E': "ÉÈÊËĒ", 'I': "ÍÌÎÏĪ", 'O': "ÓÒÔÕÖØŌ", 'U': "ÚÙÛÜŪ", 'C': "Ç", 'N': "Ñ", 'Y': "Ý",
	} {
		for _, r := range list {
			m[r] = base
		}
	}
	return m
}()

// norm deixa minúsculo e sem acento (busca).
func norm(s string) string {
	return strings.Map(func(r rune) rune { return unicode.ToLower(fold(r)) }, s)
}
