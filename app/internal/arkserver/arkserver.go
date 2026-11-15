// Package arkserver fala com a API própria do arkflix-server (/arkflix/api):
// descobrir se o servidor é um arkflix-server, administrar (estado, scan,
// usuários, envio de arquivos, esconder itens) e as listas do usuário. Não
// guarda nada: entra com o token que o app já tem (o do Jellyfin do perfil
// ou o token de dispositivo do RomM).
package arkserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Product é o que /info devolve num arkflix-server.
const Product = "arkflix-server"

const prefix = "/arkflix/api"

var (
	ErrAuth      = errors.New("o servidor arkflix recusou o acesso: vincule a conta de novo")
	ErrForbidden = errors.New("só um administrador do servidor pode fazer isso")
)

// Error é um erro com a mensagem do servidor (status HTTP + texto).
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

// Info é o /info público.
type Info struct {
	Product  string   `json:"product"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Writable bool     `json:"writable"`
	Features []string `json:"features"`
}

// Has diz se o servidor anuncia o recurso ("admin", "lists", "upload"...).
func (i Info) Has(feature string) bool {
	for _, f := range i.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// User é um usuário do servidor.
type User struct {
	ID        string    `json:"id"`
	RID       int       `json:"rid"`
	Name      string    `json:"name"`
	Admin     bool      `json:"admin"`
	CreatedAt time.Time `json:"createdAt"`
	LastLogin time.Time `json:"lastLogin"`
}

// Library é uma biblioteca no /status.
type Library struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Count     int    `json:"count"`
	Episodes  int    `json:"episodes,omitempty"`
	Albums    int    `json:"albums,omitempty"`
	Artists   int    `json:"artists,omitempty"`
	Platforms int    `json:"platforms,omitempty"`
	Online    bool   `json:"online"`
}

// Scan é o estado da varredura.
type Scan struct {
	Running    bool      `json:"running"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Count      int       `json:"count"`
	Error      string    `json:"error"`
}

// Metadata é o estado da busca de metadados.
type Metadata struct {
	Enabled bool   `json:"enabled"`
	TMDB    bool   `json:"tmdb"`
	Running bool   `json:"running"`
	Pending int    `json:"pending"`
	Error   string `json:"error"`
}

// Session é um aparelho usando o servidor.
type Session struct {
	User     string    `json:"user"`
	Client   string    `json:"client"`
	Device   string    `json:"device"`
	LastSeen time.Time `json:"lastSeen"`
}

// Status é o /status (admin).
type Status struct {
	Version   string    `json:"version"`
	Writable  bool      `json:"writable"`
	StartedAt time.Time `json:"startedAt"`
	Libraries []Library `json:"libraries"`
	Scan      Scan      `json:"scan"`
	Metadata  Metadata  `json:"metadata"`
	Sessions  []Session `json:"sessions"`
}

// Platform é um console da biblioteca de jogos do servidor.
type Platform struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Item é um item de lista.
type Item struct {
	ID         string `json:"id"`
	RID        int    `json:"rid"`
	Kind       string `json:"kind"` // movie, series, episode, rom, album, track
	Name       string `json:"name"`
	Year       int    `json:"year"`
	Platform   string `json:"platform,omitempty"`
	SeriesName string `json:"seriesName,omitempty"`
	Image      string `json:"image"`
}

// List é uma lista do usuário (Items só no detalhe).
type List struct {
	ID        string    `json:"id"`
	RID       int       `json:"rid"`
	Name      string    `json:"name"`
	Count     int       `json:"count"`
	Kinds     []string  `json:"kinds"`
	UpdatedAt time.Time `json:"updatedAt"`
	Cover     string    `json:"cover"`
	Items     []Item    `json:"items,omitempty"`
}

// Client chama a API com o token do app.
type Client struct {
	Base  string
	Token string
	// Jellyfin: o token é do Jellyfin (X-Emby-Token); senão, Bearer (RomM).
	Jellyfin bool
	HTTP     *http.Client
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 0} // envio grande: o prazo vem do ctx
}

// GetInfo lê o /info público (sem login).
func GetInfo(ctx context.Context, hc *http.Client, base string) (Info, error) {
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+prefix+"/info", nil)
	if err != nil {
		return Info{}, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("sem API do arkflix (%d)", resp.StatusCode)
	}
	var info Info
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&info); err != nil {
		return Info{}, err
	}
	if info.Product != Product {
		return Info{}, errors.New("não é um arkflix-server")
	}
	return info, nil
}

func (c *Client) auth(req *http.Request) {
	if c.Jellyfin {
		req.Header.Set("X-Emby-Token", c.Token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *Client) send(ctx context.Context, method, p string, body io.Reader, ctype string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.Base, "/")+prefix+p, body)
	if err != nil {
		return err
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	req.Header.Set("Accept", "application/json")
	c.auth(req)
	resp, err := c.http().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("servidor fora do ar? %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		json.Unmarshal(raw, &e)
		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return ErrAuth
		case resp.StatusCode == http.StatusForbidden && e.Error == "":
			return ErrForbidden
		case e.Error == "":
			e.Error = fmt.Sprintf("o servidor respondeu %d", resp.StatusCode)
		}
		return &Error{Status: resp.StatusCode, Message: e.Error}
	}
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) call(ctx context.Context, method, p string, in, out any) error {
	var body io.Reader
	ctype := ""
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body, ctype = bytes.NewReader(raw), "application/json"
	}
	return c.send(ctx, method, p, body, ctype, out)
}

func esc(id string) string { return url.PathEscape(id) }

// Me é o usuário dono do token.
func (c *Client) Me(ctx context.Context) (User, error) {
	var u User
	return u, c.call(ctx, http.MethodGet, "/me", nil, &u)
}

// Status é o estado do servidor (admin).
func (c *Client) Status(ctx context.Context) (Status, error) {
	var st Status
	return st, c.call(ctx, http.MethodGet, "/status", nil, &st)
}

// Scan dispara a varredura em segundo plano (admin).
func (c *Client) Scan(ctx context.Context) error {
	return c.call(ctx, http.MethodPost, "/scan", nil, nil)
}

// Platforms são os consoles da biblioteca de jogos (admin). Servidor sem a
// rota: erro com Status 404.
func (c *Client) Platforms(ctx context.Context) ([]Platform, error) {
	var out []Platform
	return out, c.call(ctx, http.MethodGet, "/platforms", nil, &out)
}

// Users lista os usuários (admin).
func (c *Client) Users(ctx context.Context) ([]User, error) {
	var out []User
	return out, c.call(ctx, http.MethodGet, "/users", nil, &out)
}

// CreateUser cria um usuário (admin).
func (c *Client) CreateUser(ctx context.Context, name, password string, admin bool) (User, error) {
	var u User
	return u, c.call(ctx, http.MethodPost, "/users", map[string]any{"name": name, "password": password, "admin": admin}, &u)
}

// UserPatch muda a senha e/ou o admin (nil = não mexe).
type UserPatch struct {
	Password *string `json:"password,omitempty"`
	Admin    *bool   `json:"admin,omitempty"`
}

// UpdateUser muda um usuário (admin).
func (c *Client) UpdateUser(ctx context.Context, id string, p UserPatch) (User, error) {
	var u User
	return u, c.call(ctx, http.MethodPatch, "/users/"+esc(id), p, &u)
}

// DeleteUser apaga um usuário (admin).
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodDelete, "/users/"+esc(id), nil, nil)
}

// Hide tira o item do acervo (admin, servidor gravável).
func (c *Client) Hide(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/items/"+esc(id)+"/hide", nil, nil)
}

// Upload envia arquivos pro servidor (admin, servidor gravável). fields são
// os campos de destino (kind, title, series...). progress recebe os bytes
// enviados e o total.
func (c *Client) Upload(ctx context.Context, fields map[string]string, files []string, progress func(done, total int64)) ([]string, error) {
	var total int64
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s não é um arquivo", filepath.Base(f))
		}
		total += info.Size()
	}
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)
	go func() {
		pw.CloseWithError(writeParts(w, fields, files, &counter{total: total, fn: progress}))
	}()
	var out struct {
		Saved []string `json:"saved"`
	}
	err := c.send(ctx, http.MethodPost, "/upload", pr, w.FormDataContentType(), &out)
	pr.CloseWithError(errors.New("envio terminou"))
	return out.Saved, err
}

func writeParts(w *multipart.Writer, fields map[string]string, files []string, cnt *counter) error {
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return err
		}
	}
	for _, f := range files {
		in, err := os.Open(f)
		if err != nil {
			return err
		}
		fw, err := w.CreateFormFile("file", filepath.Base(f))
		if err == nil {
			_, err = io.Copy(io.MultiWriter(fw, cnt), in)
		}
		in.Close()
		if err != nil {
			return err
		}
	}
	return w.Close()
}

type counter struct {
	done, total int64
	fn          func(done, total int64)
}

func (c *counter) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	if c.fn != nil {
		c.fn(c.done, c.total)
	}
	return len(p), nil
}

// Lists são as listas do usuário (sem os itens).
func (c *Client) Lists(ctx context.Context) ([]List, error) {
	var out []List
	return out, c.call(ctx, http.MethodGet, "/lists", nil, &out)
}

// List é uma lista com os itens.
func (c *Client) List(ctx context.Context, id string) (List, error) {
	var l List
	return l, c.call(ctx, http.MethodGet, "/lists/"+esc(id), nil, &l)
}

// CreateList cria uma lista vazia.
func (c *Client) CreateList(ctx context.Context, name string) (List, error) {
	var l List
	return l, c.call(ctx, http.MethodPost, "/lists", map[string]string{"name": name}, &l)
}

// RenameList troca o nome.
func (c *Client) RenameList(ctx context.Context, id, name string) (List, error) {
	var l List
	return l, c.call(ctx, http.MethodPatch, "/lists/"+esc(id), map[string]string{"name": name}, &l)
}

// DeleteList apaga a lista (os itens continuam no servidor).
func (c *Client) DeleteList(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodDelete, "/lists/"+esc(id), nil, nil)
}

// AddToList põe itens ("32hex" ou "rom:<rid>"); repetido é ignorado.
func (c *Client) AddToList(ctx context.Context, id string, items ...string) (List, error) {
	var l List
	return l, c.call(ctx, http.MethodPost, "/lists/"+esc(id)+"/items", map[string][]string{"ids": items}, &l)
}

// RemoveFromList tira um item da lista.
func (c *Client) RemoveFromList(ctx context.Context, id, item string) error {
	return c.call(ctx, http.MethodDelete, "/lists/"+esc(id)+"/items/"+esc(item), nil, nil)
}
