package admin

import (
	"context"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

type libStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Count     int    `json:"count"`
	Episodes  *int   `json:"episodes,omitempty"`
	Albums    *int   `json:"albums,omitempty"`
	Artists   *int   `json:"artists,omitempty"`
	Platforms *int   `json:"platforms,omitempty"`
	Online    bool   `json:"online"`
}

type session struct {
	User     string  `json:"user"`
	Client   string  `json:"client"`
	Device   string  `json:"device"`
	LastSeen *string `json:"lastSeen"`
}

// sessionWindow: token usado nesse intervalo conta como conectado.
const sessionWindow = 15 * time.Minute

func (a *api) count(ctx context.Context, kind string) *int {
	var n int
	a.d.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE kind = ?`, kind).Scan(&n)
	return &n
}

func online(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func (a *api) libraries(ctx context.Context) []libStatus {
	cfg := a.d.Config
	out := []libStatus{}
	add := func(id, name, path, main string, fill func(*libStatus)) {
		if path == "" {
			return
		}
		l := libStatus{ID: id, Name: name, Path: path, Count: *a.count(ctx, main), Online: online(path)}
		if fill != nil {
			fill(&l)
		}
		out = append(out, l)
	}
	add("movies", "Filmes", cfg.Movies, store.KindMovie, nil)
	add("shows", "Séries", cfg.Shows, store.KindSeries, func(l *libStatus) { l.Episodes = a.count(ctx, store.KindEpisode) })
	add("music", "Músicas", cfg.Music, store.KindTrack, func(l *libStatus) {
		l.Albums, l.Artists = a.count(ctx, store.KindAlbum), a.count(ctx, store.KindArtist)
	})
	add("games", "Jogos", cfg.Games, store.KindRom, func(l *libStatus) { l.Platforms = a.count(ctx, store.KindPlatform) })
	return out
}

// sessions: quem usou um token nos últimos minutos (um por aparelho).
func (a *api) sessions(ctx context.Context) ([]session, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT u.name, t.client, t.device, t.device_id, t.last_used
FROM tokens t JOIN users u ON u.id = t.user_id WHERE t.last_used IS NOT NULL AND t.kind != ?`, auth.KindRomMRefresh)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type seen struct {
		s  session
		at time.Time
	}
	byKey := map[string]seen{}
	for rows.Next() {
		var s session
		var devID, used string
		if err := rows.Scan(&s.User, &s.Client, &s.Device, &devID, &used); err != nil {
			return nil, err
		}
		at, err := time.Parse(time.RFC3339Nano, used)
		if err != nil || time.Since(at) > sessionWindow {
			continue
		}
		s.LastSeen = date(at)
		key := s.User + "\x00" + s.Client + "\x00" + s.Device + "\x00" + devID
		if old, ok := byKey[key]; !ok || at.After(old.at) {
			byKey[key] = seen{s, at}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	list := make([]seen, 0, len(byKey))
	for _, v := range byKey {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].at.After(list[j].at) })
	out := make([]session, len(list))
	for i, v := range list {
		out[i] = v.s
	}
	return out, nil
}

// GET /status
func (a *api) status(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	ctx := r.Context()
	sess, err := a.sessions(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	sc := a.d.Library.Status()
	meta := map[string]any{"enabled": a.d.Config.Metadata, "tmdb": a.d.Config.TMDBKey != "", "running": false, "pending": 0, "error": ""}
	if a.d.Meta != nil {
		ms := a.d.Meta.Status()
		meta["running"], meta["pending"], meta["error"] = ms.Running, ms.Pending, ms.Error
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version": a.d.Version, "writable": a.d.Config.Writable, "startedAt": date(a.started),
		"libraries": a.libraries(ctx),
		"scan": map[string]any{"running": sc.Running, "startedAt": date(sc.Started), "finishedAt": date(sc.Finished),
			"count": sc.Count, "error": sc.Error},
		"metadata": meta,
		"sessions": sess,
	})
}
