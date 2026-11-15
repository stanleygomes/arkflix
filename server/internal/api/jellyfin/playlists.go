package jellyfin

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// As playlists são as listas de internal/lists (as mesmas do RomM, do
// Subsonic e do app): aparecem as do usuário e as públicas que têm algo
// que o Jellyfin mostra (ou estão vazias), só com esses itens.

var playlistsViewID = viewID("playlists")

// tipos do acervo que entram numa playlist do Jellyfin
var jfListKinds = []string{store.KindMovie, store.KindSeries, store.KindSeason, store.KindEpisode,
	store.KindArtist, store.KindAlbum, store.KindTrack}

type plist struct {
	l  *lists.List
	es []lists.Entry
}

func (c *rc) lists() *lists.Lists { return lists.New(c.s.d.Store) }

// playlists são as listas que o usuário vê no Jellyfin.
func (c *rc) playlists() ([]plist, error) {
	ls, err := c.lists().Visible(c.ctx(), c.uid())
	if err != nil {
		return nil, err
	}
	var out []plist
	for _, l := range ls {
		es, err := c.lists().Entries(c.ctx(), l.ID)
		if err != nil {
			return nil, err
		}
		if lists.HasKinds(es, jfListKinds...) {
			out = append(out, plist{l, es})
		}
	}
	return out, nil
}

// userList carrega uma lista que o usuário pode ver (ok=false: não é lista).
func (c *rc) userList(id string) (plist, bool) {
	if id == "" {
		return plist{}, false
	}
	l, err := c.lists().Get(c.ctx(), id)
	if err != nil || (c.user != nil && !l.CanSee(c.uid())) {
		return plist{}, false
	}
	es, err := c.lists().Entries(c.ctx(), l.ID)
	if err != nil {
		return plist{}, false
	}
	return plist{l, es}, true
}

// playlistImage: a capa da playlist é a do primeiro item que tiver.
func (c *rc) playlistImage(p plist) string {
	for _, id := range lists.ItemIDs(p.es, jfListKinds...) {
		if it := c.lookup(id); it != nil {
			if src := c.imageSrc(it, "Primary"); src != "" {
				return src
			}
		}
	}
	return ""
}

func (c *rc) playlistDto(p plist) itemDto {
	d := newItemDto(c.s.serverID)
	d.Name, d.Id, d.Type, d.IsFolder = p.l.Name, p.l.ID, "Playlist", true
	d.SortName, d.ParentId, d.LocationType = strings.ToLower(p.l.Name), playlistsViewID, "FileSystem"
	d.DateCreated, d.Overview, d.DisplayPreferencesId = jfTime(p.l.Created), p.l.Comment, p.l.ID
	d.Etag = hash(p.l.ID, p.l.Updated.String())[:16]
	d.CanDelete = p.l.UserID == c.uid()
	var audio, video, n int
	var dur float64
	for _, id := range lists.ItemIDs(p.es, jfListKinds...) {
		it := c.lookup(id)
		if it == nil {
			continue
		}
		n++
		dur += it.Duration
		switch mediaType(it.Kind) {
		case "Audio":
			audio++
		case "Video":
			video++
		}
	}
	switch {
	case audio > 0 && video == 0:
		d.MediaType = "Audio"
	case video > 0 && audio == 0:
		d.MediaType = "Video"
	}
	d.ChildCount, d.RecursiveItemCount, d.RunTimeTicks = intp(n), intp(n), ticks(dur)
	if t := c.tag(c.playlistImage(p)); t != "" {
		d.ImageTags["Primary"] = t
		d.PrimaryImageAspectRatio = 1
	}
	d.UserData = &userDataDto{Key: p.l.ID, ItemId: p.l.ID}
	return d
}

// playlistsViewDto é a pasta "Playlists" das bibliotecas.
func (c *rc) playlistsViewDto(n int) itemDto {
	d := newItemDto(c.s.serverID)
	d.Name, d.Id, d.Type, d.CollectionType = "Playlists", playlistsViewID, "ManualPlaylistsFolder", "playlists"
	d.SortName, d.IsFolder, d.ParentId, d.LocationType = "playlists", true, rootID, "FileSystem"
	d.DisplayPreferencesId, d.Etag, d.ChildCount = playlistsViewID, playlistsViewID[:16], intp(n)
	d.UserData = &userDataDto{Key: playlistsViewID, ItemId: playlistsViewID}
	return d
}

// playlistsResult responde a lista de playlists paginada.
func (s *Server) playlistsResult(c *rc) {
	ps, err := c.playlists()
	if err != nil {
		c.err(err)
		return
	}
	if t := strings.TrimSpace(c.qs("SearchTerm")); t != "" {
		ps = filter(ps, func(p plist) bool { return strings.Contains(store.Fold(p.l.Name), store.Fold(t)) })
	}
	out := []itemDto{}
	for _, p := range ps {
		out = append(out, c.playlistDto(p))
	}
	c.json(page(out, c))
}

func page(out []itemDto, c *rc) queryResult {
	total, start, limit := len(out), max(c.qi("StartIndex", 0), 0), c.qi("Limit", 0)
	out = out[min(start, total):]
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return result(out, total, start)
}

// playlistItems responde os itens da playlist, na ordem, com
// PlaylistItemId (a entrada, pra tirar).
func (s *Server) playlistItems(c *rc, p plist) {
	ids := lists.ItemIDs(p.es, jfListKinds...)
	q := query{}
	q.in("i.id", ids)
	kinds, ok := c.kinds()
	if !ok {
		c.json(result(nil, 0, 0))
		return
	}
	if kinds != nil {
		q.in("i.kind", kinds)
	}
	c.filters(&q)
	rows, _, err := c.run(q)
	if err != nil {
		c.err(err)
		return
	}
	byID := map[string]row{}
	for _, r := range rows {
		byID[r.it.ID] = r
	}
	out := []itemDto{}
	for _, e := range p.es {
		if r, ok := byID[e.ItemID]; ok {
			d := c.dto(r.it, r.ud)
			d.PlaylistItemId = e.ID
			out = append(out, d)
		}
	}
	c.json(page(out, c))
}

// onlyPlaylists: IncludeItemTypes pede só Playlist.
func (c *rc) onlyPlaylists() bool {
	types := c.ql("IncludeItemTypes", ",|")
	return len(types) > 0 && !slices.ContainsFunc(types, func(t string) bool { return !strings.EqualFold(t, "playlist") })
}

// playlistQuery trata o /Items que é de playlist; false = não era.
func (s *Server) playlistQuery(c *rc) bool {
	parent := store.NormalizeID(c.qs("ParentId"))
	switch {
	case parent == playlistsViewID || (c.onlyPlaylists() && (parent == "" || parent == rootID)):
		s.playlistsResult(c)
		return true
	case parent != "" && parent != rootID:
		if _, ok := s.viewByID(parent); ok {
			return false
		}
		if p, ok := c.userList(parent); ok {
			s.playlistItems(c, p)
			return true
		}
	}
	return false
}

// GET /Playlists/{id}/Items
func (s *Server) getPlaylistItems(c *rc) {
	p, ok := c.userList(c.pid("id"))
	if !ok {
		c.notFound()
		return
	}
	s.playlistItems(c, p)
}

// ownPlaylist exige que a playlist seja do usuário (403 se for de outro).
func (c *rc) ownPlaylist() (plist, bool) {
	p, ok := c.userList(c.pid("id"))
	if !ok {
		c.fail(http.StatusNotFound, "Playlist not found")
		return p, false
	}
	if p.l.UserID != c.user.ID {
		c.fail(http.StatusForbidden, "Forbidden")
		return p, false
	}
	return p, true
}

// jfItemIDs fica só com ids de itens que existem e entram em playlist.
func (c *rc) jfItemIDs(ids []string) []string {
	var out []string
	for _, id := range normIDs(ids) {
		if it := c.lookup(id); it != nil && slices.Contains(jfListKinds, it.Kind) {
			out = append(out, it.ID)
		}
	}
	return out
}

// POST /Playlists (CreatePlaylistDto no corpo, ou name/ids na query)
func (s *Server) createPlaylist(c *rc) {
	var body struct {
		Name     string
		Ids      []string
		IsPublic *bool
	}
	if err := c.body(&body); err != nil {
		c.fail(http.StatusBadRequest, "corpo inválido")
		return
	}
	name, ids := c.qs("Name"), c.ql("Ids", ",")
	if name == "" {
		name = body.Name
	}
	if len(ids) == 0 {
		ids = body.Ids
	}
	l, err := c.lists().Create(c.ctx(), c.user.ID, name)
	if errors.Is(err, lists.ErrName) {
		c.fail(http.StatusBadRequest, "Name is required")
		return
	}
	if err != nil {
		c.err(err)
		return
	}
	if body.IsPublic != nil {
		c.lists().Update(c.ctx(), l.ID, lists.Changes{Public: body.IsPublic})
	}
	if err := c.lists().Add(c.ctx(), l.ID, c.jfItemIDs(ids)); err != nil {
		c.err(err)
		return
	}
	c.json(map[string]string{"Id": l.ID})
}

// POST /Playlists/{id} (UpdatePlaylistDto: Name, Ids, IsPublic)
func (s *Server) updatePlaylist(c *rc) {
	p, ok := c.ownPlaylist()
	if !ok {
		return
	}
	var body struct {
		Name     *string
		Ids      *[]string
		IsPublic *bool
	}
	if err := c.body(&body); err != nil {
		c.fail(http.StatusBadRequest, "corpo inválido")
		return
	}
	if err := c.lists().Update(c.ctx(), p.l.ID, lists.Changes{Name: body.Name, Public: body.IsPublic}); err != nil {
		c.fail(http.StatusBadRequest, err.Error())
		return
	}
	if body.Ids != nil {
		if err := c.lists().SetKinds(c.ctx(), p.l.ID, jfListKinds, c.jfItemIDs(*body.Ids)); err != nil {
			c.err(err)
			return
		}
	}
	c.noContent()
}

// GET /Playlists/{id}: PlaylistDto
func (s *Server) getPlaylist(c *rc) {
	p, ok := c.userList(c.pid("id"))
	if !ok {
		c.fail(http.StatusNotFound, "Playlist not found")
		return
	}
	c.json(map[string]any{"OpenAccess": p.l.Public, "Shares": []any{}, "ItemIds": nonNil(lists.ItemIDs(p.es, jfListKinds...))})
}

// GET /Playlists/{id}/Users: os compartilhamentos (não tem; só o dono vê)
func (s *Server) playlistUsers(c *rc) {
	if _, ok := c.ownPlaylist(); ok {
		c.json([]any{})
	}
}

// GET /Playlists/{id}/Users/{userId}: o dono pode editar
func (s *Server) playlistUser(c *rc) {
	p, ok := c.userList(c.pid("id"))
	if !ok || store.NormalizeID(c.p["userId"]) != p.l.UserID {
		c.fail(http.StatusNotFound, "Playlist not found")
		return
	}
	c.json(map[string]any{"UserId": p.l.UserID, "CanEdit": true})
}

// POST /Playlists/{id}/Items?ids=
func (s *Server) addToPlaylist(c *rc) {
	p, ok := c.ownPlaylist()
	if !ok {
		return
	}
	if err := c.lists().Add(c.ctx(), p.l.ID, c.jfItemIDs(c.ql("Ids", ","))); err != nil {
		c.err(err)
		return
	}
	c.noContent()
}

// DELETE /Playlists/{id}/Items?entryIds=
func (s *Server) removeFromPlaylist(c *rc) {
	p, ok := c.ownPlaylist()
	if !ok {
		return
	}
	if err := c.lists().Remove(c.ctx(), p.l.ID, c.ql("EntryIds", ",")); err != nil {
		c.err(err)
		return
	}
	c.noContent()
}

// POST /Playlists/{id}/Items/{itemId}/Move/{newIndex}
func (s *Server) movePlaylistItem(c *rc) {
	p, ok := c.ownPlaylist()
	if !ok {
		return
	}
	idx := 0
	for _, r := range c.p["newIndex"] {
		if r < '0' || r > '9' {
			c.fail(http.StatusBadRequest, "índice inválido")
			return
		}
		idx = idx*10 + int(r-'0')
	}
	if err := c.lists().Move(c.ctx(), p.l.ID, c.p["itemId"], idx); err != nil {
		c.err(err)
		return
	}
	c.noContent()
}

// DELETE /Items/{id}: só playlist (mídia não se apaga por aqui).
func (s *Server) deleteItem(c *rc) {
	if _, ok := c.userList(c.pid("id")); !ok {
		c.fail(http.StatusForbidden, "Forbidden")
		return
	}
	p, ok := c.ownPlaylist()
	if !ok {
		return
	}
	if err := c.lists().Delete(c.ctx(), p.l.ID); err != nil {
		c.err(err)
		return
	}
	c.noContent()
}
