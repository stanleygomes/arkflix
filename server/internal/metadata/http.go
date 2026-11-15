package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// errNotFound: o provedor respondeu 404 (não é falha de rede).
var errNotFound = errors.New("não achado no provedor")

// limiter espaça as requisições de um provedor (every entre uma e outra).
type limiter struct {
	mu    sync.Mutex
	every time.Duration
	next  time.Time
}

func (l *limiter) wait(ctx context.Context) error {
	l.mu.Lock()
	now := time.Now()
	at := l.next
	if at.Before(now) {
		at = now
	}
	l.next = at.Add(l.every)
	l.mu.Unlock()
	if d := at.Sub(now); d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	return nil
}

// getter faz GET com limite, User-Agent e nova tentativa (429/5xx/rede).
type getter struct {
	client  *http.Client
	ua      func() string
	tries   int           // tentativas no total
	backoff time.Duration // espera base, dobra a cada tentativa
	maxWait time.Duration // teto da espera (Retry-After incluso)
}

const maxBody = 32 << 20

// get devolve o corpo do 200; 404 vira errNotFound.
func (g *getter) get(ctx context.Context, lim *limiter, url string, hdr http.Header) ([]byte, error) {
	var last error
	for try := 0; try < g.tries; try++ {
		if err := lim.wait(ctx); err != nil {
			return nil, err
		}
		body, retry, err := g.once(ctx, url, hdr)
		if err == nil || retry < 0 {
			return body, err
		}
		last = err
		if try == g.tries-1 {
			break
		}
		wait := max(retry, g.backoff<<try)
		t := time.NewTimer(min(wait, g.maxWait))
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
	return nil, last
}

// once faz uma requisição. retry < 0 = não adianta tentar de novo; >= 0 é
// a espera pedida pelo servidor (Retry-After).
func (g *getter) once(ctx context.Context, url string, hdr http.Header) ([]byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, -1, err
	}
	for k, v := range hdr {
		req.Header[k] = v
	}
	req.Header.Set("User-Agent", g.ua())
	resp, err := g.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, -1, ctx.Err()
		}
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	switch {
	case resp.StatusCode == http.StatusOK && err == nil:
		return body, 0, nil
	case resp.StatusCode == http.StatusOK:
		return nil, 0, err
	case resp.StatusCode == http.StatusNotFound:
		return nil, -1, errNotFound
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 503 || resp.StatusCode >= 500:
		return nil, retryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("%s: HTTP %d", host(url), resp.StatusCode)
	}
	return nil, -1, fmt.Errorf("%s: HTTP %d", host(url), resp.StatusCode)
}

func retryAfter(v string) time.Duration {
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return 0
}

func host(url string) string {
	rest := url[strings.Index(url, "//")+2:]
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return rest[:i]
	}
	return rest
}

// getJSON decodifica o corpo em v.
func (g *getter) getJSON(ctx context.Context, lim *limiter, url string, hdr http.Header, v any) error {
	body, err := g.get(ctx, lim, url, hdr)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("%s: resposta inválida: %w", host(url), err)
	}
	return nil
}

// download baixa a imagem em dir/name.<ext da url> (.part e troca) e
// devolve o caminho final.
func (g *getter) download(ctx context.Context, lim *limiter, url, dir, name string) (string, error) {
	body, err := g.get(ctx, lim, url, nil)
	if err != nil {
		return "", err
	}
	if len(body) == 0 || !strings.HasPrefix(http.DetectContentType(body), "image/") {
		return "", fmt.Errorf("%s: não é imagem", host(url))
	}
	ext := strings.ToLower(path.Ext(strings.SplitN(url, "?", 2)[0]))
	if ext != ".png" && ext != ".webp" {
		ext = ".jpg"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, name+ext)
	part := dst + ".part"
	if err := os.WriteFile(part, body, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(part, dst); err != nil {
		os.Remove(part)
		return "", err
	}
	// sobra de outra extensão (pôster que era .png e virou .jpg)
	for _, e := range []string{".jpg", ".png", ".webp"} {
		if e != ext {
			os.Remove(filepath.Join(dir, name+e))
		}
	}
	return dst, nil
}
