package romm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// As coleções são as listas de internal/lists (as mesmas do Jellyfin, do
// Subsonic e do app): aparecem as do usuário e as públicas que têm ROM (ou
// estão vazias), só com as ROMs. O id da coleção é o rid da lista.

// collectionSchema é o CollectionSchema do RomM.
type collectionSchema struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	RomIDs          []int64  `json:"rom_ids"`
	RomCount        int      `json:"rom_count"`
	PathCoverSmall  *string  `json:"path_cover_small"`
	PathCoverLarge  *string  `json:"path_cover_large"`
	PathCoversSmall []string `json:"path_covers_small"`
	PathCoversLarge []string `json:"path_covers_large"`
	URLCover        *string  `json:"url_cover"`
	IsPublic        bool     `json:"is_public"`
	IsFavorite      bool     `json:"is_favorite"`
	IsVirtual       bool     `json:"is_virtual"`
	IsSmart         bool     `json:"is_smart"`
	UserID          int64    `json:"user_id"`
	OwnerUsername   string   `json:"owner_username"`
	CreatedAt       ts       `json:"created_at"`
	UpdatedAt       ts       `json:"updated_at"`
}

func (a *api) lists() *lists.Lists { return lists.New(a.d.Store) }

// collection monta o schema; covers são as capas das 4 primeiras ROMs.
func (a *api) collection(ctx context.Context, l *lists.List, es []lists.Entry) (collectionSchema, error) {
	s := collectionSchema{ID: l.RID, Name: l.Name, Description: l.Comment, RomIDs: []int64{}, IsPublic: l.Public,
		PathCoversSmall: []string{}, PathCoversLarge: []string{}, OwnerUsername: l.Owner,
		CreatedAt: ts(l.Created), UpdatedAt: ts(l.Updated)}
	if u, err := a.d.Auth.User(ctx, l.UserID); err == nil {
		s.UserID = u.RID
	}
	seen := map[int64]bool{}
	for _, id := range lists.ItemIDs(es, store.KindRom) {
		it, err := a.d.Store.Item(ctx, id)
		if err != nil {
			continue
		}
		if seen[it.RID] {
			continue
		}
		seen[it.RID] = true
		s.RomIDs = append(s.RomIDs, it.RID)
		if len(s.PathCoversSmall) < 4 {
			if p, err := a.platformOf(ctx, it); err == nil && it.Image != "" {
				s.PathCoversSmall = append(s.PathCoversSmall, coverPath(p.RID, it, "small"))
				s.PathCoversLarge = append(s.PathCoversLarge, coverPath(p.RID, it, "big"))
			}
		}
	}
	s.RomCount = len(s.RomIDs)
	return s, nil
}

// visibleCollections são as listas que viram coleção pro usuário.
func (a *api) visibleCollections(ctx context.Context, c *caller) ([]collectionSchema, error) {
	ls, err := a.lists().Visible(ctx, c.user.ID)
	if err != nil {
		return nil, err
	}
	out := []collectionSchema{}
	for _, l := range ls {
		es, err := a.lists().Entries(ctx, l.ID)
		if err != nil {
			return nil, err
		}
		if !lists.HasKinds(es, store.KindRom) {
			continue
		}
		s, err := a.collection(ctx, l, es)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (a *api) collections(w http.ResponseWriter, r *http.Request, c *caller) {
	out, err := a.visibleCollections(r.Context(), c)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) collectionIDs(w http.ResponseWriter, r *http.Request, c *caller) {
	out, err := a.visibleCollections(r.Context(), c)
	if err != nil {
		internal(w, err)
		return
	}
	ids := []int64{}
	for _, s := range out {
		ids = append(ids, s.ID)
	}
	writeJSON(w, http.StatusOK, ids)
}

// loadCollection lê {id}: 404 se não existe, 403 se é privada de outro
// (ou, com own, se não é do usuário).
func (a *api) loadCollection(w http.ResponseWriter, r *http.Request, c *caller, own bool) (*lists.List, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return nil, false
	}
	l, err := a.lists().ByRID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, fmt.Sprintf("Collection with id '%d' not found", id))
		return nil, false
	}
	if err != nil {
		internal(w, err)
		return nil, false
	}
	if l.UserID != c.user.ID && (own || !l.Public) {
		fail(w, http.StatusForbidden, fmt.Sprintf("Permission denied for collection with id '%d'", id))
		return nil, false
	}
	return l, true
}

func (a *api) writeCollection(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	l, err := a.lists().Get(ctx, id)
	if err != nil {
		internal(w, err)
		return
	}
	es, err := a.lists().Entries(ctx, l.ID)
	if err != nil {
		internal(w, err)
		return
	}
	s, err := a.collection(ctx, l, es)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *api) getCollection(w http.ResponseWriter, r *http.Request, c *caller) {
	if l, ok := a.loadCollection(w, r, c, false); ok {
		a.writeCollection(w, r, l.ID)
	}
}

// collectionForm lê o formulário do RomM (multipart: name, description,
// rom_ids como JSON; is_public na query) ou um JSON com os mesmos campos.
type collectionForm struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
	RomIDs      []int64 `json:"rom_ids"`
	hasRoms     bool
}

func readCollectionForm(r *http.Request) (collectionForm, error) {
	var f collectionForm
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&raw); err != nil {
			return f, errors.New("Invalid JSON")
		}
		b, _ := json.Marshal(raw)
		if err := json.Unmarshal(b, &f); err != nil {
			return f, errors.New("Invalid JSON")
		}
		_, f.hasRoms = raw["rom_ids"]
	} else {
		r.ParseMultipartForm(1 << 20)
		if _, ok := r.PostForm["name"]; ok {
			v := r.PostFormValue("name")
			f.Name = &v
		}
		if _, ok := r.PostForm["description"]; ok {
			v := r.PostFormValue("description")
			f.Description = &v
		}
		if v, ok := r.PostForm["rom_ids"]; ok {
			f.hasRoms = true
			if err := json.Unmarshal([]byte(v[0]), &f.RomIDs); err != nil {
				return f, errors.New("Invalid list for rom_ids field in update collection")
			}
		}
	}
	b, err := queryBool(r, "is_public")
	if err != nil {
		return f, err
	}
	if b != nil {
		f.IsPublic = b
	}
	return f, nil
}

// romItemIDs troca rids de ROM pelos ids do acervo (o que não é ROM some).
func (a *api) romItemIDs(ctx context.Context, rids []int64) []string {
	var out []string
	for _, rid := range rids {
		if it, err := a.itemOfKind(ctx, rid, store.KindRom); err == nil {
			out = append(out, it.ID)
		}
	}
	return out
}

// nameTaken: o usuário já tem outra lista com esse nome.
func (a *api) nameTaken(ctx context.Context, userID, name, except string) (bool, error) {
	ls, err := a.lists().Own(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, l := range ls {
		if l.ID != except && strings.EqualFold(l.Name, strings.TrimSpace(name)) {
			return true, nil
		}
	}
	return false, nil
}

// POST /api/collections
func (a *api) addCollection(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	f, err := readCollectionForm(r)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	name := ""
	if f.Name != nil {
		name = *f.Name
	}
	if strings.TrimSpace(name) == "" {
		fail(w, http.StatusBadRequest, "Collection name is required")
		return
	}
	if taken, err := a.nameTaken(ctx, c.user.ID, name, ""); err != nil || taken {
		if err != nil {
			internal(w, err)
		} else {
			fail(w, http.StatusConflict, fmt.Sprintf("Collection with name '%s' already exists", name))
		}
		return
	}
	l, err := a.lists().Create(ctx, c.user.ID, name)
	if err != nil {
		internal(w, err)
		return
	}
	if err := a.lists().Update(ctx, l.ID, lists.Changes{Comment: f.Description, Public: f.IsPublic}); err != nil {
		internal(w, err)
		return
	}
	if len(f.RomIDs) > 0 {
		if err := a.lists().Add(ctx, l.ID, a.romItemIDs(ctx, f.RomIDs)); err != nil {
			internal(w, err)
			return
		}
	}
	a.writeCollection(w, r, l.ID)
}

// PUT /api/collections/{id}: troca nome, descrição, visibilidade e as ROMs
// (o resto da lista fica).
func (a *api) updateCollection(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	l, ok := a.loadCollection(w, r, c, true)
	if !ok {
		return
	}
	f, err := readCollectionForm(r)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if f.Name != nil {
		if taken, err := a.nameTaken(ctx, c.user.ID, *f.Name, l.ID); err != nil || taken {
			if err != nil {
				internal(w, err)
			} else {
				fail(w, http.StatusConflict, fmt.Sprintf("Collection with name '%s' already exists", *f.Name))
			}
			return
		}
	}
	if err := a.lists().Update(ctx, l.ID, lists.Changes{Name: f.Name, Comment: f.Description, Public: f.IsPublic}); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if f.hasRoms {
		if err := a.lists().SetKinds(ctx, l.ID, []string{store.KindRom}, a.romItemIDs(ctx, f.RomIDs)); err != nil {
			internal(w, err)
			return
		}
	}
	a.writeCollection(w, r, l.ID)
}

// POST/DELETE /api/collections/{id}/roms {"rom_ids": [..]}
func (a *api) collectionRoms(add bool) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request, c *caller) {
		ctx := r.Context()
		l, ok := a.loadCollection(w, r, c, true)
		if !ok {
			return
		}
		var body struct {
			RomIDs []int64 `json:"rom_ids"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&body); err != nil {
			fail(w, http.StatusUnprocessableEntity, "Invalid rom_ids")
			return
		}
		ids := a.romItemIDs(ctx, body.RomIDs)
		if add {
			err := a.lists().Add(ctx, l.ID, ids)
			if err != nil {
				internal(w, err)
				return
			}
		} else if err := a.lists().Remove(ctx, l.ID, ids); err != nil {
			internal(w, err)
			return
		}
		a.writeCollection(w, r, l.ID)
	}
}

// DELETE /api/collections/{id}: apaga a lista inteira.
func (a *api) deleteCollection(w http.ResponseWriter, r *http.Request, c *caller) {
	l, ok := a.loadCollection(w, r, c, true)
	if !ok {
		return
	}
	if err := a.lists().Delete(r.Context(), l.ID); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}
