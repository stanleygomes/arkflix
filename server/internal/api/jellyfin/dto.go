package jellyfin

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

const ticksPerSecond = 10_000_000

func ticks(sec float64) int64 { return int64(sec * ticksPerSecond) }

// jfTime formata como o Jellyfin: "2024-01-02T03:04:05.0000000Z".
func jfTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.0000000Z")
}

func parseTime(s string) time.Time {
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func hash(parts ...string) string {
	sum := md5.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

// Tipo do Jellyfin de cada tipo do acervo (jogos ficam de fora).
var kindType = map[string]string{
	store.KindMovie:   "Movie",
	store.KindSeries:  "Series",
	store.KindSeason:  "Season",
	store.KindEpisode: "Episode",
	store.KindArtist:  "MusicArtist",
	store.KindAlbum:   "MusicAlbum",
	store.KindTrack:   "Audio",
}

var typeKinds = map[string][]string{
	"movie":       {store.KindMovie},
	"series":      {store.KindSeries},
	"season":      {store.KindSeason},
	"episode":     {store.KindEpisode},
	"musicartist": {store.KindArtist},
	"musicalbum":  {store.KindAlbum},
	"audio":       {store.KindTrack},
}

func isFolder(kind string) bool {
	switch kind {
	case store.KindSeries, store.KindSeason, store.KindArtist, store.KindAlbum:
		return true
	}
	return false
}

func mediaType(kind string) string {
	switch kind {
	case store.KindMovie, store.KindEpisode:
		return "Video"
	case store.KindTrack:
		return "Audio"
	}
	return "Unknown"
}

type nameID struct {
	Name string
	Id   string
}

// itemDto é o BaseItemDto. Campos que o Jellyfin manda null ficam de fora
// (omitempty); listas sempre vão (decodificadores rígidos agradecem).
type itemDto struct {
	Name                    string
	ServerId                string
	Id                      string
	Etag                    string `json:",omitempty"`
	DateCreated             string `json:",omitempty"`
	DateLastMediaAdded      string `json:",omitempty"`
	CanDelete               bool
	CanDownload             bool
	SortName                string            `json:",omitempty"`
	PremiereDate            string            `json:",omitempty"`
	ExternalUrls            []any             `json:"ExternalUrls"`
	Path                    string            `json:",omitempty"`
	Overview                string            `json:",omitempty"`
	Taglines                []string          `json:"Taglines"`
	Genres                  []string          `json:"Genres"`
	RunTimeTicks            int64             `json:",omitempty"`
	ProductionYear          int               `json:",omitempty"`
	IndexNumber             *int              `json:",omitempty"`
	ParentIndexNumber       *int              `json:",omitempty"`
	ProviderIds             map[string]string `json:"ProviderIds"`
	IsFolder                bool
	ParentId                string            `json:",omitempty"`
	Type                    string            `json:"Type"`
	People                  []any             `json:"People"`
	Studios                 []nameID          `json:"Studios"`
	GenreItems              []nameID          `json:"GenreItems"`
	ParentLogoItemId        string            `json:",omitempty"`
	ParentBackdropItemId    string            `json:",omitempty"`
	ParentBackdropImageTags []string          `json:",omitempty"`
	LocalTrailerCount       int               `json:"LocalTrailerCount"`
	UserData                *userDataDto      `json:",omitempty"`
	RecursiveItemCount      *int              `json:",omitempty"`
	ChildCount              *int              `json:",omitempty"`
	SeriesName              string            `json:",omitempty"`
	SeriesId                string            `json:",omitempty"`
	SeasonId                string            `json:",omitempty"`
	SpecialFeatureCount     int               `json:"SpecialFeatureCount"`
	DisplayPreferencesId    string            `json:",omitempty"`
	Tags                    []string          `json:"Tags"`
	PrimaryImageAspectRatio float64           `json:",omitempty"`
	Artists                 []string          `json:",omitempty"`
	ArtistItems             []nameID          `json:",omitempty"`
	Album                   string            `json:",omitempty"`
	CollectionType          string            `json:",omitempty"`
	AlbumId                 string            `json:",omitempty"`
	AlbumPrimaryImageTag    string            `json:",omitempty"`
	SeriesPrimaryImageTag   string            `json:",omitempty"`
	AlbumArtist             string            `json:",omitempty"`
	AlbumArtists            []nameID          `json:",omitempty"`
	SeasonName              string            `json:",omitempty"`
	MediaStreams            []streamDto       `json:",omitempty"`
	VideoType               string            `json:",omitempty"`
	MediaSourceCount        int               `json:",omitempty"`
	ImageTags               map[string]string `json:"ImageTags"`
	BackdropImageTags       []string          `json:"BackdropImageTags"`
	ParentLogoImageTag      string            `json:",omitempty"`
	ImageBlurHashes         map[string]any    `json:"ImageBlurHashes"`
	Chapters                []any             `json:"Chapters"`
	LocationType            string            `json:",omitempty"`
	MediaType               string            `json:"MediaType"`
	LockedFields            []string          `json:"LockedFields"`
	LockData                bool
	Width                   int         `json:",omitempty"`
	Height                  int         `json:",omitempty"`
	Container               string      `json:",omitempty"`
	MediaSources            []sourceDto `json:",omitempty"`
	HasSubtitles            bool        `json:",omitempty"`
	PlayAccess              string      `json:",omitempty"`
	AlbumCount              *int        `json:",omitempty"`
	SongCount               *int        `json:",omitempty"`
	MovieCount              *int        `json:",omitempty"`
	SeriesCount             *int        `json:",omitempty"`
	EpisodeCount            *int        `json:",omitempty"`
	PlaylistItemId          string      `json:",omitempty"`
}

// userDataDto é o UserItemDataDto.
type userDataDto struct {
	PlayedPercentage      float64 `json:",omitempty"`
	UnplayedItemCount     *int    `json:",omitempty"`
	PlaybackPositionTicks int64
	PlayCount             int
	IsFavorite            bool
	LastPlayedDate        string `json:",omitempty"`
	Played                bool
	Key                   string
	ItemId                string
}

func newItemDto(serverID string) itemDto {
	return itemDto{
		ServerId: serverID, CanDownload: true, ExternalUrls: []any{}, Taglines: []string{}, Genres: []string{},
		ProviderIds: map[string]string{}, People: []any{}, Studios: []nameID{}, GenreItems: []nameID{}, Tags: []string{},
		ImageTags: map[string]string{}, BackdropImageTags: []string{}, ImageBlurHashes: map[string]any{},
		Chapters: []any{}, LockedFields: []string{}, MediaType: "Unknown", PlayAccess: "Full",
	}
}

func intp(n int) *int { return &n }

// pair só sai com id (cliente Kotlin/Swift quebra com UUID vazio).
func pair(name, id string) []nameID {
	if name == "" || id == "" {
		return nil
	}
	return []nameID{{Name: name, Id: id}}
}

func genreID(name string) string { return store.ID("genre", strings.ToLower(name)) }

// lookup busca um item com memória por requisição (pais repetem muito).
func (c *rc) lookup(id string) *store.Item {
	if id == "" {
		return nil
	}
	if c.memo == nil {
		c.memo = map[string]*store.Item{}
	}
	if it, ok := c.memo[id]; ok {
		return it
	}
	it, err := c.s.d.Store.Item(c.ctx(), id)
	if err != nil {
		it = nil
	}
	c.memo[id] = it
	return it
}

// parentID é o pai lógico pro cliente navegar.
func (c *rc) parentID(it *store.Item) string {
	if it.ParentID != "" {
		return it.ParentID
	}
	return viewID(it.Library)
}

func premiere(it *store.Item) string {
	if it.Premiere != "" {
		if t := parseTime(it.Premiere); !t.IsZero() {
			return jfTime(t)
		}
	}
	if it.Year > 0 {
		return jfTime(time.Date(it.Year, 1, 1, 0, 0, 0, 0, time.UTC))
	}
	return ""
}

// dto monta o BaseItemDto completo de um item.
func (c *rc) dto(it *store.Item, ud store.UserData) itemDto {
	d := newItemDto(c.s.serverID)
	d.Name, d.Id, d.Type = it.Name, it.ID, kindType[it.Kind]
	d.Etag = hash(it.ID, it.UpdatedAt.String(), it.Path)[:16]
	d.DateCreated = jfTime(it.AddedAt)
	d.SortName = it.SortName
	d.PremiereDate = premiere(it)
	if !strings.HasPrefix(it.Path, "artist:") && !strings.HasPrefix(it.Path, "album:") {
		d.Path = it.Path
	}
	d.Overview, d.ProductionYear = it.Overview, it.Year
	d.Genres = append(d.Genres, it.Genres...)
	for _, g := range it.Genres {
		d.GenreItems = append(d.GenreItems, nameID{Name: g, Id: genreID(g)})
	}
	d.RunTimeTicks = ticks(it.Duration)
	d.IsFolder = isFolder(it.Kind)
	d.ParentId = c.parentID(it)
	d.MediaType = mediaType(it.Kind)
	d.LocationType = "FileSystem"
	d.DisplayPreferencesId = it.ID
	c.images(&d, it)

	switch it.Kind {
	case store.KindMovie, store.KindEpisode:
		d.VideoType, d.Width, d.Height, d.Container = "VideoFile", it.Width, it.Height, it.Container
		d.MediaStreams = c.streams(it, nil)
		d.MediaSources = []sourceDto{c.source(it, nil)}
		d.MediaSourceCount = 1
		for _, s := range it.Streams {
			d.HasSubtitles = d.HasSubtitles || s.Type == "subtitle"
		}
	case store.KindTrack:
		d.Container = it.Container
		d.MediaStreams = c.streams(it, nil)
		d.MediaSources = []sourceDto{c.source(it, nil)}
		d.MediaSourceCount = 1
	}

	switch it.Kind {
	case store.KindSeason:
		d.IndexNumber = intp(it.Index)
		c.seriesInfo(&d, it)
	case store.KindEpisode:
		d.IndexNumber, d.ParentIndexNumber = intp(it.Index), intp(it.ParentIndex)
		c.seriesInfo(&d, it)
		d.SeasonId = it.SeasonID
		if s := c.lookup(it.SeasonID); s != nil {
			d.SeasonName = s.Name
		}
	case store.KindTrack:
		if it.Index > 0 {
			d.IndexNumber = intp(it.Index)
		}
		if it.ParentIndex > 0 {
			d.ParentIndexNumber = intp(it.ParentIndex)
		}
		d.Album, d.AlbumId = it.Album, it.AlbumID
		d.AlbumArtist = it.AlbumArtist
		if it.Artist != "" {
			d.Artists = []string{it.Artist}
			d.ArtistItems = pair(it.Artist, it.ArtistID)
		}
		if al := c.lookup(it.AlbumID); al != nil {
			d.AlbumArtists = pair(al.AlbumArtist, al.ArtistID)
			if t := c.tag(al.Image); t != "" {
				d.AlbumPrimaryImageTag = t
			}
		}
	case store.KindAlbum:
		d.AlbumArtist = it.AlbumArtist
		if it.AlbumArtist != "" {
			d.Artists = []string{it.AlbumArtist}
			d.AlbumArtists = pair(it.AlbumArtist, it.ArtistID)
			d.ArtistItems = d.AlbumArtists
		}
		d.PrimaryImageAspectRatio = 1
	case store.KindSeries, store.KindMovie:
		d.PrimaryImageAspectRatio = 2.0 / 3
	case store.KindArtist:
		d.PrimaryImageAspectRatio = 1
	}
	d.UserData = c.userData(it, ud, &d)
	return d
}

func (c *rc) seriesInfo(d *itemDto, it *store.Item) {
	d.SeriesId = it.SeriesID
	sr := c.lookup(it.SeriesID)
	if sr == nil {
		return
	}
	d.SeriesName = sr.Name
	if t := c.tag(sr.Image); t != "" {
		d.SeriesPrimaryImageTag = t
	}
	if _, ok := d.ImageTags["Backdrop"]; !ok && len(d.BackdropImageTags) == 0 {
		if t := c.tag(sr.Backdrop); t != "" {
			d.ParentBackdropItemId, d.ParentBackdropImageTags = sr.ID, []string{t}
		}
	}
	if _, ok := d.ImageTags["Logo"]; !ok {
		if t := c.tag(sr.Logo); t != "" {
			d.ParentLogoItemId, d.ParentLogoImageTag = sr.ID, t
		}
	}
}

// userData monta o UserItemDataDto; pastas ganham contagens.
func (c *rc) userData(it *store.Item, ud store.UserData, d *itemDto) *userDataDto {
	u := &userDataDto{
		PlaybackPositionTicks: ticks(ud.Position), PlayCount: ud.PlayCount, IsFavorite: ud.Favorite,
		LastPlayedDate: jfTime(ud.LastPlayed), Played: ud.Played, Key: it.ID, ItemId: it.ID,
	}
	if it.Duration > 0 && ud.Position > 0 {
		u.PlayedPercentage = ud.Position / it.Duration * 100
	}
	if isFolder(it.Kind) {
		st := c.folderStats(it)
		if d != nil {
			d.ChildCount, d.RecursiveItemCount = intp(st.children), intp(st.recursive)
			if it.Kind == store.KindArtist {
				d.AlbumCount, d.SongCount = intp(st.children), intp(st.recursive)
			}
			if it.Kind == store.KindSeries || it.Kind == store.KindSeason {
				d.EpisodeCount = intp(st.recursive)
			}
		}
		if it.Kind == store.KindSeries || it.Kind == store.KindSeason {
			u.UnplayedItemCount = intp(st.unplayed)
			u.Played = st.recursive > 0 && st.unplayed == 0
			if st.recursive > 0 {
				u.PlayedPercentage = float64(st.recursive-st.unplayed) / float64(st.recursive) * 100
			}
		}
	}
	return u
}

type stats struct{ children, recursive, unplayed int }

func (c *rc) folderStats(it *store.Item) stats {
	db, ctx := c.s.d.Store.DB, c.ctx()
	var st stats
	uid := c.uid()
	switch it.Kind {
	case store.KindSeries:
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE kind = 'season' AND series_id = ?`, it.ID).Scan(&st.children)
		db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(u.played,0) = 0 THEN 1 ELSE 0 END),0)
FROM items e LEFT JOIN userdata u ON u.item_id = e.id AND u.user_id = ? WHERE e.kind = 'episode' AND e.series_id = ?`, uid, it.ID).
			Scan(&st.recursive, &st.unplayed)
	case store.KindSeason:
		db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(CASE WHEN COALESCE(u.played,0) = 0 THEN 1 ELSE 0 END),0)
FROM items e LEFT JOIN userdata u ON u.item_id = e.id AND u.user_id = ? WHERE e.kind = 'episode' AND e.season_id = ?`, uid, it.ID).
			Scan(&st.recursive, &st.unplayed)
		st.children = st.recursive
	case store.KindAlbum:
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE kind = 'track' AND album_id = ?`, it.ID).Scan(&st.children)
		st.recursive = st.children
	case store.KindArtist:
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE kind = 'album' AND artist_id = ?`, it.ID).Scan(&st.children)
		db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE kind = 'track' AND (artist_id = ? OR album_id IN
(SELECT id FROM items WHERE kind = 'album' AND artist_id = ?))`, it.ID, it.ID).Scan(&st.recursive)
	}
	return st
}

// item completo com o estado do usuário.
func (c *rc) itemDtoByID(id string) (itemDto, error) {
	if v, ok := c.s.viewByID(id); ok {
		return c.viewDto(v), nil
	}
	it, err := c.s.d.Store.Item(c.ctx(), id)
	if err != nil {
		return itemDto{}, err
	}
	if _, ok := kindType[it.Kind]; !ok {
		return itemDto{}, store.ErrNotFound
	}
	ud, err := c.s.d.Store.UserData(c.ctx(), c.uid(), it.ID)
	if err != nil {
		return itemDto{}, err
	}
	return c.dto(it, ud), nil
}
