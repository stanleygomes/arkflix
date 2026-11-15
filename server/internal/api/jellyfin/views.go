package jellyfin

import (
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// view é uma biblioteca (CollectionFolder). Jogos não entram.
type view struct {
	Lib, Name, Collection, Path, ID string
}

func viewID(lib string) string { return store.ID("view", lib) }

var rootID = store.ID("root", "jellyfin")

func (s *Server) views() []view {
	cfg := s.d.Config
	var out []view
	add := func(lib, name, coll, path string) {
		if path != "" {
			out = append(out, view{Lib: lib, Name: name, Collection: coll, Path: path, ID: viewID(lib)})
		}
	}
	add("movies", "Filmes", "movies", cfg.Movies)
	add("shows", "Séries", "tvshows", cfg.Shows)
	add("music", "Músicas", "music", cfg.Music)
	return out
}

func (s *Server) viewByID(id string) (view, bool) {
	id = store.NormalizeID(id)
	for _, v := range s.views() {
		if v.ID == id {
			return v, true
		}
	}
	return view{}, false
}

// topKind é o que aparece na raiz de cada biblioteca.
func topKind(lib string) string {
	switch lib {
	case "movies":
		return store.KindMovie
	case "shows":
		return store.KindSeries
	}
	return store.KindArtist
}

func (c *rc) viewDto(v view) itemDto {
	d := newItemDto(c.s.serverID)
	d.Name, d.Id, d.Type, d.CollectionType = v.Name, v.ID, "CollectionFolder", v.Collection
	d.SortName, d.IsFolder, d.ParentId, d.Path = strings.ToLower(v.Name), true, rootID, v.Path
	d.LocationType, d.DisplayPreferencesId, d.Etag = "FileSystem", v.ID, v.ID[:16]
	if t := c.tag(c.viewImage(v)); t != "" {
		d.ImageTags["Primary"] = t
		d.PrimaryImageAspectRatio = 16.0 / 9
	}
	var n int
	c.s.d.Store.DB.QueryRowContext(c.ctx(), `SELECT COUNT(*) FROM items WHERE library = ? AND kind = ?`, v.Lib, topKind(v.Lib)).Scan(&n)
	d.ChildCount = intp(n)
	if c.user != nil {
		d.UserData = &userDataDto{Key: v.ID, ItemId: v.ID}
	}
	return d
}

func (c *rc) rootDto() itemDto {
	d := newItemDto(c.s.serverID)
	d.Name, d.Id, d.Type, d.IsFolder, d.LocationType = "Media Folders", rootID, "UserRootFolder", true, "FileSystem"
	d.ChildCount = intp(len(c.s.views()))
	d.UserData = &userDataDto{Key: rootID, ItemId: rootID}
	return d
}

// GET /Users/{userId}/Views e /UserViews
func (s *Server) userViews(c *rc) { s.viewList(c, true) }

func (s *Server) viewList(c *rc, withPlaylists bool) {
	out := []itemDto{}
	for _, v := range s.views() {
		out = append(out, c.viewDto(v))
	}
	// a pasta de playlists só aparece se tiver alguma
	if ps, err := c.playlists(); withPlaylists && err == nil && len(ps) > 0 {
		out = append(out, c.playlistsViewDto(len(ps)))
	}
	c.json(result(out, len(out), 0))
}

// GET /Library/VirtualFolders (só leitura)
func (s *Server) virtualFolders(c *rc) {
	out := []map[string]any{}
	for _, v := range s.views() {
		out = append(out, map[string]any{
			"Name": v.Name, "Locations": []string{v.Path}, "CollectionType": v.Collection, "ItemId": v.ID,
			"PrimaryImageItemId": v.ID, "RefreshStatus": "Idle", "LibraryOptions": map[string]any{
				"Enabled": true, "EnableRealtimeMonitor": false, "PathInfos": []map[string]string{{"Path": v.Path}},
				"TypeOptions": []any{}, "DisabledSubtitleFetchers": []string{}, "SubtitleFetcherOrder": []string{},
				"MetadataSavers": []string{}, "LocalMetadataReaderOrder": []string{}, "DisabledLocalMetadataReaders": []string{},
				"SubtitleDownloadLanguages": []string{}, "DisabledMediaSegmentProviders": []string{},
				"MediaSegmentProviderOrder": []string{}, "DisabledLyricFetchers": []string{}, "LyricFetcherOrder": []string{},
				"CustomTagDelimiters": []string{}, "DelimiterWhitelist": []string{},
			},
		})
	}
	c.json(out)
}

// GET /Library/MediaFolders
func (s *Server) mediaFolders(c *rc) { s.viewList(c, false) }

// queryResult é o BaseItemDtoQueryResult.
type queryResult struct {
	Items            []itemDto
	TotalRecordCount int
	StartIndex       int
}

func result(items []itemDto, total, start int) queryResult {
	if items == nil {
		items = []itemDto{}
	}
	return queryResult{Items: items, TotalRecordCount: total, StartIndex: start}
}
