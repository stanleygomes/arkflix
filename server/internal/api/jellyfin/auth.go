package jellyfin

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// authHeader é o `Authorization: MediaBrowser Client="..", Device="..",
// DeviceId="..", Version="..", Token=".."`.
type authHeader struct {
	Client, Device, DeviceID, Version, Token string
}

func parseAuthHeader(v string) authHeader {
	var a authHeader
	v = strings.TrimSpace(v)
	// tira o esquema ("MediaBrowser", "Emby")
	if sp := strings.IndexByte(v, ' '); sp > 0 && !strings.Contains(v[:sp], "=") {
		v = v[sp+1:]
	}
	for _, part := range splitQuoted(v) {
		k, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"`)
		if u, err := url.PathUnescape(val); err == nil {
			val = u
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "client":
			a.Client = val
		case "device":
			a.Device = val
		case "deviceid":
			a.DeviceID = val
		case "version":
			a.Version = val
		case "token":
			a.Token = val
		}
	}
	return a
}

// splitQuoted corta por vírgula fora de aspas.
func splitQuoted(s string) []string {
	var out []string
	var cur strings.Builder
	quoted := false
	for _, r := range s {
		switch {
		case r == '"':
			quoted = !quoted
			cur.WriteRune(r)
		case r == ',' && !quoted:
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	return append(out, cur.String())
}

func requestAuth(r *http.Request, q map[string][]string) (authHeader, string) {
	a := parseAuthHeader(r.Header.Get("Authorization"))
	if a == (authHeader{}) {
		a = parseAuthHeader(r.Header.Get("X-Emby-Authorization"))
	}
	tok := a.Token
	for _, h := range []string{"X-Emby-Token", "X-MediaBrowser-Token"} {
		if tok == "" {
			tok = r.Header.Get(h)
		}
	}
	for _, k := range []string{"api_key", "apikey", "x-emby-token"} {
		if v := q[k]; tok == "" && len(v) > 0 {
			tok = v[0]
		}
	}
	if q := q["deviceid"]; a.DeviceID == "" && len(q) > 0 {
		a.DeviceID = q[0]
	}
	return a, tok
}

func (s *Server) authenticate(c *rc) bool {
	ah, tok := requestAuth(c.r, c.q)
	c.ah = ah
	u, t, err := s.d.Auth.CheckToken(c.ctx(), tok, auth.KindJellyfin)
	if err != nil {
		return false
	}
	c.user, c.tok = u, t
	s.sessions.touch(t, u, c.r)
	return true
}

// optionalAuth: rota pública, mas se vier token válido, usa.
func (s *Server) optionalAuth(c *rc) {
	ah, tok := requestAuth(c.r, c.q)
	c.ah = ah
	if tok == "" {
		return
	}
	if u, t, err := s.d.Auth.CheckToken(c.ctx(), tok, auth.KindJellyfin); err == nil {
		c.user, c.tok = u, t
	}
}

// POST /Users/AuthenticateByName {Username, Pw}
func (s *Server) login(c *rc) {
	var body struct{ Username, Pw, Password string }
	if err := c.body(&body); err != nil {
		c.fail(http.StatusBadRequest, "corpo inválido")
		return
	}
	if body.Pw == "" {
		body.Pw = body.Password
	}
	u, err := s.d.Auth.Login(c.ctx(), body.Username, body.Pw)
	if errors.Is(err, auth.ErrBadLogin) {
		c.fail(http.StatusUnauthorized, "Error processing request.")
		return
	}
	if err != nil {
		c.err(err)
		return
	}
	ah := c.ah
	if ah.DeviceID == "" {
		ah.DeviceID = store.ID("device", ah.Client+"|"+c.r.UserAgent()+"|"+u.ID)
	}
	if ah.Client == "" {
		ah.Client = "Desconhecido"
	}
	if ah.Device == "" {
		ah.Device = ah.Client
	}
	// mesmo aparelho logando de novo: o token velho cai
	for _, t := range s.deviceTokens(c, u.ID, ah.DeviceID) {
		s.sessions.drop(t)
		s.hls.stopToken(t)
	}
	if err := s.d.Auth.RevokeDevice(c.ctx(), u.ID, auth.KindJellyfin, ah.DeviceID); err != nil {
		c.err(err)
		return
	}
	t, err := s.d.Auth.NewToken(c.ctx(), auth.Token{UserID: u.ID, Kind: auth.KindJellyfin, Name: ah.Device,
		Client: ah.Client, Device: ah.Device, DeviceID: ah.DeviceID, Version: ah.Version}, "", 0)
	if err != nil {
		c.err(err)
		return
	}
	c.user, c.tok = u, t
	sess := s.sessions.touch(t, u, c.r)
	c.json(map[string]any{
		"User":        s.userDto(u),
		"SessionInfo": s.sessionDto(sess),
		"AccessToken": t.Token,
		"ServerId":    s.serverID,
	})
}

func (s *Server) deviceTokens(c *rc, userID, deviceID string) []string {
	ts, _ := s.d.Auth.Tokens(c.ctx(), userID, auth.KindJellyfin)
	var out []string
	for _, t := range ts {
		if t.DeviceID == deviceID {
			out = append(out, t.Token)
		}
	}
	return out
}

// POST /Sessions/Logout
func (s *Server) logout(c *rc) {
	s.d.Auth.RevokeToken(c.ctx(), c.tok.Token)
	s.sessions.drop(c.tok.Token)
	s.hls.stopToken(c.tok.Token)
	c.noContent()
}

// sessions guarda em memória quem está conectado e o que está tocando.
type sessions struct {
	mu   sync.Mutex
	byTk map[string]*session
}

type session struct {
	ID         string
	Token      auth.Token
	UserID     string
	UserName   string
	Remote     string
	LastActive time.Time
	LastCheck  time.Time
	Playing    string // id do item
	Position   int64  // ticks
	Paused     bool
	PlaySess   string
}

func newSessions() *sessions { return &sessions{byTk: map[string]*session{}} }

func (ss *sessions) touch(t auth.Token, u *auth.User, r *http.Request) session {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	s := ss.byTk[t.Token]
	if s == nil {
		s = &session{ID: store.ID("session", t.Token), Token: t}
		ss.byTk[t.Token] = s
	}
	s.UserID, s.UserName, s.Remote, s.LastActive = u.ID, u.Name, r.RemoteAddr, time.Now().UTC()
	return *s
}

func (ss *sessions) drop(token string) {
	ss.mu.Lock()
	delete(ss.byTk, token)
	ss.mu.Unlock()
}

// playing atualiza o que a sessão está tocando (item "" = parou).
func (ss *sessions) playing(token, item, playSess string, pos int64, paused bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if s := ss.byTk[token]; s != nil {
		s.Playing, s.PlaySess, s.Position, s.Paused, s.LastCheck = item, playSess, pos, paused, time.Now().UTC()
	}
}

func (ss *sessions) list(userID string) []session {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	var out []session
	for _, s := range ss.byTk {
		if userID == "" || s.UserID == userID {
			out = append(out, *s)
		}
	}
	return out
}
