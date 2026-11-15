package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// variousArtists é o artista que o scanner dá pra coletânea sem tag.
const variousArtists = "Vários Artistas"

type mbSearch struct {
	Releases []struct {
		ID           string `json:"id"`
		Score        int    `json:"score"`
		Title        string `json:"title"`
		ArtistCredit []struct {
			Name   string `json:"name"`
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
		} `json:"artist-credit"`
		ReleaseGroup struct {
			ID string `json:"id"`
		} `json:"release-group"`
	} `json:"releases"`
}

// musicAlbum: release no MusicBrainz por artista do álbum + álbum e a
// capa da frente no Cover Art Archive (do release, senão do grupo).
func (f *Fetcher) musicAlbum(ctx context.Context, it *store.Item) (*patch, error) {
	if it.Name == "" {
		return nil, fmt.Errorf("%w: álbum sem nome", errNotFound)
	}
	artist := it.AlbumArtist
	if artist == variousArtists {
		artist = ""
	}
	q := `release:"` + lucene(it.Name) + `"`
	if artist != "" {
		q += ` AND artist:"` + lucene(artist) + `"`
	}
	var res mbSearch
	u := f.urls.musicbrainz + "/ws/2/release/?" + url.Values{"query": {q}, "fmt": {"json"}, "limit": {"10"}}.Encode()
	if err := f.http.getJSON(ctx, f.lim.mb, u, http.Header{"Accept": {"application/json"}}, &res); err != nil {
		return nil, err
	}
	var covers []string
	mbid := ""
	for _, r := range res.Releases {
		if r.Score < 90 || norm(r.Title) != norm(it.Name) || !creditMatch(r.ArtistCredit, artist) {
			continue
		}
		if mbid == "" {
			mbid = r.ID
		}
		covers = append(covers, f.urls.coverart+"/release/"+r.ID+"/front-500")
		if r.ReleaseGroup.ID != "" {
			covers = append(covers, f.urls.coverart+"/release-group/"+r.ReleaseGroup.ID+"/front-500")
		}
		if len(covers) >= 4 {
			break
		}
	}
	if mbid == "" {
		return nil, fmt.Errorf("%w: álbum %q", errNotFound, it.Name)
	}
	return &patch{source: "musicbrainz", images: map[string][]string{"image": covers},
		extra: map[string]any{"mbid": mbid}}, nil
}

func creditMatch(credits []struct {
	Name   string `json:"name"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
}, artist string) bool {
	if artist == "" {
		return true
	}
	var names []string
	for _, c := range credits {
		if norm(c.Name) == norm(artist) || norm(c.Artist.Name) == norm(artist) {
			return true
		}
		names = append(names, c.Name)
	}
	// "A & B" no crédito dividido em dois
	return norm(strings.Join(names, " & ")) == norm(artist)
}

// lucene escapa aspas e barra pra frase da busca.
func lucene(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}
