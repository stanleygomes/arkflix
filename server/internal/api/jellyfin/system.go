package jellyfin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

func init() {
	store.Register(store.Migration{Name: "jellyfin/001-displayprefs", SQL: `
CREATE TABLE jellyfin_displayprefs (
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  id      TEXT NOT NULL,
  client  TEXT NOT NULL DEFAULT '',
  data    TEXT NOT NULL,
  PRIMARY KEY (user_id, id, client)
);`})
}

func (c *rc) localAddress() string {
	scheme := "http"
	if c.r.TLS != nil || c.r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.r.Host
}

func (s *Server) publicInfo(c *rc) map[string]any {
	return map[string]any{
		"LocalAddress": c.localAddress(), "ServerName": s.d.Config.Name, "Version": apiVersion,
		"ProductName": productName, "OperatingSystem": "Linux", "Id": s.serverID, "StartupWizardCompleted": true,
	}
}

// GET /System/Info/Public (sem login)
func (s *Server) systemInfoPublic(c *rc) { c.json(s.publicInfo(c)) }

// GET /System/Info
func (s *Server) systemInfo(c *rc) {
	info := s.publicInfo(c)
	port := 80
	if _, p, err := net.SplitHostPort(c.r.Host); err == nil {
		port, _ = strconv.Atoi(p)
	}
	arch := map[string]string{"amd64": "X64", "arm64": "Arm64", "arm": "Arm", "386": "X86"}[runtime.GOARCH]
	for k, v := range map[string]any{
		"OperatingSystemDisplayName": "Linux", "PackageName": "arkflix-server " + s.d.Version,
		"HasPendingRestart": false, "IsShuttingDown": false, "SupportsLibraryMonitor": false,
		"WebSocketPortNumber": port, "CompletedInstallations": []any{}, "CanSelfRestart": false,
		"CanLaunchWebBrowser": false, "ProgramDataPath": s.d.Config.Data, "WebPath": "", "ItemsByNamePath": "",
		"CachePath": s.d.Config.Data + "/cache", "LogPath": "", "InternalMetadataPath": "",
		"TranscodingTempPath": s.hls.dir, "CastReceiverApplications": []any{}, "HasUpdateAvailable": false,
		"EncoderLocation": "System", "SystemArchitecture": arch,
	} {
		info[k] = v
	}
	c.json(info)
}

// GET/POST /System/Ping
func (s *Server) ping(c *rc) {
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.w.Write([]byte(`"Jellyfin Server"`))
}

// GET /System/Endpoint
func (s *Server) endpoint(c *rc) {
	host, _, _ := net.SplitHostPort(c.r.RemoteAddr)
	ip := net.ParseIP(host)
	local := ip != nil && (ip.IsLoopback() || ip.IsPrivate())
	c.json(map[string]bool{"IsLocal": ip != nil && ip.IsLoopback(), "IsInNetwork": local})
}

// GET /System/Configuration: o mínimo que clientes leem.
func (s *Server) systemConfig(c *rc) {
	c.json(map[string]any{
		"ServerName": s.d.Config.Name, "UICulture": "pt-BR", "IsStartupWizardCompleted": true,
		"EnableExternalContentInSuggestions": false, "MinResumePct": 5, "MaxResumePct": 90,
		"MinResumeDurationSeconds": 0, "MinAudiobookResume": 5, "MaxAudiobookResume": 5,
		"EnableFolderView": false, "EnableGroupingIntoCollections": false, "DisplaySpecialsWithinSeasons": true,
		"PreferredMetadataLanguage": "pt", "MetadataCountryCode": "BR", "LibraryMonitorDelay": 60,
		"QuickConnectAvailable": false, "CastReceiverApplications": []any{}, "PluginRepositories": []any{},
		"ContentTypes": []any{}, "PathSubstitutions": []any{}, "CodecsUsed": []any{},
		"MetadataOptions": []any{}, "SortRemoveCharacters": []any{}, "SortRemoveWords": []any{}, "SortReplaceCharacters": []any{},
		"TrickplayOptions": map[string]any{"EnableHwAcceleration": false, "Interval": 10000, "WidthResolutions": []int{320}},
	})
}

// --- usuários ---

func (s *Server) userDto(u *auth.User) map[string]any {
	d := map[string]any{
		"Name": u.Name, "ServerId": s.serverID, "ServerName": s.d.Config.Name, "Id": u.ID,
		"HasPassword": true, "HasConfiguredPassword": true, "HasConfiguredEasyPassword": false,
		"EnableAutoLogin": false, "LastActivityDate": jfTime(time.Now()),
		"Configuration": map[string]any{
			"PlayDefaultAudioTrack": true, "SubtitleLanguagePreference": "", "DisplayMissingEpisodes": false,
			"GroupedFolders": []string{}, "SubtitleMode": "Default", "DisplayCollectionsView": false,
			"EnableLocalPassword": false, "OrderedViews": []string{}, "LatestItemsExcludes": []string{},
			"MyMediaExcludes": []string{}, "HidePlayedInLatest": true, "RememberAudioSelections": true,
			"RememberSubtitleSelections": true, "EnableNextEpisodeAutoPlay": true, "AudioLanguagePreference": "",
		},
		"Policy": map[string]any{
			"IsAdministrator": u.Admin, "IsHidden": false, "EnableCollectionManagement": false,
			"EnableSubtitleManagement": false, "EnableLyricManagement": false, "IsDisabled": false,
			"BlockedTags": []string{}, "AllowedTags": []string{}, "EnableUserPreferenceAccess": true,
			"AccessSchedules": []any{}, "BlockUnratedItems": []string{}, "EnableRemoteControlOfOtherUsers": false,
			"EnableSharedDeviceControl": false, "EnableRemoteAccess": true, "EnableLiveTvManagement": false,
			"EnableLiveTvAccess": false, "EnableMediaPlayback": true, "EnableAudioPlaybackTranscoding": true,
			"EnableVideoPlaybackTranscoding": true, "EnablePlaybackRemuxing": true, "ForceRemoteSourceTranscoding": false,
			"EnableContentDeletion": false, "EnableContentDeletionFromFolders": []string{}, "EnableContentDownloading": true,
			"EnableSyncTranscoding": false, "EnableMediaConversion": false, "EnabledDevices": []string{},
			"EnableAllDevices": true, "EnabledChannels": []string{}, "EnableAllChannels": true, "EnabledFolders": []string{},
			"EnableAllFolders": true, "InvalidLoginAttemptCount": 0, "LoginAttemptsBeforeLockout": -1,
			"MaxActiveSessions": 0, "EnablePublicSharing": false, "BlockedMediaFolders": []string{},
			"BlockedChannels": []string{}, "RemoteClientBitrateLimit": 0,
			"AuthenticationProviderId": "Jellyfin.Server.Implementations.Users.DefaultAuthenticationProvider",
			"PasswordResetProviderId":  "Jellyfin.Server.Implementations.Users.DefaultPasswordResetProvider",
			"SyncPlayAccess":           "CreateAndJoinGroups",
		},
	}
	if !u.LastLogin.IsZero() {
		d["LastLoginDate"] = jfTime(u.LastLogin)
	}
	return d
}

// GET /Users/Me
func (s *Server) me(c *rc) { c.json(s.userDto(c.user)) }

// GET /Users/{userId}
func (s *Server) userByID(c *rc) {
	u, err := s.d.Auth.User(c.ctx(), c.uid())
	if err != nil {
		c.err(err)
		return
	}
	c.json(s.userDto(u))
}

// GET /Users: admin vê todos; os outros, só a si.
func (s *Server) users(c *rc) {
	list := []*auth.User{c.user}
	if c.user.Admin {
		all, err := s.d.Auth.Users(c.ctx())
		if err != nil {
			c.err(err)
			return
		}
		list = all
	}
	out := []map[string]any{}
	for _, u := range list {
		out = append(out, s.userDto(u))
	}
	c.json(out)
}

// --- sessões e aparelhos ---

func (s *Server) sessionDto(ss session) map[string]any {
	d := map[string]any{
		"PlayState": map[string]any{"CanSeek": ss.Playing != "", "IsPaused": ss.Paused, "IsMuted": false,
			"RepeatMode": "RepeatNone", "PlaybackOrder": "Default"},
		"AdditionalUsers": []any{},
		"Capabilities": map[string]any{"PlayableMediaTypes": []string{}, "SupportedCommands": []string{},
			"SupportsMediaControl": false, "SupportsPersistentIdentifier": true},
		"RemoteEndPoint": ss.Remote, "PlayableMediaTypes": []string{}, "Id": ss.ID, "UserId": ss.UserID,
		"UserName": ss.UserName, "Client": ss.Token.Client, "LastActivityDate": jfTime(ss.LastActive),
		"LastPlaybackCheckIn": jfTime(ss.LastCheck), "DeviceName": ss.Token.Device, "DeviceId": ss.Token.DeviceID,
		"ApplicationVersion": ss.Token.Version, "IsActive": true, "SupportsMediaControl": false,
		"SupportsRemoteControl": false, "NowPlayingQueue": []any{}, "NowPlayingQueueFullItems": []any{},
		"HasCustomDeviceName": false, "ServerId": s.serverID, "SupportedCommands": []string{},
	}
	if d["LastPlaybackCheckIn"] == "" {
		d["LastPlaybackCheckIn"] = "0001-01-01T00:00:00.0000000Z"
	}
	if ss.Playing != "" {
		d["PlayState"].(map[string]any)["PositionTicks"] = ss.Position
		d["PlayState"].(map[string]any)["PlayMethod"] = "DirectPlay"
	}
	return d
}

// GET /Sessions
func (s *Server) sessionsList(c *rc) {
	out := []map[string]any{}
	for _, ss := range s.sessions.list(c.user.ID) {
		d := s.sessionDto(ss)
		if ss.Playing != "" {
			if it, err := c.itemDtoByID(ss.Playing); err == nil {
				d["NowPlayingItem"] = it
			}
		}
		out = append(out, d)
	}
	c.json(out)
}

// GET /Devices
func (s *Server) devices(c *rc) {
	ts, err := s.d.Auth.Tokens(c.ctx(), c.user.ID, auth.KindJellyfin)
	if err != nil {
		c.err(err)
		return
	}
	out := []map[string]any{}
	for _, t := range ts {
		last := t.LastUsed
		if last.IsZero() {
			last = t.Created
		}
		out = append(out, map[string]any{
			"Name": t.Device, "CustomName": "", "Id": t.DeviceID, "LastUserName": c.user.Name, "AppName": t.Client,
			"AppVersion": t.Version, "LastUserId": c.user.ID, "DateLastActivity": jfTime(last),
			"Capabilities": map[string]any{"PlayableMediaTypes": []string{}, "SupportedCommands": []string{},
				"SupportsMediaControl": false, "SupportsPersistentIdentifier": true},
		})
	}
	c.json(map[string]any{"Items": out, "TotalRecordCount": len(out), "StartIndex": 0})
}

// --- branding, localização ---

func (s *Server) branding(c *rc) {
	c.json(map[string]any{"LoginDisclaimer": "", "CustomCss": "", "SplashscreenEnabled": false})
}

func (s *Server) brandingCSS(c *rc) {
	c.w.Header().Set("Content-Type", "text/css; charset=utf-8")
}

func (s *Server) cultures(c *rc) {
	c.json([]map[string]any{
		{"Name": "Portuguese (Brazil)", "DisplayName": "Português (Brasil)", "TwoLetterISOLanguageName": "pt",
			"ThreeLetterISOLanguageName": "por", "ThreeLetterISOLanguageNames": []string{"por"}},
		{"Name": "English", "DisplayName": "English", "TwoLetterISOLanguageName": "en",
			"ThreeLetterISOLanguageName": "eng", "ThreeLetterISOLanguageNames": []string{"eng"}},
	})
}

func (s *Server) countries(c *rc) {
	c.json([]map[string]string{
		{"Name": "BR", "DisplayName": "Brasil", "TwoLetterISORegionName": "BR", "ThreeLetterISORegionName": "BRA"},
		{"Name": "US", "DisplayName": "United States", "TwoLetterISORegionName": "US", "ThreeLetterISORegionName": "USA"},
	})
}

func (s *Server) localizationOptions(c *rc) {
	c.json([]map[string]string{{"Name": "Português (Brasil)", "Value": "pt-BR"}, {"Name": "English", "Value": "en-US"}})
}

// GET /Playback/BitrateTest?size=N: o jellyfin-web mede a banda com isso.
func (s *Server) bitrateTest(c *rc) {
	n := min(max(c.qi("Size", 102400), 0), 100<<20)
	c.w.Header().Set("Content-Type", "application/octet-stream")
	c.w.Header().Set("Content-Length", strconv.Itoa(n))
	buf := make([]byte, 32<<10)
	for n > 0 {
		k := min(n, len(buf))
		if _, err := c.w.Write(buf[:k]); err != nil {
			return
		}
		n -= k
	}
}

// --- preferências de exibição (por usuário, item e cliente) ---

type displayPrefs struct{ st *store.Store }

func (p *displayPrefs) get(ctx context.Context, user, id, client string) (map[string]any, error) {
	var raw string
	err := p.st.DB.QueryRowContext(ctx, `SELECT data FROM jellyfin_displayprefs WHERE user_id = ? AND id = ? AND client = ?`,
		user, id, client).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]any{
			"Id": id, "SortBy": "SortName", "RememberIndexing": false, "PrimaryImageHeight": 250,
			"PrimaryImageWidth": 250, "CustomPrefs": map[string]string{}, "ScrollDirection": "Horizontal",
			"ShowBackdrop": true, "RememberSorting": false, "SortOrder": "Ascending", "ShowSidebar": false, "Client": client,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var out map[string]any
	return out, json.Unmarshal([]byte(raw), &out)
}

func (s *Server) displayPrefsGet(c *rc) {
	p, err := s.prefs.get(c.ctx(), c.uid(), c.p["id"], c.qs("client"))
	if err != nil {
		c.err(err)
		return
	}
	c.json(p)
}

func (s *Server) displayPrefsSet(c *rc) {
	var body map[string]any
	if err := c.body(&body); err != nil || body == nil {
		c.fail(http.StatusBadRequest, "corpo inválido")
		return
	}
	body["Id"] = c.p["id"]
	if _, ok := body["CustomPrefs"]; !ok {
		body["CustomPrefs"] = map[string]string{}
	}
	raw, _ := json.Marshal(body)
	_, err := s.d.Store.DB.ExecContext(c.ctx(), `INSERT INTO jellyfin_displayprefs (user_id, id, client, data) VALUES (?,?,?,?)
ON CONFLICT(user_id, id, client) DO UPDATE SET data = excluded.data`, c.uid(), c.p["id"], c.qs("client"), string(raw))
	if err != nil {
		c.err(err)
		return
	}
	c.noContent()
}

func noContent(c *rc) { c.noContent() }

func jsonFalse(c *rc) { c.json(false) }

func emptyObject(c *rc) { c.json(map[string]any{}) }
