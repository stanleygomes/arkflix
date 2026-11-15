package subsonic

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
)

// params são os da query e do corpo (formPost), nomes com maiúscula igual.
type params url.Values

func (p params) str(name string) string {
	if v := p[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func (p params) has(name string) bool { return len(p[name]) > 0 }

func (p params) strs(name string) []string {
	var out []string
	for _, v := range p[name] {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// need é o parâmetro obrigatório (erro 10 se faltar).
func (p params) need(name string) (string, error) {
	v := p.str(name)
	if v == "" {
		return "", fail(errMissingParam, "missing parameter: '%s'", name)
	}
	return v, nil
}

func (p params) int(name string, def int) int {
	v, err := strconv.Atoi(p.str(name))
	if err != nil {
		return def
	}
	return v
}

func (p params) needInt(name string) (int, error) {
	s, err := p.need(name)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fail(errGeneric, "invalid parameter '%s': expected integer, got '%s'", name, s)
	}
	return v, nil
}

func (p params) bool(name string, def bool) bool {
	v, err := strconv.ParseBool(p.str(name))
	if err != nil {
		return def
	}
	return v
}

// ms lê um instante em milissegundos desde 1970 (zero se não veio/inválido).
func (p params) millis(name string) time.Time {
	v, err := strconv.ParseInt(p.str(name), 10, 64)
	if err != nil || v < 86400000 {
		return time.Time{}
	}
	return time.UnixMilli(v)
}

// conflitam com apiKey
var apiKeyConflicts = []string{"u", "p", "t", "s", "jwt"}

// authenticate segue a ordem do Navidrome: v e c obrigatórios (u também,
// sem apiKey); apiKey não pode vir com outra credencial (43) e chave ruim
// é 44; usuário/senha/token ruins são sempre 40, sem dizer o que errou.
func (a *API) authenticate(c *call) (*auth.User, error) {
	key := c.p.str("apiKey")
	need := []string{"u", "v", "c"}
	if key != "" {
		need = need[1:]
	}
	for _, n := range need {
		if _, err := c.p.need(n); err != nil {
			return nil, err
		}
	}
	if key != "" {
		for _, n := range apiKeyConflicts {
			if c.p.has(n) {
				return nil, fail(errMultiAuth, "")
			}
		}
		u, _, err := a.d.Auth.CheckToken(c.ctx, key, auth.KindSubsonic)
		if err != nil {
			return nil, fail(errBadAPIKey, "")
		}
		return u, nil
	}
	name, pass, token, salt := c.p.str("u"), c.p.str("p"), c.p.str("t"), c.p.str("s")
	var u *auth.User
	var err error
	switch {
	case pass != "":
		u, err = a.d.Auth.SubsonicPassword(c.ctx, name, pass)
	case token != "":
		u, err = a.d.Auth.SubsonicToken(c.ctx, name, strings.TrimSpace(token), salt)
	default:
		err = auth.ErrBadLogin
	}
	if err != nil {
		// qualquer falha vira 40: não conta se foi o usuário ou a senha
		return nil, fail(errBadLogin, "")
	}
	return u, nil
}
