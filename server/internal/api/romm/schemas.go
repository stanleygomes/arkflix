package romm

import (
	"regexp"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
)

// ts é uma data no formato do pydantic (datetime.isoformat() em UTC):
// "2024-01-02T03:04:05.123456+00:00", sem fração quando é zero. Zero = null.
type ts time.Time

func (t ts) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + isoformat(tt) + `"`), nil
}

func isoformat(t time.Time) string {
	t = t.UTC()
	layout := "2006-01-02T15:04:05"
	if t.Nanosecond()/1000 != 0 {
		layout += ".000000"
	}
	return t.Format(layout) + "+00:00"
}

// Nomes de arquivo como o RomM corta (models/base.py).
var (
	extRe = regexp.MustCompile(`\.(([a-z]+\.)*\w+)$`)
	tagRe = regexp.MustCompile(`(?:\s*(?:\([^)]*\)|\[[^\]]*\]))+\s*$`)
)

func nameNoExt(name string) string { return strings.TrimSpace(extRe.ReplaceAllString(name, "")) }

func nameNoTags(name string) string {
	return strings.TrimSpace(tagRe.ReplaceAllString(nameNoExt(name), ""))
}

func nameExt(name string) string {
	if m := extRe.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return ""
}

// orNil devolve nil pra string vazia (campo str | None).
func orNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// userSchema é o UserSchema.
type userSchema struct {
	ID                 int64    `json:"id"`
	Username           string   `json:"username"`
	Email              *string  `json:"email"`
	Enabled            bool     `json:"enabled"`
	Role               string   `json:"role"`
	PermissionGroupID  *int64   `json:"permission_group_id"`
	OAuthScopes        []string `json:"oauth_scopes"`
	HasPassword        bool     `json:"has_password"`
	EmailManagedByOIDC bool     `json:"email_managed_by_oidc"`
	AvatarPath         string   `json:"avatar_path"`
	LastLogin          ts       `json:"last_login"`
	LastActive         ts       `json:"last_active"`
	RAUsername         *string  `json:"ra_username"`
	RAProgression      any      `json:"ra_progression"`
	UISettings         any      `json:"ui_settings"`
	CurrentDeviceID    *string  `json:"current_device_id"`
	CreatedAt          ts       `json:"created_at"`
	UpdatedAt          ts       `json:"updated_at"`
}

func newUserSchema(u *auth.User) userSchema {
	role := "user"
	if u.Admin {
		role = "admin"
	}
	return userSchema{ID: u.RID, Username: u.Name, Enabled: true, Role: role, OAuthScopes: userScopes(u),
		HasPassword: true, LastLogin: ts(u.LastLogin), LastActive: ts(u.LastLogin),
		CreatedAt: ts(u.CreatedAt), UpdatedAt: ts(u.CreatedAt)}
}

// clientTokenSchema é o ClientTokenSchema (+ raw_token na criação, +
// usuário na lista do admin).
type clientTokenSchema struct {
	ID             int64    `json:"id"`
	Name           string   `json:"name"`
	Scopes         []string `json:"scopes"`
	ExpiresAt      ts       `json:"expires_at"`
	LastUsedAt     ts       `json:"last_used_at"`
	CreatedAt      ts       `json:"created_at"`
	UserID         int64    `json:"user_id"`
	DeviceID       *string  `json:"device_id"`
	RawToken       string   `json:"raw_token,omitempty"`
	Username       string   `json:"username,omitempty"`
	UserAvatarPath *string  `json:"user_avatar_path,omitempty"`
	UserUpdatedAt  *ts      `json:"user_updated_at,omitempty"`
}

func newClientTokenSchema(ct clientToken, userRID int64) clientTokenSchema {
	s := clientTokenSchema{ID: ct.ID, Name: ct.Name, Scopes: ct.Scopes, ExpiresAt: ts(ct.Expires), LastUsedAt: ts(ct.LastUsed),
		CreatedAt: ts(ct.Created), UserID: userRID}
	if s.Scopes == nil {
		s.Scopes = []string{}
	}
	if ct.DeviceID != "" {
		s.DeviceID = &ct.DeviceID
	}
	return s
}

// firmwareSchema é o FirmwareSchema.
type firmwareSchema struct {
	ID             int64  `json:"id"`
	PlatformID     int64  `json:"platform_id"`
	FileName       string `json:"file_name"`
	FileNameNoTags string `json:"file_name_no_tags"`
	FileNameNoExt  string `json:"file_name_no_ext"`
	FileExtension  string `json:"file_extension"`
	FilePath       string `json:"file_path"`
	FileSizeBytes  int64  `json:"file_size_bytes"`
	FullPath       string `json:"full_path"`
	IsVerified     bool   `json:"is_verified"`
	CRCHash        string `json:"crc_hash"`
	MD5Hash        string `json:"md5_hash"`
	SHA1Hash       string `json:"sha1_hash"`
	MissingFromFS  bool   `json:"missing_from_fs"`
	CreatedAt      ts     `json:"created_at"`
	UpdatedAt      ts     `json:"updated_at"`
}

// platformSchema é o PlatformSchema.
type platformSchema struct {
	ID               int64            `json:"id"`
	Slug             string           `json:"slug"`
	FsSlug           string           `json:"fs_slug"`
	RomCount         int64            `json:"rom_count"`
	Name             string           `json:"name"`
	IGDBSlug         *string          `json:"igdb_slug"`
	MobySlug         *string          `json:"moby_slug"`
	HLTBSlug         *string          `json:"hltb_slug"`
	LibretroSlug     *string          `json:"libretro_slug"`
	CustomName       *string          `json:"custom_name"`
	Description      *string          `json:"description"`
	IGDBID           *int64           `json:"igdb_id"`
	SGDBID           *int64           `json:"sgdb_id"`
	MobyID           *int64           `json:"moby_id"`
	LaunchboxID      *int64           `json:"launchbox_id"`
	SSID             *int64           `json:"ss_id"`
	RAID             *int64           `json:"ra_id"`
	HasheousID       *int64           `json:"hasheous_id"`
	TGDBID           *int64           `json:"tgdb_id"`
	FlashpointID     *int64           `json:"flashpoint_id"`
	Category         *string          `json:"category"`
	Generation       *int64           `json:"generation"`
	FirstReleaseYear *int64           `json:"first_release_year"`
	FamilyName       *string          `json:"family_name"`
	FamilySlug       *string          `json:"family_slug"`
	URL              *string          `json:"url"`
	URLLogo          *string          `json:"url_logo"`
	LogoPath         string           `json:"logo_path"`
	Firmware         []firmwareSchema `json:"firmware"`
	CreatedAt        ts               `json:"created_at"`
	UpdatedAt        ts               `json:"updated_at"`
	FsSizeBytes      int64            `json:"fs_size_bytes"`
	IsUnidentified   bool             `json:"is_unidentified"`
	IsIdentified     bool             `json:"is_identified"`
	MissingFromFS    bool             `json:"missing_from_fs"`
	AspectRatio      string           `json:"aspect_ratio"`
	DisplayName      string           `json:"display_name"`
	Abbreviation     string           `json:"abbreviation"`
	AlternativeNames []string         `json:"alternative_names"`
	FirmwareCount    int              `json:"firmware_count"`
}

// romFileSchema é o RomFileSchema.
type romFileSchema struct {
	ID            int64   `json:"id"`
	RomID         int64   `json:"rom_id"`
	FileName      string  `json:"file_name"`
	FilePath      string  `json:"file_path"`
	FileSizeBytes int64   `json:"file_size_bytes"`
	FullPath      string  `json:"full_path"`
	IsTopLevel    bool    `json:"is_top_level"`
	CreatedAt     ts      `json:"created_at"`
	UpdatedAt     ts      `json:"updated_at"`
	LastModified  ts      `json:"last_modified"`
	CRCHash       any     `json:"crc_hash"`
	MD5Hash       any     `json:"md5_hash"`
	SHA1Hash      any     `json:"sha1_hash"`
	RAHash        any     `json:"ra_hash"`
	CHDSHA1Hash   any     `json:"chd_sha1_hash"`
	ArchiveMember any     `json:"archive_members"`
	Category      *string `json:"category"`
	TrackMeta     any     `json:"track_meta"`
	DocMeta       any     `json:"doc_meta"`
}

// romUserSchema é o RomUserSchema.
type romUserSchema struct {
	ID            int64    `json:"id"`
	UserID        int64    `json:"user_id"`
	RomID         int64    `json:"rom_id"`
	CreatedAt     ts       `json:"created_at"`
	UpdatedAt     ts       `json:"updated_at"`
	LastPlayed    ts       `json:"last_played"`
	IsMainSibling bool     `json:"is_main_sibling"`
	Backlogged    bool     `json:"backlogged"`
	NowPlaying    bool     `json:"now_playing"`
	Hidden        bool     `json:"hidden"`
	Rating        int      `json:"rating"`
	Difficulty    int      `json:"difficulty"`
	Completion    int      `json:"completion"`
	Status        *string  `json:"status"`
	PinnedMedia   []string `json:"pinned_media"`
}

// romMetadataSchema é o RomMetadataSchema (metadatum).
type romMetadataSchema struct {
	RomID            int64    `json:"rom_id"`
	Genres           []string `json:"genres"`
	Franchises       []string `json:"franchises"`
	Collections      []string `json:"collections"`
	Companies        []string `json:"companies"`
	Publishers       []string `json:"publishers"`
	Developers       []string `json:"developers"`
	GameModes        []string `json:"game_modes"`
	AgeRatings       []string `json:"age_ratings"`
	PlayerCount      string   `json:"player_count"`
	FirstReleaseDate *int64   `json:"first_release_date"`
	AverageRating    *float64 `json:"average_rating"`
}

// romSchema é o RomSchema: a base do SimpleRomSchema e do DetailedRomSchema.
type romSchema struct {
	ID                 int64             `json:"id"`
	IGDBID             any               `json:"igdb_id"`
	SGDBID             any               `json:"sgdb_id"`
	MobyID             any               `json:"moby_id"`
	SSID               any               `json:"ss_id"`
	RAID               any               `json:"ra_id"`
	LaunchboxID        any               `json:"launchbox_id"`
	HasheousID         any               `json:"hasheous_id"`
	TGDBID             any               `json:"tgdb_id"`
	FlashpointID       any               `json:"flashpoint_id"`
	HLTBID             any               `json:"hltb_id"`
	DemozooID          any               `json:"demozoo_id"`
	PouetID            any               `json:"pouet_id"`
	CSDBID             any               `json:"csdb_id"`
	SteamID            any               `json:"steam_id"`
	GamelistID         any               `json:"gamelist_id"`
	LibretroID         any               `json:"libretro_id"`
	PlatformID         int64             `json:"platform_id"`
	PlatformSlug       string            `json:"platform_slug"`
	PlatformFsSlug     string            `json:"platform_fs_slug"`
	PlatformCustomName any               `json:"platform_custom_name"`
	PlatformName       string            `json:"platform_name"`
	PlatformDisplay    string            `json:"platform_display_name"`
	FsName             string            `json:"fs_name"`
	FsNameNoTags       string            `json:"fs_name_no_tags"`
	FsNameNoExt        string            `json:"fs_name_no_ext"`
	FsExtension        string            `json:"fs_extension"`
	FsPath             string            `json:"fs_path"`
	FsSizeBytes        int64             `json:"fs_size_bytes"`
	Name               string            `json:"name"`
	NameSortKey        string            `json:"name_sort_key"`
	Slug               any               `json:"slug"`
	Summary            any               `json:"summary"`
	AlternativeNames   []string          `json:"alternative_names"`
	YoutubeVideoID     any               `json:"youtube_video_id"`
	Metadatum          romMetadataSchema `json:"metadatum"`
	IGDBMetadata       any               `json:"igdb_metadata"`
	MobyMetadata       any               `json:"moby_metadata"`
	SSMetadata         any               `json:"ss_metadata"`
	LaunchboxMetadata  any               `json:"launchbox_metadata"`
	HasheousMetadata   any               `json:"hasheous_metadata"`
	FlashpointMetadata any               `json:"flashpoint_metadata"`
	HLTBMetadata       any               `json:"hltb_metadata"`
	DemozooMetadata    any               `json:"demozoo_metadata"`
	PouetMetadata      any               `json:"pouet_metadata"`
	CSDBMetadata       any               `json:"csdb_metadata"`
	SteamMetadata      any               `json:"steam_metadata"`
	GamelistMetadata   any               `json:"gamelist_metadata"`
	ManualMetadata     any               `json:"manual_metadata"`
	PathCoverSmall     string            `json:"path_cover_small"`
	PathCoverLarge     string            `json:"path_cover_large"`
	URLCover           any               `json:"url_cover"`
	HasManual          bool              `json:"has_manual"`
	HasSoundtrack      bool              `json:"has_soundtrack"`
	PathManual         any               `json:"path_manual"`
	URLManual          any               `json:"url_manual"`
	PathVideo          any               `json:"path_video"`
	IsIdentifying      bool              `json:"is_identifying"`
	IsUnidentified     bool              `json:"is_unidentified"`
	IsIdentified       bool              `json:"is_identified"`
	Revision           any               `json:"revision"`
	Version            any               `json:"version"`
	Regions            []string          `json:"regions"`
	Languages          []string          `json:"languages"`
	Tags               []string          `json:"tags"`
	CRCHash            any               `json:"crc_hash"`
	MD5Hash            any               `json:"md5_hash"`
	SHA1Hash           any               `json:"sha1_hash"`
	RAHash             any               `json:"ra_hash"`
	TitleID            any               `json:"title_id"`
	SaveTarget         any               `json:"save_target"`
	SaveTargetLayout   any               `json:"save_target_layout"`
	HasSimpleSingle    bool              `json:"has_simple_single_file"`
	HasNestedSingle    bool              `json:"has_nested_single_file"`
	HasMultipleFiles   bool              `json:"has_multiple_files"`
	IsEasyRPGGame      bool              `json:"is_easyrpg_game"`
	FullPath           string            `json:"full_path"`
	CreatedAt          ts                `json:"created_at"`
	UpdatedAt          ts                `json:"updated_at"`
	MissingFromFS      bool              `json:"missing_from_fs"`
	IsPhysical         bool              `json:"is_physical"`
	HasFileOnDisk      bool              `json:"has_file_on_disk"`
	UPC                any               `json:"upc"`
	HasNotes           bool              `json:"has_notes"`
	RomUser            romUserSchema     `json:"rom_user"`
	MergedScreenshots  []string          `json:"merged_screenshots"`
	MergedRAMetadata   any               `json:"merged_ra_metadata"`
	Files              []romFileSchema   `json:"files"`
	SiblingRoms        []any             `json:"sibling_roms"`
	// campos de versões antigas que clientes velhos ainda leem
	Multi bool `json:"multi"`
}

type simpleRomSchema struct {
	romSchema
	ScreenshotPath any `json:"screenshot_path"`
}

type detailedRomSchema struct {
	romSchema
	UserSaves          []assetJSON `json:"user_saves"`
	UserStates         []assetJSON `json:"user_states"`
	AllUserSaves       []assetJSON `json:"all_user_saves"`
	AllUserStates      []assetJSON `json:"all_user_states"`
	UserChannels       []any       `json:"user_channels"`
	ChannelFileID      *int64      `json:"channel_file_id"`
	SnapshotSaveIDs    []int64     `json:"snapshot_save_ids"`
	UserScreenshots    []assetJSON `json:"user_screenshots"`
	AllUserScreenshots []assetJSON `json:"all_user_screenshots"`
	UserCollections    []any       `json:"user_collections"`
	AllUserNotes       []any       `json:"all_user_notes"`
	DownloadFormats    []string    `json:"download_formats"`
	// nomes antigos (RomM 3.x) que alguns clientes ainda leem
	UserNotes []any `json:"user_notes"`
}

// assetJSON é um SaveSchema, StateSchema ou ScreenshotSchema (cada tipo
// tem campos próprios; ver assetSchema em saves.go).
type assetJSON = map[string]any
