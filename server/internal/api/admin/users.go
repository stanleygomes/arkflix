package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

func userJSON(u *auth.User) map[string]any {
	return map[string]any{"id": u.ID, "rid": u.RID, "name": u.Name, "admin": u.Admin,
		"createdAt": date(u.CreatedAt), "lastLogin": date(u.LastLogin)}
}

// GET /users
func (a *api) users(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	list, err := a.d.Auth.Users(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := []map[string]any{}
	for _, u := range list {
		out = append(out, userJSON(u))
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /users {name, password, admin}
func (a *api) createUser(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Admin    bool   `json:"admin"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || body.Password == "" || strings.ContainsFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		fail(w, http.StatusBadRequest, "nome e senha são obrigatórios")
		return
	}
	u, err := a.d.Auth.CreateUser(r.Context(), name, body.Password, body.Admin)
	if errors.Is(err, auth.ErrExists) {
		fail(w, http.StatusConflict, "usuário já existe")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, userJSON(u))
}

// admins conta os administradores.
func (a *api) admins(ctx context.Context) (int, error) {
	var n int
	err := a.d.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE admin = 1`).Scan(&n)
	return n, err
}

func (a *api) target(w http.ResponseWriter, r *http.Request) *auth.User {
	u, err := a.d.Auth.User(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "usuário não encontrado")
		return nil
	}
	if err != nil {
		internal(w, err)
		return nil
	}
	return u
}

// PATCH /users/{id} {password?, admin?}
func (a *api) patchUser(w http.ResponseWriter, r *http.Request, _ *auth.User) {
	ctx := r.Context()
	u := a.target(w, r)
	if u == nil {
		return
	}
	var body struct {
		Password *string `json:"password"`
		Admin    *bool   `json:"admin"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if body.Password != nil && *body.Password == "" {
		fail(w, http.StatusBadRequest, "senha vazia")
		return
	}
	if body.Admin != nil && !*body.Admin && u.Admin {
		n, err := a.admins(ctx)
		if err != nil {
			internal(w, err)
			return
		}
		if n <= 1 {
			fail(w, http.StatusConflict, "não dá pra tirar o último administrador")
			return
		}
	}
	if body.Password != nil {
		if err := a.d.Auth.SetPassword(ctx, u.ID, *body.Password); err != nil {
			internal(w, err)
			return
		}
	}
	if body.Admin != nil {
		if _, err := a.d.Store.DB.ExecContext(ctx, `UPDATE users SET admin = ? WHERE id = ?`, *body.Admin, u.ID); err != nil {
			internal(w, err)
			return
		}
	}
	if u = a.target(w, r); u != nil {
		writeJSON(w, http.StatusOK, userJSON(u))
	}
}

// DELETE /users/{id}: nem a si mesmo nem o último admin.
func (a *api) deleteUser(w http.ResponseWriter, r *http.Request, me *auth.User) {
	ctx := r.Context()
	u := a.target(w, r)
	if u == nil {
		return
	}
	if u.ID == me.ID {
		fail(w, http.StatusConflict, "não dá pra apagar a si mesmo")
		return
	}
	if u.Admin {
		n, err := a.admins(ctx)
		if err != nil {
			internal(w, err)
			return
		}
		if n <= 1 {
			fail(w, http.StatusConflict, "não dá pra apagar o último administrador")
			return
		}
	}
	if err := a.d.Auth.DeleteUser(ctx, u.ID); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
