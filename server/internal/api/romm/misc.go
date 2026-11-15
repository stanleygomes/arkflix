package romm

import (
	"errors"
	"net/http"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// heartbeat é o GET /api/heartbeat (público): versão e o que está ligado.
func (a *api) heartbeat(w http.ResponseWriter, r *http.Request, _ *caller) {
	plats, err := a.platformsByID(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	slugs, bySlug := []string{}, map[string]string{}
	for _, p := range sortedPlatforms(plats) {
		slugs = append(slugs, p.it.Platform)
		bySlug[p.it.Platform] = p.it.Platform
	}
	meta := map[string]bool{}
	for _, k := range []string{"ANY_SOURCE_ENABLED", "IGDB_API_ENABLED", "SS_API_ENABLED", "SS_DEV_CREDENTIALS_SET",
		"MOBY_API_ENABLED", "STEAMGRIDDB_API_ENABLED", "RA_API_ENABLED", "LAUNCHBOX_API_ENABLED", "HASHEOUS_API_ENABLED",
		"PLAYMATCH_API_ENABLED", "TGDB_API_ENABLED", "FLASHPOINT_API_ENABLED", "HLTB_API_ENABLED", "DEMOZOO_API_ENABLED",
		"POUET_API_ENABLED", "CSDB_API_ENABLED", "STEAM_API_ENABLED", "LIBRETRO_API_ENABLED"} {
		meta[k] = false
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"SYSTEM":           map[string]any{"VERSION": Version, "GIT_BRANCH": nil, "SHOW_SETUP_WIZARD": false},
		"METADATA_SOURCES": meta,
		"FILESYSTEM":       map[string]any{"FS_PLATFORMS": slugs, "FS_PLATFORM_SLUGS": bySlug, "TITLE_ID_EXTRACTION_ENABLED": false},
		"EMULATION": map[string]any{"DISABLE_EMULATOR_JS": true, "DISABLE_RUFFLE_RS": true, "DISABLE_JSDOS": true,
			"DISABLE_PICO8": true, "DISABLE_EASYRPG": true, "EJS_CORES": map[string]any{}, "EJS_NIGHTLY_CORES": map[string]any{}},
		"FRONTEND":       map[string]any{"DISABLE_USERPASS_LOGIN": false, "DISABLE_LOGS_VIEWER": true, "YOUTUBE_BASE_URL": "https://www.youtube.com"},
		"OIDC":           map[string]any{"ENABLED": false, "AUTOLOGIN": false, "PROVIDER": "", "RP_INITIATED_LOGOUT": false},
		"NOTIFICATIONS":  map[string]any{"EMAIL_ENABLED": false, "EMAILS_RESET_LINKS": false},
		"DEVICE_INSTALL": map[string]any{"ENABLED": false, "EXCLUDED_PLATFORM_SLUGS": []string{}},
		"CONVERTO":       map[string]any{"ENABLED": false},
		"SAVE_SYNC":      map[string]any{"SNAPSHOT_RETENTION": 0, "SNAPSHOT_BRANCH_LIFETIME_DAYS": 0},
		"ROM_PATCHER":    map[string]any{"SUPPORTED_EXTENSIONS": []string{}},
		"TASKS": map[string]any{"ENABLE_SCHEDULED_RESCAN": false, "SCHEDULED_RESCAN_CRON": "",
			"ENABLE_SCHEDULED_UPDATE_SWITCH_TITLEDB": false, "SCHEDULED_UPDATE_SWITCH_TITLEDB_CRON": "",
			"ENABLE_SCHEDULED_UPDATE_LAUNCHBOX_METADATA": false, "SCHEDULED_UPDATE_LAUNCHBOX_METADATA_CRON": "",
			"ENABLE_SCHEDULED_CONVERT_IMAGES_TO_WEBP": false, "SCHEDULED_CONVERT_IMAGES_TO_WEBP_CRON": ""},
	})
}

// config é o GET /api/config (público): a config do RomM, toda vazia.
func (a *api) config(w http.ResponseWriter, r *http.Request, _ *caller) {
	empty := []string{}
	writeJSON(w, http.StatusOK, map[string]any{
		"CONFIG_FILE_MOUNTED": false, "CONFIG_FILE_WRITABLE": false, "CONFIG_FILE_PARSE_ERROR": nil,
		"EXCLUDED_PLATFORMS": empty, "EXCLUDED_SINGLE_EXT": empty, "EXCLUDED_SINGLE_FILES": empty,
		"EXCLUDED_MULTI_FILES": empty, "EXCLUDED_MULTI_PARTS_EXT": empty, "EXCLUDED_MULTI_PARTS_FILES": empty,
		"DEFAULT_EXCLUDED_PLATFORM_DIRS": empty, "DEFAULT_EXCLUDED_MULTI_FILE_DIRS": empty, "DEFAULT_EXCLUDED_FILES": empty,
		"DEFAULT_EXCLUDED_EXTENSIONS": empty, "PLATFORMS_BINDING": map[string]string{}, "PLATFORMS_VERSIONS": map[string]string{},
		"SKIP_HASH_CALCULATION": true, "EJS_DEBUG": false, "EJS_CACHE_LIMIT": nil, "EJS_DISABLE_AUTO_UNLOAD": false,
		"EJS_DISABLE_BATCH_BOOTUP": false, "EJS_ENABLE_AUTO_SAVE_SYNC": false, "EJS_NETPLAY_ENABLED": false,
		"EJS_NETPLAY_ICE_SERVERS": []any{}, "EJS_DEFAULT_CORES": map[string]string{}, "EJS_SETTINGS": map[string]any{},
		"EJS_CONTROLS": map[string]any{}, "SCAN_METADATA_PRIORITY": empty, "SCAN_ARTWORK_PRIORITY": empty,
		"SCAN_ARTWORK_PRIORITY_OVERRIDES": map[string]any{}, "SCAN_REGION_PRIORITY": empty, "SCAN_LANGUAGE_PRIORITY": empty,
		"SCAN_MEDIA": empty, "GAMELIST_AUTO_EXPORT_ON_SCAN": false, "GAMELIST_MEDIA_THUMBNAIL": "box2d",
		"GAMELIST_MEDIA_IMAGE": "screenshot", "PEGASUS_AUTO_EXPORT_ON_SCAN": false, "CONVERTO": map[string]any{},
		"CONVERTO_LIBRARY_TARGETS": map[string]any{},
	})
}

func (a *api) me(w http.ResponseWriter, r *http.Request, c *caller) {
	writeJSON(w, http.StatusOK, newUserSchema(c.user))
}

func (a *api) users(w http.ResponseWriter, r *http.Request, c *caller) {
	list, err := a.d.Auth.Users(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := []userSchema{}
	for _, u := range list {
		out = append(out, newUserSchema(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) userIDs(w http.ResponseWriter, r *http.Request, c *caller) {
	list, err := a.d.Auth.Users(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := []int64{}
	for _, u := range list {
		out = append(out, u.RID)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) user(w http.ResponseWriter, r *http.Request, c *caller) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	u, err := a.d.Auth.UserByRID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newUserSchema(u))
}

// stats é o GET /api/stats (os saves contados são de todo mundo, como no RomM).
func (a *api) stats(w http.ResponseWriter, r *http.Request, c *caller) {
	db := a.d.Store.DB
	var plats, roms, size, saves, states, shots int64
	err := db.QueryRowContext(r.Context(), `SELECT
  (SELECT COUNT(*) FROM items WHERE kind = 'platform'),
  (SELECT COUNT(*) FROM items WHERE kind = 'rom'),
  (SELECT COALESCE(SUM(size),0) FROM items WHERE kind = 'rom'),
  (SELECT COUNT(*) FROM romm_assets WHERE kind = 'saves'),
  (SELECT COUNT(*) FROM romm_assets WHERE kind = 'states'),
  (SELECT COUNT(*) FROM romm_assets WHERE kind = 'screenshots')`).Scan(&plats, &roms, &size, &saves, &states, &shots)
	if err != nil {
		internal(w, err)
		return
	}
	out := map[string]any{"PLATFORMS": plats, "ROMS": roms, "SAVES": saves, "STATES": states, "SCREENSHOTS": shots,
		"TOTAL_FILESIZE_BYTES": size}
	if b, _ := queryBool(r, "include_platform_stats"); b != nil && *b {
		out["METADATA_COVERAGE"] = map[string]any{}
		out["REGION_BREAKDOWN"] = map[string]any{}
	}
	writeJSON(w, http.StatusOK, out)
}

// Tarefas: só o scan existe (library.Trigger).
const scanTask = "scan_library"

func scanTaskInfo() map[string]any {
	return map[string]any{"name": scanTask, "type": "scan", "manual_run": true, "title": "Scan library",
		"description": "Rescan the media folders", "enabled": true, "destructive": false, "cron_string": ""}
}

func (a *api) tasks(w http.ResponseWriter, r *http.Request, c *caller) {
	writeJSON(w, http.StatusOK, map[string]any{"scheduled": []any{}, "manual": []any{scanTaskInfo()}, "watcher": []any{}})
}

func isoOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return isoformat(t)
}

// scanStatus é o TaskStatusResponse do scan a partir do library.Status.
func scanStatus(st library.Status) map[string]any {
	status := "finished"
	switch {
	case st.Running:
		status = "started"
	case st.Error != "":
		status = "failed"
	case st.Started.IsZero():
		status = "queued"
	}
	return map[string]any{"task_key": scanTask, "task_name": "Scan library", "task_id": scanTask, "task_type": "scan",
		"status": status, "created_at": isoOrNil(st.Started), "enqueued_at": isoOrNil(st.Started),
		"started_at": isoOrNil(st.Started), "ended_at": isoOrNil(st.Finished),
		"meta": map[string]any{"scan_stats": map[string]any{"total_platforms": 0, "total_roms": st.Count,
			"scanned_platforms": 0, "new_platforms": 0, "identified_platforms": 0, "scanned_roms": st.Count, "new_roms": 0,
			"identified_roms": 0, "scanned_firmware": 0, "new_firmware": 0, "updated_roms": 0, "new_files": 0}}}
}

func (a *api) tasksStatus(w http.ResponseWriter, r *http.Request, c *caller) {
	st := a.d.Library.Status()
	if st.Started.IsZero() {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, []any{scanStatus(st)})
}

func (a *api) taskByID(w http.ResponseWriter, r *http.Request, c *caller) {
	if r.PathValue("task_id") != scanTask {
		fail(w, http.StatusNotFound, "Task not found")
		return
	}
	writeJSON(w, http.StatusOK, scanStatus(a.d.Library.Status()))
}

func (a *api) startScan(w http.ResponseWriter, status int) {
	if a.d.Library.Status().Running {
		fail(w, http.StatusConflict, "Scan library is already queued or running")
		return
	}
	a.d.Library.Trigger()
	now := isoformat(time.Now())
	writeJSON(w, status, map[string]any{"task_key": scanTask, "task_name": "Scan library", "task_id": scanTask,
		"status": "queued", "created_at": now, "enqueued_at": now})
}

// runTask é o POST /api/tasks/run/{task_name}: qualquer tarefa de scan
// dispara o scan do acervo.
func (a *api) runTask(w http.ResponseWriter, r *http.Request, c *caller) {
	switch r.PathValue("task_name") {
	case scanTask, "scan", "scan_rescan", "quick_scan", "rescan":
		a.startScan(w, http.StatusOK)
	default:
		fail(w, http.StatusNotFound, "Task '"+r.PathValue("task_name")+"' not found, available tasks are "+scanTask)
	}
}

func (a *api) runScan(w http.ResponseWriter, r *http.Request, c *caller) {
	a.startScan(w, http.StatusAccepted)
}
