package subsonic

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Proxy serve áudio e capas do servidor de músicas em 127.0.0.1, pondo a
// autenticação na query aqui dentro: a interface nunca vê t, s nem a chave.
// Só GET e HEAD de /rest/stream, /rest/getCoverArt e /rest/download, atrás de
// um prefixo secreto por execução. /local/<id> serve a cópia baixada da faixa
// (toca sem internet).
type Proxy struct {
	// Account dá a conta do perfil em uso (false = sem conta vinculada).
	Account func() (Account, bool)
	// Local acha o arquivo baixado de uma faixa (false = não tem).
	Local func(id string) (string, bool)

	mu     sync.Mutex
	base   string
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

// métodos que passam (com ou sem .view)
var allowed = map[string]bool{"stream": true, "getCoverArt": true, "download": true}

// parâmetros que a interface pode mandar (o resto cai, inclusive login)
var passQuery = map[string]bool{"id": true, "size": true, "format": true, "maxBitRate": true, "timeOffset": true, "estimateContentLength": true}

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
	if !ok || secret == "" || (r.Method != http.MethodGet && r.Method != http.MethodHead) || strings.Contains(rest, "..") {
		http.NotFound(w, r)
		return
	}
	if id, ok := strings.CutPrefix(rest, "/local/"); ok {
		p.serveLocal(w, r, id)
		return
	}
	method, ok := strings.CutPrefix(rest, "/rest/")
	method = strings.TrimSuffix(method, ".view")
	if !ok || !allowed[method] {
		http.NotFound(w, r)
		return
	}
	acc, ok := p.Account()
	if !ok {
		http.Error(w, "sem conta de músicas", http.StatusServiceUnavailable)
		return
	}
	target, err := url.Parse(acc.Base)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	q := acc.Auth()
	for k, v := range r.URL.Query() {
		if passQuery[k] {
			q[k] = v
		}
	}
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme, pr.Out.URL.Host = target.Scheme, target.Host
			pr.Out.URL.Path = strings.TrimRight(target.Path, "/") + "/rest/" + method + ".view"
			pr.Out.URL.RawPath = ""
			pr.Out.URL.RawQuery = q.Encode()
			pr.Out.Host = target.Host
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Del("Origin")
			pr.Out.Header.Del("Referer")
			pr.Out.Header.Del("Authorization")
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Del("Set-Cookie")
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, "servidor de músicas fora do ar: "+stripURL(err).Error(), http.StatusBadGateway)
		},
	}
	rp.ServeHTTP(w, r)
}

func (p *Proxy) serveLocal(w http.ResponseWriter, r *http.Request, id string) {
	if p.Local == nil || id == "" || strings.ContainsAny(id, "/\\") {
		http.NotFound(w, r)
		return
	}
	path, ok := p.Local(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if ct, ok := audioTypes[strings.ToLower(filepath.Ext(path))]; ok {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

// audioTypes: o tipo das cópias baixadas (o sistema nem sempre conhece flac).
var audioTypes = map[string]string{".mp3": "audio/mpeg", ".flac": "audio/flac", ".ogg": "audio/ogg", ".oga": "audio/ogg",
	".opus": "audio/ogg", ".m4a": "audio/mp4", ".aac": "audio/aac", ".wav": "audio/wav", ".webm": "audio/webm"}
