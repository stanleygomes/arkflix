package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/lists"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// tipos que entram numa lista
var listKinds = []string{store.KindMovie, store.KindSeries, store.KindEpisode, store.KindRom, store.KindAlbum, store.KindTrack}

type listJSON struct {
	ID        string      `json:"id"`
	RID       int64       `json:"rid"`
	Name      string      `json:"name"`
	Count     int         `json:"count"`
	Kinds     []string    `json:"kinds"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	Cover     string      `json:"cover"`
	Items     *[]itemJSON `json:"items,omitempty"`
}

// listItems são os itens que ainda existem, na ordem.
func (a *api) listItems(ctx context.Context, l *lists.List) ([]*store.Item, error) {
	es, err := a.lists.Entries(ctx, l.ID)
	if err != nil {
		return nil, err
	}
	var out []*store.Item
	for _, id := range lists.ItemIDs(es) {
		if it, err := a.d.Store.Item(ctx, id); err == nil {
			out = append(out, it)
		}
	}
	return out, nil
}

func (a *api) listJSON(ctx context.Context, l *lists.List, withItems bool) (listJSON, error) {
	its, err := a.listItems(ctx, l)
	if err != nil {
		return listJSON{}, err
	}
	out := listJSON{ID: l.ID, RID: l.RID, Name: l.Name, Count: len(its), Kinds: []string{},
		CreatedAt: date(l.Created), UpdatedAt: date(l.Updated)}
	var items []itemJSON
	for _, it := range its {
		if !slices.Contains(out.Kinds, it.Kind) {
			out.Kinds = append(out.Kinds, it.Kind)
		}
		if out.Cover == "" && a.imageSrc(ctx, it) != "" {
			out.Cover = it.ID
		}
		if withItems {
			items = append(items, a.itemJSON(ctx, it))
		}
	}
	if withItems {
		if items == nil {
			items = []itemJSON{}
		}
		out.Items = &items
	}
	return out, nil
}

func (a *api) writeList(w http.ResponseWriter, r *http.Request, status int, id string, withItems bool) {
	l, err := a.lists.Get(r.Context(), id)
	if err != nil {
		internal(w, err)
		return
	}
	out, err := a.listJSON(r.Context(), l, withItems)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, status, out)
}

// ownList carrega {id} e exige que seja do usuário (senão 404).
func (a *api) ownList(w http.ResponseWriter, r *http.Request, u *auth.User) *lists.List {
	l, err := a.lists.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) || (err == nil && l.UserID != u.ID) {
		fail(w, http.StatusNotFound, "lista não encontrada")
		return nil
	}
	if err != nil {
		internal(w, err)
		return nil
	}
	return l
}

// GET /lists
func (a *api) listLists(w http.ResponseWriter, r *http.Request, u *auth.User) {
	ls, err := a.lists.Own(r.Context(), u.ID)
	if err != nil {
		internal(w, err)
		return
	}
	out := []listJSON{}
	for _, l := range ls {
		j, err := a.listJSON(r.Context(), l, false)
		if err != nil {
			internal(w, err)
			return
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /lists {name}
func (a *api) createList(w http.ResponseWriter, r *http.Request, u *auth.User) {
	var body struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	l, err := a.lists.Create(r.Context(), u.ID, body.Name)
	if errors.Is(err, lists.ErrName) {
		fail(w, http.StatusBadRequest, "nome vazio")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	a.writeList(w, r, http.StatusCreated, l.ID, false)
}

// GET /lists/{id}
func (a *api) getList(w http.ResponseWriter, r *http.Request, u *auth.User) {
	if l := a.ownList(w, r, u); l != nil {
		a.writeList(w, r, http.StatusOK, l.ID, true)
	}
}

// PATCH /lists/{id} {name}
func (a *api) patchList(w http.ResponseWriter, r *http.Request, u *auth.User) {
	l := a.ownList(w, r, u)
	if l == nil {
		return
	}
	var body struct {
		Name *string `json:"name"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := a.lists.Update(r.Context(), l.ID, lists.Changes{Name: body.Name}); errors.Is(err, lists.ErrName) {
		fail(w, http.StatusBadRequest, "nome vazio")
		return
	} else if err != nil {
		internal(w, err)
		return
	}
	a.writeList(w, r, http.StatusOK, l.ID, false)
}

// DELETE /lists/{id}
func (a *api) deleteList(w http.ResponseWriter, r *http.Request, u *auth.User) {
	l := a.ownList(w, r, u)
	if l == nil {
		return
	}
	if err := a.lists.Delete(r.Context(), l.ID); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// itemIDs troca os ids pedidos ("32hex" ou "rom:<rid>") pelos do acervo;
// item que não existe ou de tipo que não entra em lista: 400.
func (a *api) itemIDs(ctx context.Context, w http.ResponseWriter, ids []string) ([]string, bool) {
	out := []string{}
	for _, id := range ids {
		it, err := a.item(ctx, id)
		if err != nil || !slices.Contains(listKinds, it.Kind) {
			fail(w, http.StatusBadRequest, "item inválido: "+id)
			return nil, false
		}
		out = append(out, it.ID)
	}
	return out, true
}

// POST /lists/{id}/items {ids}
func (a *api) addListItems(w http.ResponseWriter, r *http.Request, u *auth.User) {
	l := a.ownList(w, r, u)
	if l == nil {
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	ids, ok := a.itemIDs(r.Context(), w, body.IDs)
	if !ok {
		return
	}
	if err := a.lists.Add(r.Context(), l.ID, ids); err != nil {
		internal(w, err)
		return
	}
	a.writeList(w, r, http.StatusOK, l.ID, false)
}

// DELETE /lists/{id}/items/{itemId}
func (a *api) removeListItem(w http.ResponseWriter, r *http.Request, u *auth.User) {
	l := a.ownList(w, r, u)
	if l == nil {
		return
	}
	id := r.PathValue("itemId")
	if it, err := a.item(r.Context(), id); err == nil {
		id = it.ID
	}
	if err := a.lists.Remove(r.Context(), l.ID, []string{id}); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /lists/{id}/order {ids}
func (a *api) orderList(w http.ResponseWriter, r *http.Request, u *auth.User) {
	l := a.ownList(w, r, u)
	if l == nil {
		return
	}
	var body struct {
		IDs []string `json:"ids"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	ids := make([]string, 0, len(body.IDs))
	for _, id := range body.IDs {
		if it, err := a.item(r.Context(), id); err == nil {
			id = it.ID
		}
		ids = append(ids, id)
	}
	if err := a.lists.Reorder(r.Context(), l.ID, ids); err != nil {
		internal(w, err)
		return
	}
	a.writeList(w, r, http.StatusOK, l.ID, true)
}
