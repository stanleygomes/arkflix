package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha512"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"hash"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// legacy_hash guarda o hash de outro servidor (Jellyfin) até o 1º login.
func init() {
	store.Register(store.Migration{Name: "auth/001-legacy-hash", SQL: `ALTER TABLE users ADD COLUMN legacy_hash TEXT;`})
}

// creds é o que o banco tem pra conferir a senha de um usuário.
type creds struct {
	hash   string // bcrypt ("" = ainda não tem)
	enc    string // AES-GCM ("" = importado sem a senha em mãos)
	legacy string // hash de outro servidor ("" = nenhum)
}

// check confere a senha: bcrypt se tiver, senão o hash legado.
func (c creds) check(pass string) bool {
	switch {
	case c.hash != "":
		return bcrypt.CompareHashAndPassword([]byte(c.hash), []byte(pass)) == nil
	case c.legacy != "":
		return checkLegacy(c.legacy, pass)
	}
	bcrypt.CompareHashAndPassword(dummy, []byte(pass))
	return false
}

func (c creds) empty() bool { return c.hash == "" && c.legacy == "" }

// IsLegacyHash diz se é um hash do Jellyfin ($PBKDF2-SHA512$... ou o antigo
// $PBKDF2$..., SHA1).
func IsLegacyHash(h string) bool {
	return strings.HasPrefix(h, "$PBKDF2-SHA512$") || strings.HasPrefix(h, "$PBKDF2$")
}

// IsBcrypt diz se é um hash bcrypt ($2a$, $2b$, $2y$).
func IsBcrypt(h string) bool {
	_, err := bcrypt.Cost([]byte(h))
	return err == nil
}

// checkLegacy confere o formato do Jellyfin:
// $<id>$iterations=<n>$<sal hex>$<hash hex>.
func checkLegacy(h, pass string) bool {
	parts := strings.Split(h, "$")
	if len(parts) != 5 || parts[0] != "" {
		return false
	}
	var fn func() hash.Hash
	switch parts[1] {
	case "PBKDF2-SHA512":
		fn = sha512.New
	case "PBKDF2":
		fn = sha1.New
	default:
		return false
	}
	iter := 0
	for _, kv := range strings.Split(parts[2], ",") {
		if v, ok := strings.CutPrefix(kv, "iterations="); ok {
			iter, _ = strconv.Atoi(v)
		}
	}
	salt, err1 := hex.DecodeString(parts[3])
	want, err2 := hex.DecodeString(parts[4])
	if err1 != nil || err2 != nil || iter <= 0 || iter > 10_000_000 || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(fn, pass, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// Imported é um usuário vindo de outro servidor. Basta um entre Password e
// Hash; sem nenhum o usuário fica sem conseguir entrar até `user passwd`.
type Imported struct {
	Name      string
	Admin     bool
	Password  string // texto puro (Navidrome): vira bcrypt + cifrada na hora
	Hash      string // bcrypt (RomM) ou legado (Jellyfin): regravado no 1º login
	Overwrite bool   // troca a senha de quem já existe
}

// ErrUnknownHash vem quando o hash não é bcrypt nem PBKDF2 do Jellyfin.
var ErrUnknownHash = errors.New("formato de hash desconhecido")

// ImportResult diz o que ImportUser fez.
type ImportResult struct {
	User       *User
	Created    bool // usuário novo
	PassSet    bool // senha gravada/trocada
	PassKept   bool // já tinha senha e ficou a dela
	NoPassword bool // ficou sem senha nenhuma
}

// ImportUser cria o usuário ou junta com o que já existe (mesmo nome, sem
// diferenciar maiúsculas). Admin só soma: quem é admin em alguma fonte
// fica admin. A senha de quem já existe só muda com Overwrite, se ainda não
// tiver nenhuma, ou se a senha em texto puro confere com o hash guardado
// (aí só ganha a cópia cifrada que faltava).
func (a *Auth) ImportUser(ctx context.Context, in Imported) (ImportResult, error) {
	var res ImportResult
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return res, errors.New("nome vazio")
	}
	if in.Password == "" && in.Hash != "" && !IsBcrypt(in.Hash) && !IsLegacyHash(in.Hash) {
		return res, ErrUnknownHash
	}
	var id string
	var c creds
	var admin bool
	err := a.st.DB.QueryRowContext(ctx, `SELECT id, pass_hash, pass_enc, COALESCE(legacy_hash,''), admin FROM users WHERE name = ?`, name).
		Scan(&id, &c.hash, &c.enc, &c.legacy, &admin)
	if errors.Is(err, sql.ErrNoRows) {
		id = RandomHex(16)
		_, err = a.st.DB.ExecContext(ctx, `INSERT INTO users (id, name, pass_hash, pass_enc, admin, created_at) VALUES (?,?,'','',?,?)`,
			id, name, in.Admin, store.Now())
		if err != nil {
			return res, err
		}
		res.Created = true
		c = creds{}
	} else if err != nil {
		return res, err
	} else if in.Admin && !admin {
		if _, err := a.st.DB.ExecContext(ctx, `UPDATE users SET admin = 1 WHERE id = ?`, id); err != nil {
			return res, err
		}
	}

	switch {
	case in.Password == "" && in.Hash == "":
		res.PassKept = !c.empty()
	case in.Overwrite || c.empty():
		res.PassSet = true
		err = a.setCreds(ctx, id, in)
	case in.Password != "" && c.enc == "" && c.check(in.Password):
		res.PassSet = true
		err = a.SetPassword(ctx, id, in.Password)
	default:
		res.PassKept = true
	}
	if err != nil {
		return res, err
	}
	res.NoPassword = c.empty() && !res.PassSet
	res.User, err = a.User(ctx, id)
	return res, err
}

// setCreds grava a senha importada: texto puro vira bcrypt + cifrada;
// bcrypt fica como está (sem a cifrada); legado vai pra legacy_hash.
func (a *Auth) setCreds(ctx context.Context, id string, in Imported) error {
	if in.Password != "" {
		return a.SetPassword(ctx, id, in.Password)
	}
	hash, legacy := in.Hash, any(nil)
	if IsLegacyHash(in.Hash) {
		hash, legacy = "", in.Hash
	}
	_, err := a.st.DB.ExecContext(ctx, `UPDATE users SET pass_hash = ?, pass_enc = '', legacy_hash = ? WHERE id = ?`, hash, legacy, id)
	return err
}

// HasPassword diz se o usuário tem alguma senha que dê pra conferir.
func (a *Auth) HasPassword(ctx context.Context, id string) (bool, error) {
	var c creds
	err := a.st.DB.QueryRowContext(ctx, `SELECT pass_hash, pass_enc, COALESCE(legacy_hash,'') FROM users WHERE id = ?`, id).
		Scan(&c.hash, &c.enc, &c.legacy)
	return !c.empty(), err
}
