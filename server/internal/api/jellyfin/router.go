package jellyfin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// route casa método + caminho. O padrão tem segmentos literais (sem
// diferenciar maiúsculas), {nome} (o segmento todo), "stream.{ext}"
// (prefixo + resto) ou "{seg}.{ext}" (corta no último ponto).
type route struct {
	method string // "" = qualquer; GET também atende HEAD
	segs   []string
	public bool
	h      func(*rc)
}

// rc é o contexto de uma requisição.
type rc struct {
	s      *Server
	w      http.ResponseWriter
	r      *http.Request
	p      map[string]string
	q      map[string][]string // query com chave minúscula
	user   *auth.User
	tok    auth.Token
	ah     authHeader
	target string // usuário dono dos dados (userId do caminho/query)
	memo   map[string]*store.Item
}

// uid é o usuário cujo estado (favorito, onde parou) vale na resposta.
func (c *rc) uid() string {
	if c.target != "" {
		return c.target
	}
	if c.user != nil {
		return c.user.ID
	}
	return ""
}

func split(p string) []string {
	var out []string
	for s := range strings.SplitSeq(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (s *Server) add(method, pattern string, h func(*rc)) {
	s.routes = append(s.routes, route{method: method, segs: split(pattern), h: h})
}

func (s *Server) addPublic(method, pattern string, h func(*rc)) {
	s.routes = append(s.routes, route{method: method, segs: split(pattern), h: h, public: true})
}

func (s *Server) match(method string, segs []string) (route, map[string]string, bool) {
	for _, rt := range s.routes {
		if rt.method != "" && rt.method != method && !(rt.method == http.MethodGet && method == http.MethodHead) {
			continue
		}
		if len(rt.segs) != len(segs) {
			continue
		}
		params := map[string]string{}
		ok := true
		for i, pat := range rt.segs {
			if !matchSeg(pat, segs[i], params) {
				ok = false
				break
			}
		}
		if ok {
			return rt, params, true
		}
	}
	return route{}, nil, false
}

func matchSeg(pat, seg string, params map[string]string) bool {
	open := strings.IndexByte(pat, '{')
	if open < 0 {
		return strings.EqualFold(pat, seg)
	}
	val, err := url.PathUnescape(seg)
	if err != nil {
		val = seg
	}
	// "{a}.{b}": corta no último ponto
	if strings.Count(pat, "{") == 2 {
		dot := strings.LastIndexByte(val, '.')
		if dot <= 0 {
			return false
		}
		names := strings.Split(strings.NewReplacer("{", "", "}", "").Replace(pat), ".")
		params[names[0]], params[names[1]] = val[:dot], val[dot+1:]
		return true
	}
	end := strings.IndexByte(pat, '}')
	prefix, name, suffix := pat[:open], pat[open+1:end], pat[end+1:]
	if len(val) < len(prefix)+len(suffix) || !strings.EqualFold(val[:len(prefix)], prefix) ||
		!strings.EqualFold(val[len(val)-len(suffix):], suffix) {
		return false
	}
	v := val[len(prefix) : len(val)-len(suffix)]
	if v == "" {
		return false
	}
	params[name] = v
	return true
}

func lowerQuery(r *http.Request) map[string][]string {
	out := map[string][]string{}
	for k, v := range r.URL.Query() {
		lk := strings.ToLower(k)
		out[lk] = append(out[lk], v...)
	}
	return out
}

func (c *rc) ctx() context.Context { return c.r.Context() }

// qs lê um parâmetro da query (sem diferenciar maiúsculas).
func (c *rc) qs(name string) string {
	if v := c.q[strings.ToLower(name)]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// ql lê uma lista: "a,b" e/ou a=..&a=.. (sep = separadores aceitos).
func (c *rc) ql(name, sep string) []string {
	var out []string
	for _, v := range c.q[strings.ToLower(name)] {
		for p := range strings.FieldsFuncSeq(v, func(r rune) bool { return strings.ContainsRune(sep, r) }) {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func (c *rc) qi(name string, def int) int {
	if n, err := strconv.Atoi(c.qs(name)); err == nil {
		return n
	}
	return def
}

func (c *rc) qi64(name string) int64 {
	n, _ := strconv.ParseInt(c.qs(name), 10, 64)
	return n
}

// qb devolve (valor, veio?).
func (c *rc) qb(name string) (bool, bool) {
	v := c.qs(name)
	if v == "" {
		return false, false
	}
	b, err := strconv.ParseBool(strings.ToLower(v))
	return b, err == nil
}

// pid lê um id do caminho, normalizado.
func (c *rc) pid(name string) string { return store.NormalizeID(c.p[name]) }

// body decodifica o JSON do corpo (corpo vazio não é erro).
func (c *rc) body(v any) error {
	err := json.NewDecoder(io.LimitReader(c.r.Body, 4<<20)).Decode(v)
	if err == io.EOF {
		return nil
	}
	return err
}

func (c *rc) json(v any) {
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.w.Header().Set("Cache-Control", "no-cache")
	if c.r.Method == http.MethodHead {
		return
	}
	json.NewEncoder(c.w).Encode(v)
}

func (c *rc) noContent() { c.w.WriteHeader(http.StatusNoContent) }

func (c *rc) fail(code int, msg string) { http.Error(c.w, msg, code) }

func (c *rc) notFound() { notFound(c.w) }

func (c *rc) err(e error) {
	if e == store.ErrNotFound {
		c.notFound()
		return
	}
	c.fail(http.StatusInternalServerError, e.Error())
}
