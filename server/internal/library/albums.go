package library

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// album agrupa as faixas de um álbum.
type album struct {
	name, artist string
	tracks       []*track
}

func (t *track) albumArtistName() string {
	return firstNonEmpty(t.albumArtist, t.artist)
}

// saveMusic agrupa em álbuns/artistas e grava tudo.
func (s *scanner) saveMusic(ts []*track) error {
	albums := map[string]*album{}
	var keys []string
	for _, t := range ts {
		k := "album:" + strings.ToLower(t.albumArtistName()) + "/" + strings.ToLower(t.album)
		if albums[k] == nil {
			albums[k] = &album{name: t.album, artist: t.albumArtistName()}
			keys = append(keys, k)
		}
		albums[k].tracks = append(albums[k].tracks, t)
	}
	artists := map[string]*store.Item{}
	for _, k := range keys {
		if err := s.saveAlbum(k, albums[k], artists); err != nil {
			return err
		}
	}
	return nil
}

// artistItem grava (uma vez por scan) o artista.
func (s *scanner) artistItem(name, dir string, artists map[string]*store.Item) (*store.Item, error) {
	key := "artist:" + strings.ToLower(name)
	it := artists[key]
	if it == nil {
		it = &store.Item{Kind: store.KindArtist, Path: key, Name: name}
		artists[key] = it
	} else if it.Image != "" || dir == "" {
		return it, nil
	}
	if dir != "" && dir != s.root {
		it.Image = s.find(dir, []string{"artist", "folder"}, imageExts)
	}
	return it, s.put(it)
}

func (s *scanner) saveAlbum(key string, al *album, artists map[string]*store.Item) error {
	sort.SliceStable(al.tracks, func(i, j int) bool {
		a, b := al.tracks[i], al.tracks[j]
		if a.disc != b.disc {
			return a.disc < b.disc
		}
		if a.num != b.num {
			return a.num < b.num
		}
		return a.f.path < b.f.path
	})
	dir := filepath.Dir(al.tracks[0].f.path)
	artistDir := ""
	if al.artist != variousArtists && dir != s.root {
		artistDir = filepath.Dir(dir)
	}
	ar, err := s.artistItem(al.artist, artistDir, artists)
	if err != nil {
		return err
	}
	it := &store.Item{Kind: store.KindAlbum, Path: key, Name: al.name, Artist: al.artist, AlbumArtist: al.artist,
		ParentID: ar.ID, ArtistID: ar.ID, RelPath: s.rel(dir), Image: s.albumImage(al, dir)}
	for _, t := range al.tracks {
		it.Duration += t.duration
		if it.Year == 0 {
			it.Year = t.year
		}
		it.Genres = appendNew(it.Genres, t.genres...)
	}
	it.Duration = round3(it.Duration)
	if err := s.put(it); err != nil {
		return err
	}
	for _, t := range al.tracks {
		if err := s.saveTrack(it, t, artists); err != nil {
			return err
		}
	}
	return nil
}

// albumImage: capa embutida da primeira faixa que tiver, senão cover.jpg.
func (s *scanner) albumImage(al *album, dir string) string {
	for _, t := range al.tracks {
		if t.picture {
			return "embedded:" + t.f.path
		}
	}
	return s.find(dir, []string{"cover", "folder", "front", "album", "albumart"}, imageExts)
}

func (s *scanner) saveTrack(al *store.Item, t *track, artists map[string]*store.Item) error {
	ar, err := s.artistItem(t.artist, "", artists)
	if err != nil {
		return err
	}
	it := &store.Item{Kind: store.KindTrack, Path: t.f.path, RelPath: s.rel(t.f.path), Name: t.title,
		Index: t.num, ParentIndex: t.disc, ParentID: al.ID, AlbumID: al.ID, ArtistID: ar.ID,
		Artist: t.artist, AlbumArtist: al.AlbumArtist, Album: al.Name, Year: t.year, Genres: t.genres,
		Container: ext(t.f.path), Duration: t.duration, Bitrate: t.bitrate, Streams: t.streams,
		Size: t.f.info.Size(), MTime: t.f.info.ModTime().Unix()}
	if t.picture {
		it.Image = "embedded:" + t.f.path
	}
	return s.put(it)
}

func appendNew(xs []string, add ...string) []string {
	for _, a := range add {
		dup := false
		for _, x := range xs {
			dup = dup || strings.EqualFold(x, a)
		}
		if !dup {
			xs = append(xs, a)
		}
	}
	return xs
}
