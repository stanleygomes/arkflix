package jellyfin

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

// Proxy serve imagens, vídeo, HLS e legendas do Jellyfin em 127.0.0.1 com o
// token no cabeçalho: a interface nunca vê o token, não depende do CORS do
// servidor nem esbarra em conteúdo misto (http em página segura). Só GET e
// HEAD de /Items/ e /Videos/, atrás de um prefixo secreto por execução: outro
// programa da máquina não usa a conta pelo proxy.
type Proxy struct {
	// Account dá a conta do perfil em uso (false = sem conta vinculada).
	Account func() (Account, bool)

	mu     sync.Mutex
	base   string // http://127.0.0.1:<porta>/<segredo>
	secret string
	ln     net.Listener
}

// URL liga o proxy na primeira chamada e devolve a base (sem barra no fim).
func (p *Proxy) URL() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.base != "" {
		return p.base, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	buf := make([]byte, 16)
	rand.Read(buf)
	p.secret, p.ln = hex.EncodeToString(buf), ln
	p.base = "http://" + ln.Addr().String() + "/" + p.secret
	go http.Serve(ln, p)
	return p.base, nil
}

// Close desliga o proxy.
func (p *Proxy) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ln == nil {
		return nil
	}
	p.base = ""
	return p.ln.Close()
}

var allowed = []string{"/Items/", "/Videos/"}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Range")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	p.mu.Lock()
	secret := p.secret
	p.mu.Unlock()
	rest, ok := strings.CutPrefix(r.URL.Path, "/"+secret)
	if !ok || secret == "" || (r.Method != http.MethodGet && r.Method != http.MethodHead) || !permitted(rest) {
		http.NotFound(w, r)
		return
	}
	acc, ok := p.Account()
	if !ok {
		http.Error(w, "sem conta do Jellyfin", http.StatusServiceUnavailable)
		return
	}
	target, err := url.Parse(acc.Base)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme, pr.Out.URL.Host = target.Scheme, target.Host
			pr.Out.URL.Path = strings.TrimRight(target.Path, "/") + rest
			pr.Out.URL.RawPath = ""
			pr.Out.Host = target.Host
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Del("Origin")
			pr.Out.Header.Del("Referer")
			pr.Out.Header.Set("Authorization", Authorization(acc))
		},
		// o ReverseProxy soma (Add) os cabeçalhos da resposta aos do w: o
		// CORS vem só daqui, senão sai "*, *" e o navegador recusa
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Del("Set-Cookie")
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, "Jellyfin fora do ar: "+err.Error(), http.StatusBadGateway)
		},
	}
	rp.ServeHTTP(w, r)
}

func permitted(p string) bool {
	if strings.Contains(p, "..") {
		return false
	}
	for _, a := range allowed {
		if strings.HasPrefix(p, a) {
			return true
		}
	}
	return false
}
