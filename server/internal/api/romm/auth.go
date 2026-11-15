package romm

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
)

// Escopos do RomM (handler/auth/constants.py).
const (
	scopeMeRead            = "me.read"
	scopeMeWrite           = "me.write"
	scopeRomsRead          = "roms.read"
	scopeRomsWrite         = "roms.write"
	scopeRomsUserRead      = "roms.user.read"
	scopeRomsUserWrite     = "roms.user.write"
	scopePlatformsRead     = "platforms.read"
	scopePlatformsWrite    = "platforms.write"
	scopeAssetsRead        = "assets.read"
	scopeAssetsWrite       = "assets.write"
	scopeDevicesRead       = "devices.read"
	scopeDevicesWrite      = "devices.write"
	scopeFirmwareRead      = "firmware.read"
	scopeFirmwareWrite     = "firmware.write"
	scopeCollectionsRead   = "collections.read"
	scopeCollectionsWrite  = "collections.write"
	scopePlaylistsRead     = "playlists.read"
	scopePlaylistsWrite    = "playlists.write"
	scopeUsersRead         = "users.read"
	scopeUsersWrite        = "users.write"
	scopeTasksRun          = "tasks.run"
	scopeLogsRead          = "logs.read"
	accessTTL              = 30 * time.Minute
	refreshTTL             = 7 * 24 * time.Hour
	sessionTTL             = 14 * 24 * time.Hour
	sessionCookie          = "romm_session"
	clientPrefix           = "rmm_"
	maxClientTokensPerUser = 25
)

// usuário comum lê e grava o que é dele; admin tem tudo (WRITE_SCOPES e
// FULL_SCOPES do RomM).
var (
	writeScopes = []string{scopeMeRead, scopeRomsRead, scopePlatformsRead, scopeAssetsRead, scopeDevicesRead,
		scopeFirmwareRead, scopeRomsUserRead, scopeCollectionsRead, scopePlaylistsRead,
		scopeMeWrite, scopeAssetsWrite, scopeDevicesWrite, scopeRomsUserWrite, scopeCollectionsWrite, scopePlaylistsWrite}
	fullScopes = append(append([]string{}, writeScopes...), scopeRomsWrite, scopePlatformsWrite, scopeFirmwareWrite,
		scopeUsersRead, scopeUsersWrite, scopeTasksRun, scopeLogsRead)
)

func userScopes(u *auth.User) []string {
	if u.Admin {
		return fullScopes
	}
	return writeScopes
}

// caller é quem fez a requisição e o que pode.
type caller struct {
	user     *auth.User
	scopes   map[string]bool
	clientID int64 // id do client token usado (0 = outro jeito)
}

// newCaller limita os escopos pedidos aos que o usuário tem.
func newCaller(u *auth.User, asked []string) *caller {
	has := userScopes(u)
	c := &caller{user: u, scopes: map[string]bool{}}
	for _, s := range asked {
		if slices.Contains(has, s) {
			c.scopes[s] = true
		}
	}
	return c
}

// authenticate tenta sessão, Basic, client token e access token, nessa
// ordem. Credencial inválida = sem login (a rota decide se dá 401).
func (a *api) authenticate(r *http.Request) *caller {
	ctx := r.Context()
	if ck, err := r.Cookie(sessionCookie); err == nil && ck.Value != "" {
		if u, _, err := a.d.Auth.CheckToken(ctx, ck.Value, auth.KindRomM); err == nil {
			return newCaller(u, userScopes(u))
		}
	}
	scheme, cred, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	if !ok {
		return nil
	}
	cred = strings.TrimSpace(cred)
	switch strings.ToLower(scheme) {
	case "basic":
		raw, err := base64.StdEncoding.DecodeString(cred)
		if err != nil {
			return nil
		}
		name, pass, ok := strings.Cut(string(raw), ":")
		if !ok {
			return nil
		}
		u := a.basicLogin(ctx, name, pass)
		if u == nil {
			return nil
		}
		return newCaller(u, userScopes(u))
	case "bearer":
		if strings.HasPrefix(cred, clientPrefix) {
			u, _, err := a.d.Auth.CheckToken(ctx, cred, auth.KindRomMClient)
			if err != nil {
				return nil
			}
			ct, err := a.clientTokenByValue(ctx, cred)
			if err != nil {
				return nil
			}
			c := newCaller(u, ct.Scopes)
			c.clientID = ct.ID
			return c
		}
		u, t, err := a.d.Auth.CheckToken(ctx, cred, auth.KindRomM)
		if err != nil || t.Client == "session" {
			return nil
		}
		return newCaller(u, strings.Fields(t.Name))
	}
	return nil
}

// basicCache guarda logins Basic que deram certo por uns minutos: tem
// cliente (WebDAV, muOS) que manda Basic em toda requisição e o bcrypt
// custa caro.
type basicCache struct {
	mu sync.Mutex
	m  map[string]basicEntry
}

type basicEntry struct {
	userID string
	until  time.Time
}

const basicTTL = 5 * time.Minute

func newBasicCache() *basicCache { return &basicCache{m: map[string]basicEntry{}} }

func (a *api) basicLogin(ctx context.Context, name, pass string) *auth.User {
	sum := sha256.Sum256([]byte(name + "\x00" + pass))
	key := hex.EncodeToString(sum[:])
	a.basic.mu.Lock()
	e, ok := a.basic.m[key]
	a.basic.mu.Unlock()
	if ok && time.Now().Before(e.until) {
		if u, err := a.d.Auth.User(ctx, e.userID); err == nil {
			return u
		}
	}
	u, err := a.d.Auth.Login(ctx, name, pass)
	if err != nil {
		return nil
	}
	a.basic.mu.Lock()
	defer a.basic.mu.Unlock()
	now := time.Now()
	for k, v := range a.basic.m { // limpa os vencidos
		if now.After(v.until) {
			delete(a.basic.m, k)
		}
	}
	a.basic.m[key] = basicEntry{userID: u.ID, until: now.Add(basicTTL)}
	return u
}

// token é o POST /api/token (OAuth2: password e refresh_token).
func (a *api) token(w http.ResponseWriter, r *http.Request, _ *caller) {
	ctx := r.Context()
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.ParseMultipartForm(1 << 20)
	} else {
		r.ParseForm()
	}
	var u *auth.User
	var scopes []string
	switch r.PostFormValue("grant_type") {
	case "refresh_token":
		old := r.PostFormValue("refresh_token")
		if old == "" {
			fail(w, http.StatusBadRequest, "Missing refresh token")
			return
		}
		ru, t, err := a.d.Auth.CheckToken(ctx, old, auth.KindRomMRefresh)
		if err != nil {
			fail(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		a.d.Auth.RevokeToken(ctx, old) // refresh é de uso único
		u, scopes = ru, strings.Fields(t.Name)
	case "password":
		name, pass := r.PostFormValue("username"), r.PostFormValue("password")
		if name == "" || pass == "" {
			fail(w, http.StatusBadRequest, "Missing username or password")
			return
		}
		lu, err := a.d.Auth.Login(ctx, name, pass)
		if err != nil {
			fail(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		u = lu
		scopes = strings.Fields(r.PostFormValue("scope"))
		has := userScopes(u)
		for _, s := range scopes {
			if !slices.Contains(has, s) {
				fail(w, http.StatusForbidden, "Insufficient scope")
				return
			}
		}
		if len(scopes) == 0 { // sem scope: tudo que o usuário pode
			scopes = has
		}
	case "client_credentials":
		fail(w, http.StatusBadRequest, "Client credentials are not yet supported")
		return
	default:
		fail(w, http.StatusBadRequest, "Invalid or unsupported grant type")
		return
	}
	joined := strings.Join(scopes, " ")
	access, err := a.d.Auth.NewToken(ctx, auth.Token{UserID: u.ID, Kind: auth.KindRomM, Name: joined, Client: "oauth"}, "", accessTTL)
	if err != nil {
		internal(w, err)
		return
	}
	refresh, err := a.d.Auth.NewToken(ctx, auth.Token{UserID: u.ID, Kind: auth.KindRomMRefresh, Name: joined, Client: "oauth"}, "", refreshTTL)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":    access.Token,
		"refresh_token":   refresh.Token,
		"token_type":      "bearer",
		"expires":         int(accessTTL.Seconds()),
		"expires_in":      int(accessTTL.Seconds()),
		"refresh_expires": int(refreshTTL.Seconds()),
		"scope":           joined,
	})
}

// login é o POST /api/login (Basic) que abre sessão por cookie.
func (a *api) login(w http.ResponseWriter, r *http.Request, _ *caller) {
	name, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic")
		fail(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	u, err := a.d.Auth.Login(r.Context(), name, pass)
	if err != nil {
		fail(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	t, err := a.d.Auth.NewToken(r.Context(), auth.Token{UserID: u.ID, Kind: auth.KindRomM, Client: "session"}, "", sessionTTL)
	if err != nil {
		internal(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: t.Token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, Expires: t.Expires})
	writeJSON(w, http.StatusOK, nil)
}

// logout fecha a sessão do cookie.
func (a *api) logout(w http.ResponseWriter, r *http.Request, _ *caller) {
	if ck, err := r.Cookie(sessionCookie); err == nil && ck.Value != "" {
		if _, t, err := a.d.Auth.CheckToken(r.Context(), ck.Value, auth.KindRomM); err == nil && t.Client == "session" {
			a.d.Auth.RevokeToken(r.Context(), ck.Value)
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, nil)
}
