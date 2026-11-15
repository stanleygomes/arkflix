package romm

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// propsIn é o RomUserData: tudo opcional, só muda o que veio.
type propsIn struct {
	IsMainSibling *bool            `json:"is_main_sibling"`
	Backlogged    *bool            `json:"backlogged"`
	NowPlaying    *bool            `json:"now_playing"`
	Hidden        *bool            `json:"hidden"`
	Rating        *int             `json:"rating"`
	Difficulty    *int             `json:"difficulty"`
	Completion    *int             `json:"completion"`
	Status        *json.RawMessage `json:"status"`
	PinnedMedia   *json.RawMessage `json:"pinned_media"`
	Data          *propsIn         `json:"data"` // formato antigo: {"data": {...}}
	// formato antigo também mandava no corpo
	UpdateLastPlayed *bool `json:"update_last_played"`
	RemoveLastPlayed *bool `json:"remove_last_played"`
}

// props é o PUT /api/roms/{id}/props (rom_user do usuário). last_played vai
// pro userdata comum, que as outras APIs também enxergam.
func (a *api) props(w http.ResponseWriter, r *http.Request, c *caller) {
	rom, _, ok := a.romItem(w, r)
	if !ok {
		return
	}
	var in propsIn
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
		fail(w, http.StatusUnprocessableEntity, "Invalid body")
		return
	}
	data := &in
	if in.Data != nil {
		data = in.Data
	}
	upd, err := queryBool(r, "update_last_played")
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	rem, err := queryBool(r, "remove_last_played")
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if upd == nil {
		upd = in.UpdateLastPlayed
	}
	if rem == nil {
		rem = in.RemoveLastPlayed
	}
	if upd != nil && *upd && rem != nil && *rem {
		fail(w, http.StatusBadRequest, "update_last_played and remove_last_played are mutually exclusive.")
		return
	}
	for _, v := range []*int{data.Rating, data.Difficulty} {
		if v != nil && (*v < 0 || *v > 10) {
			fail(w, http.StatusUnprocessableEntity, "Value must be between 0 and 10")
			return
		}
	}
	if data.Completion != nil && (*data.Completion < 0 || *data.Completion > 100) {
		fail(w, http.StatusUnprocessableEntity, "completion must be between 0 and 100")
		return
	}
	ctx := r.Context()
	ru, err := a.romUser(ctx, c.user.ID, rom.RID)
	if err != nil {
		internal(w, err)
		return
	}
	if ru == nil {
		ru = &romUser{UserID: c.user.ID, RomRID: rom.RID}
	}
	setB := func(dst *bool, v *bool) {
		if v != nil {
			*dst = *v
		}
	}
	setI := func(dst *int, v *int) {
		if v != nil {
			*dst = *v
		}
	}
	setB(&ru.IsMainSibling, data.IsMainSibling)
	setB(&ru.Backlogged, data.Backlogged)
	setB(&ru.NowPlaying, data.NowPlaying)
	setB(&ru.Hidden, data.Hidden)
	setI(&ru.Rating, data.Rating)
	setI(&ru.Difficulty, data.Difficulty)
	setI(&ru.Completion, data.Completion)
	if data.Status != nil {
		var st *string
		if err := json.Unmarshal(*data.Status, &st); err != nil || (st != nil && !slices.Contains(romUserStatuses, *st)) {
			fail(w, http.StatusUnprocessableEntity, "Invalid status")
			return
		}
		ru.Status = sql.NullString{}
		if st != nil {
			ru.Status = sql.NullString{String: *st, Valid: true}
		}
	}
	if data.PinnedMedia != nil {
		var pm []string
		if err := json.Unmarshal(*data.PinnedMedia, &pm); err != nil {
			fail(w, http.StatusUnprocessableEntity, "Invalid pinned_media")
			return
		}
		ru.PinnedMedia = sql.NullString{}
		if pm != nil {
			seen := map[string]bool{}
			pm = slices.DeleteFunc(pm, func(v string) bool { dup := seen[v]; seen[v] = true; return dup })
			raw, _ := json.Marshal(pm)
			ru.PinnedMedia = sql.NullString{String: string(raw), Valid: true}
		}
	}
	if err := a.saveRomUser(ctx, ru); err != nil {
		internal(w, err)
		return
	}
	if (upd != nil && *upd) || (rem != nil && *rem) {
		_, err := a.d.Store.UpdateUserData(ctx, c.user.ID, rom.ID, func(d *store.UserData) {
			if upd != nil && *upd {
				d.LastPlayed = time.Now().UTC()
			} else {
				d.LastPlayed = time.Time{}
			}
		})
		if err != nil {
			internal(w, err)
			return
		}
	}
	s, err := a.romUserFor(ctx, c, rom)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}
