// Package auth é o login único das três APIs. A senha fica duas vezes:
// bcrypt (pra conferir login) e cifrada com AES-GCM (o token do Subsonic é
// md5(senha+sal) e só dá pra conferir com a senha em mãos, igual o
// Navidrome faz). A chave mora em <data>/secret.key (0600).
package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Erros de login. ErrBadLogin cobre usuário inexistente e senha errada
// (não conta qual dos dois).
var (
	ErrBadLogin = errors.New("usuário ou senha incorretos")
	ErrBadToken = errors.New("token inválido")
	ErrExists   = errors.New("usuário já existe")
)

var dummy, _ = bcrypt.GenerateFromPassword([]byte("arkflix"), bcrypt.DefaultCost)

// User é um usuário (sem a senha).
type User struct {
	RID       int64
	ID        string
	Name      string
	Admin     bool
	CreatedAt time.Time
	LastLogin time.Time
}

// Token é uma sessão de um cliente.
type Token struct {
	Token    string
	UserID   string
	Kind     string
	Name     string
	Client   string
	Device   string
	DeviceID string
	Version  string
	Created  time.Time
	LastUsed time.Time
	Expires  time.Time // zero = não expira
}

// Tipos de token.
const (
	KindJellyfin    = "jellyfin"
	KindRomM        = "romm"         // access token do OAuth2 do RomM
	KindRomMRefresh = "romm-refresh" // refresh token do RomM
	KindRomMClient  = "client"       // client token do RomM (rmm_...), não expira
	KindSubsonic    = "subsonic"     // apiKey do OpenSubsonic
)

// Auth confere senhas e tokens.
type Auth struct {
	st  *store.Store
	key []byte
}

// New carrega (ou cria) a chave em dataDir/secret.key.
func New(st *store.Store, dataDir string) (*Auth, error) {
	file := filepath.Join(dataDir, "secret.key")
	raw, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		raw = []byte(hex.EncodeToString(random(32)))
		if err := os.WriteFile(file, raw, 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(key) != 32 {
		return nil, errors.New("secret.key inválida")
	}
	return &Auth{st: st, key: key}, nil
}

// NewWithKey é pra testes.
func NewWithKey(st *store.Store, key []byte) *Auth { return &Auth{st: st, key: key} }

func random(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// RandomHex devolve n bytes aleatórios em hex.
func RandomHex(n int) string { return hex.EncodeToString(random(n)) }

func (a *Auth) seal(plain string) (string, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := random(gcm.NonceSize())
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (a *Auth) open(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("senha cifrada curta")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(plain), err
}

const userCols = `rid, id, name, admin, created_at, COALESCE(last_login,'')`

func scanUser(r interface{ Scan(...any) error }) (*User, error) {
	var u User
	var created, last string
	err := r.Scan(&u.RID, &u.ID, &u.Name, &u.Admin, &created, &last)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	if u.CreatedAt.IsZero() {
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", created)
	}
	u.LastLogin, _ = time.Parse(time.RFC3339Nano, last)
	return &u, nil
}

// CreateUser cria um usuário.
func (a *Auth) CreateUser(ctx context.Context, name, pass string, admin bool) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" || pass == "" {
		return nil, errors.New("nome e senha são obrigatórios")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	enc, err := a.seal(pass)
	if err != nil {
		return nil, err
	}
	id := RandomHex(16)
	_, err = a.st.DB.ExecContext(ctx, `INSERT INTO users (id, name, pass_hash, pass_enc, admin, created_at) VALUES (?,?,?,?,?,?)`,
		id, name, string(hash), enc, admin, store.Now())
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, ErrExists
		}
		return nil, err
	}
	return a.User(ctx, id)
}

// SetPassword troca a senha.
func (a *Auth) SetPassword(ctx context.Context, userID, pass string) error {
	if pass == "" {
		return errors.New("senha vazia")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	enc, err := a.seal(pass)
	if err != nil {
		return err
	}
	_, err = a.st.DB.ExecContext(ctx, `UPDATE users SET pass_hash = ?, pass_enc = ? WHERE id = ?`, string(hash), enc, userID)
	return err
}

// User busca pelo id.
func (a *Auth) User(ctx context.Context, id string) (*User, error) {
	return scanUser(a.st.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE id = ?`, store.NormalizeID(id)))
}

// UserByRID busca pelo id numérico.
func (a *Auth) UserByRID(ctx context.Context, rid int64) (*User, error) {
	return scanUser(a.st.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE rid = ?`, rid))
}

// UserByName busca pelo nome (sem diferenciar maiúsculas).
func (a *Auth) UserByName(ctx context.Context, name string) (*User, error) {
	return scanUser(a.st.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE name = ?`, name))
}

// Users lista todos.
func (a *Auth) Users(ctx context.Context) ([]*User, error) {
	rows, err := a.st.DB.QueryContext(ctx, `SELECT `+userCols+` FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// DeleteUser apaga o usuário (tokens e estado vão junto).
func (a *Auth) DeleteUser(ctx context.Context, id string) error {
	_, err := a.st.DB.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

// Login confere nome e senha e marca o último login.
func (a *Auth) Login(ctx context.Context, name, pass string) (*User, error) {
	var id, hash string
	err := a.st.DB.QueryRowContext(ctx, `SELECT id, pass_hash FROM users WHERE name = ?`, name).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		bcrypt.CompareHashAndPassword(dummy, []byte(pass)) // mesmo tempo de quem existe
		return nil, ErrBadLogin
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) != nil {
		return nil, ErrBadLogin
	}
	a.st.DB.ExecContext(ctx, `UPDATE users SET last_login = ? WHERE id = ?`, store.Now(), id)
	return a.User(ctx, id)
}

// SubsonicToken confere o token do Subsonic: t = md5(senha + s).
func (a *Auth) SubsonicToken(ctx context.Context, name, token, salt string) (*User, error) {
	pass, id, err := a.password(ctx, name)
	if err != nil {
		return nil, err
	}
	sum := md5.Sum([]byte(pass + salt))
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(strings.ToLower(token))) != 1 {
		return nil, ErrBadLogin
	}
	return a.User(ctx, id)
}

// SubsonicPassword confere p= do Subsonic (texto puro ou "enc:<hex>").
func (a *Auth) SubsonicPassword(ctx context.Context, name, p string) (*User, error) {
	if rest, ok := strings.CutPrefix(p, "enc:"); ok {
		raw, err := hex.DecodeString(rest)
		if err != nil {
			return nil, ErrBadLogin
		}
		p = string(raw)
	}
	return a.Login(ctx, name, p)
}

func (a *Auth) password(ctx context.Context, name string) (pass, id string, err error) {
	var enc string
	err = a.st.DB.QueryRowContext(ctx, `SELECT id, pass_enc FROM users WHERE name = ?`, name).Scan(&id, &enc)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrBadLogin
	}
	if err != nil {
		return "", "", err
	}
	pass, err = a.open(enc)
	return pass, id, err
}

// NewToken cria um token. ttl 0 = não expira. prefix vai na frente
// (o RomM usa "rmm_" nos client tokens).
func (a *Auth) NewToken(ctx context.Context, t Token, prefix string, ttl time.Duration) (Token, error) {
	t.Token = prefix + RandomHex(24)
	t.Created = time.Now().UTC()
	var exp any
	if ttl > 0 {
		t.Expires = t.Created.Add(ttl)
		exp = t.Expires.Format(time.RFC3339Nano)
	}
	_, err := a.st.DB.ExecContext(ctx, `INSERT INTO tokens (token, user_id, kind, name, client, device, device_id, version, created_at, expires_at)
VALUES (?,?,?,?,?,?,?,?,?,?)`, t.Token, t.UserID, t.Kind, t.Name, t.Client, t.Device, t.DeviceID, t.Version, t.Created.Format(time.RFC3339Nano), exp)
	return t, err
}

const tokenCols = `token, user_id, kind, name, client, device, device_id, version, created_at, COALESCE(last_used,''), COALESCE(expires_at,'')`

func scanToken(r interface{ Scan(...any) error }) (Token, error) {
	var t Token
	var created, used, exp string
	err := r.Scan(&t.Token, &t.UserID, &t.Kind, &t.Name, &t.Client, &t.Device, &t.DeviceID, &t.Version, &created, &used, &exp)
	t.Created, _ = time.Parse(time.RFC3339Nano, created)
	t.LastUsed, _ = time.Parse(time.RFC3339Nano, used)
	t.Expires, _ = time.Parse(time.RFC3339Nano, exp)
	return t, err
}

// CheckToken valida o token (de um dos tipos pedidos) e devolve o dono.
func (a *Auth) CheckToken(ctx context.Context, token string, kinds ...string) (*User, Token, error) {
	if token == "" {
		return nil, Token{}, ErrBadToken
	}
	t, err := scanToken(a.st.DB.QueryRowContext(ctx, `SELECT `+tokenCols+` FROM tokens WHERE token = ?`, token))
	if err != nil {
		return nil, Token{}, ErrBadToken
	}
	ok := len(kinds) == 0
	for _, k := range kinds {
		ok = ok || k == t.Kind
	}
	if !ok || (!t.Expires.IsZero() && time.Now().After(t.Expires)) {
		return nil, Token{}, ErrBadToken
	}
	u, err := a.User(ctx, t.UserID)
	if err != nil {
		return nil, Token{}, ErrBadToken
	}
	// não grava a cada requisição: no máximo 1x por minuto
	if time.Since(t.LastUsed) > time.Minute {
		a.st.DB.ExecContext(ctx, `UPDATE tokens SET last_used = ? WHERE token = ?`, store.Now(), token)
	}
	return u, t, nil
}

// Tokens lista os tokens de um usuário (de um tipo, ou todos com kind "").
func (a *Auth) Tokens(ctx context.Context, userID, kind string) ([]Token, error) {
	rows, err := a.st.DB.QueryContext(ctx, `SELECT `+tokenCols+` FROM tokens WHERE user_id = ? AND (? = '' OR kind = ?) ORDER BY created_at`, userID, kind, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RevokeToken apaga um token.
func (a *Auth) RevokeToken(ctx context.Context, token string) error {
	_, err := a.st.DB.ExecContext(ctx, `DELETE FROM tokens WHERE token = ?`, token)
	return err
}

// RevokeDevice apaga os tokens de um aparelho (o Jellyfin troca o token
// quando o mesmo aparelho loga de novo).
func (a *Auth) RevokeDevice(ctx context.Context, userID, kind, deviceID string) error {
	if deviceID == "" {
		return nil
	}
	_, err := a.st.DB.ExecContext(ctx, `DELETE FROM tokens WHERE user_id = ? AND kind = ? AND device_id = ?`, userID, kind, deviceID)
	return err
}

// PurgeExpired apaga tokens vencidos.
func (a *Auth) PurgeExpired(ctx context.Context) error {
	_, err := a.st.DB.ExecContext(ctx, `DELETE FROM tokens WHERE expires_at IS NOT NULL AND expires_at < ?`, store.Now())
	return err
}
