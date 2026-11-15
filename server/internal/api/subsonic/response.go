package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Versão do protocolo que o Navidrome anuncia.
const apiVersion = "1.16.1"

// Códigos de erro do Subsonic (e do OpenSubsonic).
const (
	errGeneric      = 0
	errMissingParam = 10
	errBadLogin     = 40
	errMultiAuth    = 43
	errBadAPIKey    = 44
	errNotAllowed   = 50
	errNotFound     = 70
)

var errMsgs = map[int]string{
	errGeneric:      "A generic error",
	errMissingParam: "Required parameter is missing",
	errBadLogin:     "Wrong username or password",
	errMultiAuth:    "Multiple conflicting authentication mechanisms provided",
	errBadAPIKey:    "Invalid API key",
	errNotAllowed:   "User is not authorized for the given operation",
	errNotFound:     "The requested data was not found",
}

// apiErr é um erro que vira <error code= message=>.
type apiErr struct {
	code int
	msg  string
}

func (e apiErr) Error() string { return e.msg }

// fail cria o erro; sem mensagem usa a padrão do código.
func fail(code int, format string, args ...any) error {
	if format == "" {
		return apiErr{code, errMsgs[code]}
	}
	return apiErr{code, fmt.Sprintf(format, args...)}
}

func toAPIErr(err error) apiErr {
	var ae apiErr
	switch {
	case errors.As(err, &ae):
		return ae
	case errors.Is(err, store.ErrNotFound):
		return apiErr{errNotFound, "data not found"}
	default:
		return apiErr{errGeneric, "Internal Server Error: " + err.Error()}
	}
}

// Array sai como [] no JSON mesmo vazio (o OpenSubsonic exige).
type Array[T any] []T

// MarshalJSON nunca devolve null.
func (a Array[T]) MarshalJSON() ([]byte, error) {
	if len(a) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]T(a))
}

// emptyObj é {} no JSON e nada no XML (replayGain, datas sem valor).
type emptyObj struct{}

func (emptyObj) MarshalXML(*xml.Encoder, xml.StartElement) error { return nil }

// itemDate é a data de lançamento ({} quando não sabe).
type itemDate struct {
	Year  int `xml:"year,attr,omitempty" json:"year,omitempty"`
	Month int `xml:"month,attr,omitempty" json:"month,omitempty"`
	Day   int `xml:"day,attr,omitempty" json:"day,omitempty"`
}

func (d itemDate) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if d == (itemDate{}) {
		return nil
	}
	type plain itemDate
	return e.EncodeElement(plain(d), start)
}

type resp struct {
	XMLName       xml.Name `xml:"http://subsonic.org/restapi subsonic-response" json:"-"`
	Status        string   `xml:"status,attr" json:"status"`
	Version       string   `xml:"version,attr" json:"version"`
	Type          string   `xml:"type,attr" json:"type"`
	ServerVersion string   `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool     `xml:"openSubsonic,attr,omitempty" json:"openSubsonic,omitempty"`

	Error         *errorBody     `xml:"error,omitempty" json:"error,omitempty"`
	License       *license       `xml:"license,omitempty" json:"license,omitempty"`
	MusicFolders  *musicFolders  `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Indexes       *indexes       `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Directory     *directory     `xml:"directory,omitempty" json:"directory,omitempty"`
	User          *user          `xml:"user,omitempty" json:"user,omitempty"`
	Users         *users         `xml:"users,omitempty" json:"users,omitempty"`
	AlbumList     *albumList     `xml:"albumList,omitempty" json:"albumList,omitempty"`
	AlbumList2    *albumList2    `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	Playlists     *playlists     `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist      *playlistFull  `xml:"playlist,omitempty" json:"playlist,omitempty"`
	SearchResult2 *searchResult2 `xml:"searchResult2,omitempty" json:"searchResult2,omitempty"`
	SearchResult3 *searchResult3 `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	Starred       *searchResult2 `xml:"starred,omitempty" json:"starred,omitempty"`
	Starred2      *searchResult3 `xml:"starred2,omitempty" json:"starred2,omitempty"`
	NowPlaying    *nowPlaying    `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	Song          *child         `xml:"song,omitempty" json:"song,omitempty"`
	RandomSongs   *songs         `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	SongsByGenre  *songs         `xml:"songsByGenre,omitempty" json:"songsByGenre,omitempty"`
	Genres        *genres        `xml:"genres,omitempty" json:"genres,omitempty"`
	Artists       *artistsID3    `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist        *artistFull    `xml:"artist,omitempty" json:"artist,omitempty"`
	Album         *albumFull     `xml:"album,omitempty" json:"album,omitempty"`
	AlbumInfo     *albumInfo     `xml:"albumInfo,omitempty" json:"albumInfo,omitempty"`
	ArtistInfo    *artistInfo    `xml:"artistInfo,omitempty" json:"artistInfo,omitempty"`
	ArtistInfo2   *artistInfo2   `xml:"artistInfo2,omitempty" json:"artistInfo2,omitempty"`
	SimilarSongs  *songs         `xml:"similarSongs,omitempty" json:"similarSongs,omitempty"`
	SimilarSongs2 *songs         `xml:"similarSongs2,omitempty" json:"similarSongs2,omitempty"`
	TopSongs      *songs         `xml:"topSongs,omitempty" json:"topSongs,omitempty"`
	PlayQueue     *playQueue     `xml:"playQueue,omitempty" json:"playQueue,omitempty"`
	Shares        *shares        `xml:"shares,omitempty" json:"shares,omitempty"`
	Bookmarks     *bookmarks     `xml:"bookmarks,omitempty" json:"bookmarks,omitempty"`
	ScanStatus    *scanStatus    `xml:"scanStatus,omitempty" json:"scanStatus,omitempty"`
	Lyrics        *lyrics        `xml:"lyrics,omitempty" json:"lyrics,omitempty"`
	Podcasts      *podcasts      `xml:"podcasts,omitempty" json:"podcasts,omitempty"`
	NewestPodcast *newestPodcast `xml:"newestPodcasts,omitempty" json:"newestPodcasts,omitempty"`
	Radios        *radios        `xml:"internetRadioStations,omitempty" json:"internetRadioStations,omitempty"`

	Extensions       *[]extension      `xml:"openSubsonicExtensions,omitempty" json:"openSubsonicExtensions,omitempty"`
	LyricsList       *lyricsList       `xml:"lyricsList,omitempty" json:"lyricsList,omitempty"`
	PlayQueueByIndex *playQueueByIndex `xml:"playQueueByIndex,omitempty" json:"playQueueByIndex,omitempty"`
	TokenInfo        *tokenInfo        `xml:"tokenInfo,omitempty" json:"tokenInfo,omitempty"`
}

type errorBody struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type license struct {
	Valid bool `xml:"valid,attr" json:"valid"`
}

type musicFolder struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type musicFolders struct {
	Folders []musicFolder `xml:"musicFolder" json:"musicFolder,omitempty"`
}

// artist é o artista "de pasta" (getIndexes, search2, getStarred).
type artist struct {
	ID         string     `xml:"id,attr" json:"id"`
	Name       string     `xml:"name,attr" json:"name"`
	Starred    *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating int        `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	CoverArt   string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
}

type index struct {
	Name    string   `xml:"name,attr" json:"name"`
	Artists []artist `xml:"artist" json:"artist"`
}

type indexes struct {
	Index           []index `xml:"index" json:"index,omitempty"`
	LastModified    int64   `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string  `xml:"ignoredArticles,attr" json:"ignoredArticles"`
}

type indexID3 struct {
	Name    string      `xml:"name,attr" json:"name"`
	Artists []artistID3 `xml:"artist" json:"artist"`
}

type artistsID3 struct {
	Index           []indexID3 `xml:"index" json:"index,omitempty"`
	LastModified    int64      `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string     `xml:"ignoredArticles,attr" json:"ignoredArticles"`
}

type itemGenre struct {
	Name string `xml:"name,attr" json:"name"`
}

type artistRef struct {
	ID   string `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type contributor struct {
	Role   string    `xml:"role,attr" json:"role"`
	Artist artistRef `xml:"artist" json:"artist"`
}

// child é música, álbum como pasta, ou entrada de playlist/fila.
type child struct {
	ID                    string     `xml:"id,attr" json:"id"`
	Parent                string     `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	IsDir                 bool       `xml:"isDir,attr" json:"isDir"`
	Title                 string     `xml:"title,attr" json:"title"`
	Name                  string     `xml:"name,attr,omitempty" json:"name,omitempty"`
	Album                 string     `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist                string     `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track                 int        `xml:"track,attr,omitempty" json:"track,omitempty"`
	Year                  int        `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre                 string     `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt              string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size                  int64      `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType           string     `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix                string     `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	Starred               *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	TranscodedContentType string     `xml:"transcodedContentType,attr,omitempty" json:"transcodedContentType,omitempty"`
	TranscodedSuffix      string     `xml:"transcodedSuffix,attr,omitempty" json:"transcodedSuffix,omitempty"`
	Duration              int        `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	BitRate               int        `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path                  string     `xml:"path,attr,omitempty" json:"path,omitempty"`
	PlayCount             int        `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	DiscNumber            int        `xml:"discNumber,attr,omitempty" json:"discNumber,omitempty"`
	Created               *time.Time `xml:"created,attr,omitempty" json:"created,omitempty"`
	AlbumID               string     `xml:"albumId,attr,omitempty" json:"albumId,omitempty"`
	ArtistID              string     `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	Type                  string     `xml:"type,attr,omitempty" json:"type,omitempty"`
	UserRating            int        `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	SongCount             int        `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	IsVideo               bool       `xml:"isVideo,attr,omitempty" json:"isVideo,omitempty"`
	BookmarkPosition      int64      `xml:"bookmarkPosition,attr,omitempty" json:"bookmarkPosition,omitempty"`

	// OpenSubsonic
	Played             *time.Time         `xml:"played,attr,omitempty" json:"played,omitempty"`
	BPM                int                `xml:"bpm,attr,omitempty" json:"bpm"`
	Comment            string             `xml:"comment,attr,omitempty" json:"comment"`
	SortName           string             `xml:"sortName,attr,omitempty" json:"sortName"`
	MediaType          string             `xml:"mediaType,attr,omitempty" json:"mediaType"`
	MusicBrainzID      string             `xml:"musicBrainzId,attr,omitempty" json:"musicBrainzId"`
	Isrc               Array[string]      `xml:"isrc,omitempty" json:"isrc"`
	Genres             Array[itemGenre]   `xml:"genres,omitempty" json:"genres"`
	ReplayGain         emptyObj           `xml:"replayGain,omitempty" json:"replayGain"`
	ChannelCount       int                `xml:"channelCount,attr,omitempty" json:"channelCount"`
	SamplingRate       int                `xml:"samplingRate,attr,omitempty" json:"samplingRate"`
	BitDepth           int                `xml:"bitDepth,attr,omitempty" json:"bitDepth"`
	Moods              Array[string]      `xml:"moods,omitempty" json:"moods"`
	Artists            Array[artistRef]   `xml:"artists,omitempty" json:"artists"`
	DisplayArtist      string             `xml:"displayArtist,attr,omitempty" json:"displayArtist"`
	AlbumArtists       Array[artistRef]   `xml:"albumArtists,omitempty" json:"albumArtists"`
	DisplayAlbumArtist string             `xml:"displayAlbumArtist,attr,omitempty" json:"displayAlbumArtist"`
	Contributors       Array[contributor] `xml:"contributors,omitempty" json:"contributors"`
	DisplayComposer    string             `xml:"displayComposer,attr,omitempty" json:"displayComposer"`
	ExplicitStatus     string             `xml:"explicitStatus,attr,omitempty" json:"explicitStatus"`
}

type songs struct {
	Songs []child `xml:"song" json:"song,omitempty"`
}

type directory struct {
	Child      []child    `xml:"child" json:"child,omitempty"`
	ID         string     `xml:"id,attr" json:"id"`
	Name       string     `xml:"name,attr" json:"name"`
	Parent     string     `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	Starred    *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	PlayCount  int        `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Played     *time.Time `xml:"played,attr,omitempty" json:"played,omitempty"`
	UserRating int        `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	CoverArt   string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount  int        `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	AlbumCount int        `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
}

type artistID3 struct {
	ID            string        `xml:"id,attr" json:"id"`
	Name          string        `xml:"name,attr" json:"name"`
	CoverArt      string        `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount    int           `xml:"albumCount,attr" json:"albumCount"`
	Starred       *time.Time    `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating    int           `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	MusicBrainzID string        `xml:"musicBrainzId,attr,omitempty" json:"musicBrainzId"`
	SortName      string        `xml:"sortName,attr,omitempty" json:"sortName"`
	Roles         Array[string] `xml:"roles,omitempty" json:"roles"`
}

type recordLabel struct {
	Name string `xml:"name,attr" json:"name"`
}

type discTitle struct {
	Disc  int    `xml:"disc,attr" json:"disc"`
	Title string `xml:"title,attr" json:"title"`
}

type albumID3 struct {
	ID        string     `xml:"id,attr" json:"id"`
	Name      string     `xml:"name,attr" json:"name"`
	Artist    string     `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID  string     `xml:"artistId,attr,omitempty" json:"artistId,omitempty"`
	CoverArt  string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount int        `xml:"songCount,attr" json:"songCount"`
	Duration  int        `xml:"duration,attr" json:"duration"`
	PlayCount int        `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Created   time.Time  `xml:"created,attr" json:"created"`
	Starred   *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	Year      int        `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre     string     `xml:"genre,attr,omitempty" json:"genre,omitempty"`

	// OpenSubsonic
	Played              *time.Time         `xml:"played,attr,omitempty" json:"played,omitempty"`
	UserRating          int                `xml:"userRating,attr,omitempty" json:"userRating"`
	Genres              Array[itemGenre]   `xml:"genres,omitempty" json:"genres"`
	MusicBrainzID       string             `xml:"musicBrainzId,attr,omitempty" json:"musicBrainzId"`
	IsCompilation       bool               `xml:"isCompilation,attr,omitempty" json:"isCompilation"`
	SortName            string             `xml:"sortName,attr,omitempty" json:"sortName"`
	DiscTitles          Array[discTitle]   `xml:"discTitles,omitempty" json:"discTitles"`
	OriginalReleaseDate itemDate           `xml:"originalReleaseDate,omitempty" json:"originalReleaseDate"`
	ReleaseDate         itemDate           `xml:"releaseDate,omitempty" json:"releaseDate"`
	ReleaseTypes        Array[string]      `xml:"releaseTypes,omitempty" json:"releaseTypes"`
	RecordLabels        Array[recordLabel] `xml:"recordLabels,omitempty" json:"recordLabels"`
	Moods               Array[string]      `xml:"moods,omitempty" json:"moods"`
	Artists             Array[artistRef]   `xml:"artists,omitempty" json:"artists"`
	DisplayArtist       string             `xml:"displayArtist,attr,omitempty" json:"displayArtist"`
	ExplicitStatus      string             `xml:"explicitStatus,attr,omitempty" json:"explicitStatus"`
	Version             string             `xml:"version,attr,omitempty" json:"version"`
}

type artistFull struct {
	artistID3
	Album []albumID3 `xml:"album" json:"album,omitempty"`
}

type albumFull struct {
	albumID3
	Song []child `xml:"song" json:"song,omitempty"`
}

type albumList struct {
	Album []child `xml:"album" json:"album,omitempty"`
}

type albumList2 struct {
	Album []albumID3 `xml:"album" json:"album,omitempty"`
}

type playlist struct {
	ID        string    `xml:"id,attr" json:"id"`
	Name      string    `xml:"name,attr" json:"name"`
	Comment   string    `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	SongCount int       `xml:"songCount,attr" json:"songCount"`
	Duration  int       `xml:"duration,attr" json:"duration"`
	Public    bool      `xml:"public,attr" json:"public,omitempty"`
	Owner     string    `xml:"owner,attr,omitempty" json:"owner,omitempty"`
	Created   time.Time `xml:"created,attr" json:"created"`
	Changed   time.Time `xml:"changed,attr" json:"changed"`
	CoverArt  string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Readonly  bool      `xml:"readonly,attr,omitempty" json:"readonly"`
}

type playlists struct {
	Playlist []playlist `xml:"playlist" json:"playlist,omitempty"`
}

type playlistFull struct {
	playlist
	Entry []child `xml:"entry" json:"entry,omitempty"`
}

type searchResult2 struct {
	Artist []artist `xml:"artist" json:"artist,omitempty"`
	Album  []child  `xml:"album" json:"album,omitempty"`
	Song   []child  `xml:"song" json:"song,omitempty"`
}

type searchResult3 struct {
	Artist []artistID3 `xml:"artist" json:"artist,omitempty"`
	Album  []albumID3  `xml:"album" json:"album,omitempty"`
	Song   []child     `xml:"song" json:"song,omitempty"`
}

type nowPlayingEntry struct {
	child
	Username     string  `xml:"username,attr" json:"username"`
	MinutesAgo   int     `xml:"minutesAgo,attr" json:"minutesAgo"`
	PlayerID     int     `xml:"playerId,attr" json:"playerId"`
	PlayerName   string  `xml:"playerName,attr" json:"playerName,omitempty"`
	State        string  `xml:"state,attr" json:"state"`
	PositionMs   int64   `xml:"positionMs,attr" json:"positionMs"`
	PlaybackRate float64 `xml:"playbackRate,attr" json:"playbackRate"`
}

type nowPlaying struct {
	Entry []nowPlayingEntry `xml:"entry" json:"entry,omitempty"`
}

type user struct {
	Username            string `xml:"username,attr" json:"username"`
	ScrobblingEnabled   bool   `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	AdminRole           bool   `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole        bool   `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool   `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool   `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool   `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool   `xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool   `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool   `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool   `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool   `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool   `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool   `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	Folder              []int  `xml:"folder,omitempty" json:"folder,omitempty"`
}

type users struct {
	User []user `xml:"user" json:"user"`
}

type genre struct {
	Name       string `xml:",chardata" json:"value,omitempty"`
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
}

type genres struct {
	Genre []genre `xml:"genre,omitempty" json:"genre,omitempty"`
}

type albumInfo struct {
	Notes          string `xml:"notes,omitempty" json:"notes,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageURL string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageURL  string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type artistInfoBase struct {
	Biography      string `xml:"biography,omitempty" json:"biography,omitempty"`
	MusicBrainzID  string `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  string `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageURL string `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageURL  string `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
}

type artistInfo struct {
	artistInfoBase
	SimilarArtist []artist `xml:"similarArtist,omitempty" json:"similarArtist,omitempty"`
}

type artistInfo2 struct {
	artistInfoBase
	SimilarArtist []artistID3 `xml:"similarArtist,omitempty" json:"similarArtist,omitempty"`
}

type playQueue struct {
	Entry     []child   `xml:"entry,omitempty" json:"entry,omitempty"`
	Current   string    `xml:"current,attr,omitempty" json:"current,omitempty"`
	Position  int64     `xml:"position,attr,omitempty" json:"position,omitempty"`
	Username  string    `xml:"username,attr" json:"username"`
	Changed   time.Time `xml:"changed,attr" json:"changed"`
	ChangedBy string    `xml:"changedBy,attr" json:"changedBy"`
}

type playQueueByIndex struct {
	Entry        []child   `xml:"entry,omitempty" json:"entry,omitempty"`
	CurrentIndex *int      `xml:"currentIndex,attr,omitempty" json:"currentIndex,omitempty"`
	Position     int64     `xml:"position,attr,omitempty" json:"position,omitempty"`
	Username     string    `xml:"username,attr" json:"username"`
	Changed      time.Time `xml:"changed,attr" json:"changed"`
	ChangedBy    string    `xml:"changedBy,attr" json:"changedBy"`
}

type bookmark struct {
	Entry    child     `xml:"entry,omitempty" json:"entry"`
	Position int64     `xml:"position,attr,omitempty" json:"position,omitempty"`
	Username string    `xml:"username,attr" json:"username"`
	Comment  string    `xml:"comment,attr" json:"comment"`
	Created  time.Time `xml:"created,attr" json:"created"`
	Changed  time.Time `xml:"changed,attr" json:"changed"`
}

type bookmarks struct {
	Bookmark []bookmark `xml:"bookmark,omitempty" json:"bookmark,omitempty"`
}

type shares struct {
	Share []struct{} `xml:"share,omitempty" json:"share,omitempty"`
}

type podcasts struct {
	Channel []struct{} `xml:"channel,omitempty" json:"channel,omitempty"`
}

type newestPodcast struct {
	Episode []struct{} `xml:"episode,omitempty" json:"episode,omitempty"`
}

type radios struct {
	Radios []struct{} `xml:"internetRadioStation" json:"internetRadioStation,omitempty"`
}

type scanStatus struct {
	Scanning    bool       `xml:"scanning,attr" json:"scanning"`
	Count       int64      `xml:"count,attr" json:"count"`
	FolderCount int64      `xml:"folderCount,attr" json:"folderCount"`
	LastScan    *time.Time `xml:"lastScan,attr,omitempty" json:"lastScan,omitempty"`
	Error       string     `xml:"error,attr,omitempty" json:"error,omitempty"`
}

type lyrics struct {
	Artist string `xml:"artist,omitempty,attr" json:"artist,omitempty"`
	Title  string `xml:"title,omitempty,attr" json:"title,omitempty"`
	Value  string `xml:",chardata" json:"value"`
}

type line struct {
	Start *int64 `xml:"start,attr,omitempty" json:"start,omitempty"`
	Value string `xml:",chardata" json:"value"`
}

type structuredLyric struct {
	DisplayArtist string `xml:"displayArtist,attr,omitempty" json:"displayArtist,omitempty"`
	DisplayTitle  string `xml:"displayTitle,attr,omitempty" json:"displayTitle,omitempty"`
	Lang          string `xml:"lang,attr" json:"lang"`
	Line          []line `xml:"line" json:"line"`
	Offset        *int64 `xml:"offset,attr,omitempty" json:"offset,omitempty"`
	Synced        bool   `xml:"synced,attr" json:"synced"`
}

type lyricsList struct {
	StructuredLyrics []structuredLyric `xml:"structuredLyrics,omitempty" json:"structuredLyrics,omitempty"`
}

type extension struct {
	Name     string `xml:"name,attr" json:"name"`
	Versions []int  `xml:"versions" json:"versions"`
}

type tokenInfo struct {
	Username string `xml:"username,attr" json:"username"`
}

// newResp é a resposta ok vazia.
func (a *API) newResp() *resp {
	return &resp{Status: "ok", Version: apiVersion, Type: "arkflix", ServerVersion: a.d.Version, OpenSubsonic: true}
}

var jsIdent = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$.]*$`)

// send escreve a resposta no formato pedido (f=xml|json|jsonp).
func (a *API) send(w http.ResponseWriter, r *http.Request, res *resp) {
	var body []byte
	var err error
	switch r.Form.Get("f") {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		body, err = json.Marshal(map[string]*resp{"subsonic-response": res})
	case "jsonp":
		cb := r.Form.Get("callback")
		if !jsIdent.MatchString(cb) {
			bad := a.newResp()
			bad.Status, bad.Error = "failed", &errorBody{errGeneric, "invalid callback parameter"}
			w.Header().Set("Content-Type", "application/json")
			body, err = json.Marshal(map[string]*resp{"subsonic-response": bad})
			break
		}
		w.Header().Set("Content-Type", "application/javascript")
		body, err = json.Marshal(map[string]*resp{"subsonic-response": res})
		body = fmt.Appendf(nil, "%s(%s)", cb, body)
	default:
		w.Header().Set("Content-Type", "application/xml")
		body, err = xml.Marshal(res)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(body)
}

// sendErr manda <error> com status ok do HTTP (como o Navidrome).
func (a *API) sendErr(w http.ResponseWriter, r *http.Request, err error) {
	ae := toAPIErr(err)
	res := a.newResp()
	res.Status, res.Error = "failed", &errorBody{ae.code, ae.msg}
	a.send(w, r, res)
}

// ms corta em milissegundos (cliente velho engasga com nanossegundo).
func ms(t time.Time) time.Time { return t.UTC().Truncate(time.Millisecond) }

// tp devolve ponteiro pra hora, nil se zero.
func tp(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := ms(t)
	return &v
}
