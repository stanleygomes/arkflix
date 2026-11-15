package jellyfin

import "net/http"

const (
	get  = http.MethodGet
	post = http.MethodPost
	del  = http.MethodDelete
)

// table lista as rotas. A ordem importa: o literal vem antes do {id}.
func (s *Server) table() []route {
	s.routes = nil
	pub, add := s.addPublic, s.add

	// público
	pub(get, "/System/Info/Public", s.systemInfoPublic)
	pub("", "/System/Ping", s.ping)
	pub(get, "/Users/Public", emptyList)
	pub(post, "/Users/AuthenticateByName", s.login)
	pub(get, "/Branding/Configuration", s.branding)
	pub(get, "/Branding/Css", s.brandingCSS)
	pub(get, "/Branding/Css.css", s.brandingCSS)
	pub(get, "/QuickConnect/Enabled", jsonFalse)
	pub(get, "/Items/{id}/Images/{type}", s.itemImage)
	pub(get, "/Items/{id}/Images/{type}/{index}", s.itemImage)
	pub(get, "/Artists/{name}/Images/{type}", s.artistImage)
	pub(get, "/Artists/{name}/Images/{type}/{index}", s.artistImage)
	pub(get, "/web", webMissing)
	pub(get, "/web/{any}", webMissing)
	pub(get, "/", webMissing)

	// sistema
	add(get, "/System/Info", s.systemInfo)
	add(get, "/System/Endpoint", s.endpoint)
	add(get, "/System/Configuration", s.systemConfig)
	add(get, "/System/Configuration/{key}", emptyObject)
	add(get, "/web/ConfigurationPages", emptyList)
	add(get, "/Plugins", emptyList)
	add(get, "/Packages", emptyList)
	add(get, "/ScheduledTasks", emptyList)
	add(get, "/Localization/Cultures", s.cultures)
	add(get, "/Localization/Countries", s.countries)
	add(get, "/Localization/ParentalRatings", emptyList)
	add(get, "/Localization/Options", s.localizationOptions)
	add(get, "/Playback/BitrateTest", s.bitrateTest)
	add(get, "/socket", s.socket)

	// usuários e sessões
	add(get, "/Users/Me", s.me)
	add(get, "/Users", s.users)
	add(get, "/Users/{userId}", s.userByID)
	add(post, "/Users/{userId}/Configuration", noContent)
	add(post, "/Users/Configuration", noContent)
	add(post, "/Sessions/Logout", s.logout)
	add(get, "/Sessions", s.sessionsList)
	add(post, "/Sessions/Capabilities", noContent)
	add(post, "/Sessions/Capabilities/Full", noContent)
	add(post, "/Sessions/Viewing", noContent)
	add(get, "/Devices", s.devices)
	add(get, "/DisplayPreferences/{id}", s.displayPrefsGet)
	add(post, "/DisplayPreferences/{id}", s.displayPrefsSet)

	// bibliotecas
	add(get, "/Users/{userId}/Views", s.userViews)
	add(get, "/UserViews", s.userViews)
	add(get, "/Users/{userId}/GroupingOptions", emptyList)
	add(get, "/UserViews/GroupingOptions", emptyList)
	add(get, "/Library/VirtualFolders", s.virtualFolders)
	add(get, "/Library/MediaFolders", s.mediaFolders)

	// itens (literais antes de {id})
	for _, pre := range []string{"", "/Users/{userId}"} {
		add(get, pre+"/Items/Resume", s.resume)
		add(get, pre+"/Items/Latest", s.latest)
		add(get, pre+"/Items/Root", s.rootItem)
		add(get, pre+"/Items", s.getItems)
	}
	add(get, "/UserItems/Resume", s.resume)
	add(get, "/Items/Filters", s.itemFilters)
	add(get, "/Items/Filters2", s.itemFilters2)
	add(get, "/Items/Counts", s.counts)
	add(get, "/Items/Suggestions", emptyResult)
	add(get, "/Users/{userId}/Suggestions", emptyResult)
	add(get, "/Shows/NextUp", s.nextUp)
	add(get, "/Shows/{id}/Seasons", s.seasons)
	add(get, "/Shows/{id}/Episodes", s.episodes)
	add(get, "/Shows/{id}/Similar", s.similar)
	add(get, "/Search/Hints", s.searchHints)
	add(get, "/Genres", s.genresHandler(false))
	add(get, "/Genres/{name}", s.genreHandler(false))
	add(get, "/MusicGenres", s.genresHandler(true))
	add(get, "/MusicGenres/InstantMix", s.instantMix)
	add(get, "/MusicGenres/{name}", s.genreHandler(true))
	add(get, "/MusicGenres/{id}/InstantMix", s.instantMix)
	add(get, "/Persons", emptyResult)
	add(get, "/Studios", emptyResult)
	add(get, "/Trailers", emptyResult)
	add(get, "/Movies/Recommendations", emptyList)
	add(get, "/Playlists", s.playlistsResult)
	add(post, "/Playlists", s.createPlaylist)
	add(get, "/Playlists/{id}", s.getPlaylist)
	add(post, "/Playlists/{id}", s.updatePlaylist)
	add(get, "/Playlists/{id}/Items", s.getPlaylistItems)
	add(post, "/Playlists/{id}/Items", s.addToPlaylist)
	add(del, "/Playlists/{id}/Items", s.removeFromPlaylist)
	add(post, "/Playlists/{id}/Items/{itemId}/Move/{newIndex}", s.movePlaylistItem)
	add(get, "/Playlists/{id}/Users", s.playlistUsers)
	add(get, "/Playlists/{id}/Users/{userId}", s.playlistUser)
	add(get, "/Artists", s.artists(false))
	add(get, "/Artists/AlbumArtists", s.artists(true))
	add(get, "/Artists/InstantMix", s.instantMix)
	add(get, "/Artists/{name}", s.artist)
	add(get, "/Artists/{id}/InstantMix", s.instantMix)
	add(get, "/Albums/{id}/InstantMix", s.instantMix)
	add(get, "/Albums/{id}/Similar", s.similar)
	add(get, "/Artists/{id}/Similar", s.similar)
	add(get, "/Songs/{id}/InstantMix", s.instantMix)
	add(get, "/Items/{id}/InstantMix", s.instantMix)
	add(get, "/MediaSegments/{id}", emptyResult)

	for _, pre := range []string{"", "/Users/{userId}"} {
		add(get, pre+"/Items/{id}", s.getItem)
		add(del, pre+"/Items/{id}", s.deleteItem)
		add(get, pre+"/Items/{id}/Intros", emptyResult)
		add(get, pre+"/Items/{id}/LocalTrailers", emptyList)
		add(get, pre+"/Items/{id}/SpecialFeatures", emptyList)
		add("", pre+"/Items/{id}/UserData", s.itemUserData)
		add(post, pre+"/Items/{id}/Rating", s.rating)
		add(del, pre+"/Items/{id}/Rating", s.rating)
	}
	add(get, "/Items/{id}/Similar", s.similar)
	add(get, "/Items/{id}/ThemeMedia", s.themeMedia)
	add(get, "/Items/{id}/ThemeSongs", emptyResult)
	add(get, "/Items/{id}/ThemeVideos", emptyResult)
	add(get, "/Items/{id}/Ancestors", s.ancestors)
	add(get, "/Items/{id}/Images", s.imageInfos)
	add(get, "/Items/{id}/ExternalIdInfos", emptyList)
	add(get, "/Items/{id}/CriticReviews", emptyResult)
	add(get, "/Items/{id}/Download", s.download)
	add(get, "/Items/{id}/File", s.download)
	add(get, "/Videos/{id}/AdditionalParts", emptyResult)
	add("", "/UserItems/{id}/UserData", s.itemUserData)
	add(post, "/UserItems/{id}/Rating", s.rating)
	add(del, "/UserItems/{id}/Rating", s.rating)

	// estado do usuário
	add(post, "/Users/{userId}/FavoriteItems/{id}", s.favorite)
	add(del, "/Users/{userId}/FavoriteItems/{id}", s.favorite)
	add(post, "/UserFavoriteItems/{id}", s.favorite)
	add(del, "/UserFavoriteItems/{id}", s.favorite)
	add(post, "/Users/{userId}/PlayedItems/{id}", s.played)
	add(del, "/Users/{userId}/PlayedItems/{id}", s.played)
	add(post, "/UserPlayedItems/{id}", s.played)
	add(del, "/UserPlayedItems/{id}", s.played)

	// relatório de reprodução
	add(post, "/Sessions/Playing", s.reportBody("start"))
	add(post, "/Sessions/Playing/Progress", s.reportBody("progress"))
	add(post, "/Sessions/Playing/Stopped", s.reportBody("stop"))
	add(post, "/Sessions/Playing/Ping", s.playingPing)
	for _, pre := range []string{"", "/Users/{userId}"} {
		add(post, pre+"/PlayingItems/{id}", s.reportLegacy("start"))
		add(del, pre+"/PlayingItems/{id}", s.reportLegacy("start"))
		add(post, pre+"/PlayingItems/{id}/Progress", s.reportLegacy("progress"))
	}

	// reprodução
	add("", "/Items/{id}/PlaybackInfo", s.playbackInfo)
	add(get, "/Videos/{id}/stream", s.videoStream)
	add(get, "/Videos/{id}/stream.{container}", s.videoStream)
	add(get, "/Videos/{id}/master.m3u8", s.hlsMaster)
	add(get, "/Videos/{id}/main.m3u8", s.hlsMain)
	add(get, "/Videos/{id}/hls1/{playlist}/{seg}.{ext}", s.hlsSegment)
	add(del, "/Videos/ActiveEncodings", s.stopEncodings)
	add(get, "/Videos/{id}/{msid}/Subtitles/{index}/Stream.{format}", s.subtitle)
	add(get, "/Videos/{id}/{msid}/Subtitles/{index}/{ticks}/Stream.{format}", s.subtitle)
	add(get, "/Audio/{id}/universal", s.audioUniversal)
	add(get, "/Audio/{id}/stream", s.audioStream)
	add(get, "/Audio/{id}/stream.{container}", s.audioStream)
	return s.routes
}

// sem cliente web embutido
func webMissing(c *rc) {
	c.fail(http.StatusNotFound, "arkflix-server não traz o jellyfin-web: use um app cliente")
}
