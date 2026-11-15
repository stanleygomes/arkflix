package romm

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// validade aceita no expires_in (utils/client_tokens.py)
var tokenExpiry = map[string]time.Duration{"30d": 30 * 24 * time.Hour, "90d": 90 * 24 * time.Hour, "1y": 365 * 24 * time.Hour}

// createClientToken é o POST /api/client-tokens {name, scopes, expires_in}.
func (a *api) createClientToken(w http.ResponseWriter, r *http.Request, c *caller) {
	var in struct {
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		ExpiresIn *string  `json:"expires_in"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
		fail(w, http.StatusUnprocessableEntity, "Invalid body")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Scopes) == 0 {
		fail(w, http.StatusUnprocessableEntity, "name and scopes are required")
		return
	}
	var bad []string
	for _, s := range in.Scopes {
		if !c.scopes[s] {
			bad = append(bad, s)
		}
	}
	if len(bad) > 0 {
		slices.Sort(bad)
		fail(w, http.StatusForbidden, "Requested scopes exceed your permissions: "+strings.Join(bad, ", "))
		return
	}
	var ttl time.Duration
	if in.ExpiresIn != nil && *in.ExpiresIn != "never" {
		d, ok := tokenExpiry[*in.ExpiresIn]
		if !ok {
			fail(w, http.StatusUnprocessableEntity, "Invalid expires_in value: "+*in.ExpiresIn+". Valid values: 30d, 90d, 1y, never")
			return
		}
		ttl = d
	}
	ctx := r.Context()
	have, err := a.clientTokens(ctx, `c.user_id = ?`, c.user.ID)
	if err != nil {
		internal(w, err)
		return
	}
	if len(have) >= maxClientTokensPerUser {
		fail(w, http.StatusBadRequest, "Maximum of 25 tokens per user reached")
		return
	}
	t, err := a.d.Auth.NewToken(ctx, auth.Token{UserID: c.user.ID, Kind: auth.KindRomMClient, Name: in.Name, Client: "romm"}, clientPrefix, ttl)
	if err != nil {
		internal(w, err)
		return
	}
	res, err := a.d.Store.DB.ExecContext(ctx, `INSERT INTO romm_client_tokens (token, user_id, name, scopes, created_at) VALUES (?,?,?,?,?)`,
		t.Token, c.user.ID, in.Name, strings.Join(in.Scopes, " "), store.Now())
	if err != nil {
		a.d.Auth.RevokeToken(ctx, t.Token)
		internal(w, err)
		return
	}
	id, _ := res.LastInsertId()
	ct, err := a.clientTokenByID(ctx, id)
	if err != nil {
		internal(w, err)
		return
	}
	s := newClientTokenSchema(ct, c.user.RID)
	s.RawToken = t.Token
	writeJSON(w, http.StatusCreated, s)
}

func (a *api) listClientTokens(w http.ResponseWriter, r *http.Request, c *caller) {
	list, err := a.clientTokens(r.Context(), `c.user_id = ?`, c.user.ID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]clientTokenSchema, 0, len(list))
	for _, ct := range list {
		out = append(out, newClientTokenSchema(ct, c.user.RID))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) listAllClientTokens(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	list, err := a.clientTokens(ctx, `1 = 1`)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]clientTokenSchema, 0, len(list))
	for _, ct := range list {
		u, err := a.d.Auth.User(ctx, ct.UserID)
		if err != nil {
			continue
		}
		s := newClientTokenSchema(ct, u.RID)
		avatar, upd := "", ts(u.CreatedAt)
		s.Username, s.UserAvatarPath, s.UserUpdatedAt = u.Name, &avatar, &upd
		out = append(out, s)
	}
	writeJSON(w, http.StatusOK, out)
}

// ownToken busca um token do usuário (404 se for de outro).
func (a *api) ownToken(w http.ResponseWriter, r *http.Request, c *caller) (clientToken, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return clientToken{}, false
	}
	ct, err := a.clientTokenByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && ct.UserID != c.user.ID) {
		fail(w, http.StatusNotFound, "Token not found")
		return ct, false
	}
	if err != nil {
		internal(w, err)
		return ct, false
	}
	// token não mexe em outro mais poderoso que ele
	for _, s := range ct.Scopes {
		if !c.scopes[s] {
			fail(w, http.StatusForbidden, "Forbidden")
			return ct, false
		}
	}
	return ct, true
}

// deleteClientToken revoga um token do usuário; o próprio token sempre
// pode se revogar.
func (a *api) deleteClientToken(w http.ResponseWriter, r *http.Request, c *caller) {
	if id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64); id != c.clientID && !c.scopes[scopeMeWrite] {
		fail(w, http.StatusForbidden, "Forbidden")
		return
	}
	ct, ok := a.ownToken(w, r, c)
	if !ok {
		return
	}
	if err := a.d.Auth.RevokeToken(r.Context(), ct.Token); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (a *api) adminDeleteClientToken(w http.ResponseWriter, r *http.Request, c *caller) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	ct, err := a.clientTokenByID(r.Context(), id)
	if err != nil {
		fail(w, http.StatusNotFound, "Token not found")
		return
	}
	a.d.Auth.RevokeToken(r.Context(), ct.Token)
	writeJSON(w, http.StatusOK, nil)
}

// regenerateClientToken troca o valor do token mantendo id, nome e escopos.
func (a *api) regenerateClientToken(w http.ResponseWriter, r *http.Request, c *caller) {
	ct, ok := a.ownToken(w, r, c)
	if !ok {
		return
	}
	ctx := r.Context()
	var ttl time.Duration
	if !ct.Expires.IsZero() {
		ttl = time.Until(ct.Expires)
		if ttl <= 0 {
			ttl = time.Second
		}
	}
	t, err := a.d.Auth.NewToken(ctx, auth.Token{UserID: c.user.ID, Kind: auth.KindRomMClient, Name: ct.Name, Client: "romm"}, clientPrefix, ttl)
	if err != nil {
		internal(w, err)
		return
	}
	if _, err := a.d.Store.DB.ExecContext(ctx, `UPDATE romm_client_tokens SET token = ? WHERE id = ?`, t.Token, ct.ID); err != nil {
		a.d.Auth.RevokeToken(ctx, t.Token)
		internal(w, err)
		return
	}
	a.d.Auth.RevokeToken(ctx, ct.Token)
	nt, err := a.clientTokenByID(ctx, ct.ID)
	if err != nil {
		internal(w, err)
		return
	}
	s := newClientTokenSchema(nt, c.user.RID)
	s.RawToken = t.Token
	writeJSON(w, http.StatusOK, s)
}
