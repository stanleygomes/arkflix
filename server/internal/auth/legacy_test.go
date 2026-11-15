package auth_test

import (
	"context"
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// jellyfinHash monta o hash como o Jellyfin grava (hex maiúsculo).
func jellyfinHash(t *testing.T, pass string, sha512Alg bool) string {
	t.Helper()
	salt := []byte("0123456789abcdef")
	id, fn, n := "PBKDF2-SHA512", sha512.New, 64
	if !sha512Alg {
		key, err := pbkdf2.Key(sha1.New, pass, salt, 1000, 32)
		if err != nil {
			t.Fatal(err)
		}
		return "$PBKDF2$iterations=1000$" + strings.ToUpper(hex.EncodeToString(salt)) + "$" + strings.ToUpper(hex.EncodeToString(key))
	}
	key, err := pbkdf2.Key(fn, pass, salt, 1000, n)
	if err != nil {
		t.Fatal(err)
	}
	return "$" + id + "$iterations=1000$" + strings.ToUpper(hex.EncodeToString(salt)) + "$" + strings.ToUpper(hex.EncodeToString(key))
}

func newAuth(t *testing.T) (*auth.Auth, *store.Store) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return auth.NewWithKey(st, make([]byte, 32)), st
}

func subsonicOK(a *auth.Auth, name, pass string) bool {
	sum := md5.Sum([]byte(pass + "sal"))
	_, err := a.SubsonicToken(context.Background(), name, hex.EncodeToString(sum[:]), "sal")
	return err == nil
}

func TestLegacyLoginRehash(t *testing.T) {
	ctx := context.Background()
	for _, sha512Alg := range []bool{true, false} {
		a, st := newAuth(t)
		res, err := a.ImportUser(ctx, auth.Imported{Name: "Ana", Admin: true, Hash: jellyfinHash(t, "velha", sha512Alg)})
		if err != nil || !res.Created || !res.PassSet || !res.User.Admin {
			t.Fatalf("%+v %v", res, err)
		}
		if subsonicOK(a, "ana", "velha") {
			t.Fatal("token do Subsonic antes do 1º login não tem como conferir")
		}
		if _, err := a.Login(ctx, "ana", "errada"); err != auth.ErrBadLogin {
			t.Fatal(err)
		}
		if u, err := a.Login(ctx, "ANA", "velha"); err != nil || u.ID != res.User.ID {
			t.Fatalf("login com a senha antiga: %v", err)
		}
		var hash, legacy string
		st.DB.QueryRow(`SELECT pass_hash, COALESCE(legacy_hash,'') FROM users`).Scan(&hash, &legacy)
		if !strings.HasPrefix(hash, "$2") || legacy != "" {
			t.Fatalf("não regravou em bcrypt: %q %q", hash, legacy)
		}
		if !subsonicOK(a, "ana", "velha") {
			t.Fatal("depois do login o token do Subsonic tem que valer")
		}
		if _, err := a.Login(ctx, "ana", "velha"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestImportBcryptAndMerge(t *testing.T) {
	ctx := context.Background()
	a, st := newAuth(t)
	h, _ := bcrypt.GenerateFromPassword([]byte("romm"), bcrypt.MinCost)
	res, err := a.ImportUser(ctx, auth.Imported{Name: "bia", Hash: string(h)})
	if err != nil || !res.Created {
		t.Fatal(err)
	}
	// outra fonte com outra senha: não troca sem Overwrite
	res, err = a.ImportUser(ctx, auth.Imported{Name: "BIA", Admin: true, Hash: jellyfinHash(t, "jelly", true)})
	if err != nil || res.Created || !res.PassKept || !res.User.Admin {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := a.Login(ctx, "bia", "jelly"); err == nil {
		t.Fatal("senha da 2ª fonte não devia valer")
	}
	// mesma senha em texto puro: ganha a cifrada sem trocar nada
	res, err = a.ImportUser(ctx, auth.Imported{Name: "bia", Password: "romm"})
	if err != nil || !res.PassSet {
		t.Fatalf("%+v %v", res, err)
	}
	if !subsonicOK(a, "bia", "romm") {
		t.Fatal("cifrada não gravada")
	}
	// Overwrite troca
	if _, err := a.ImportUser(ctx, auth.Imported{Name: "bia", Hash: jellyfinHash(t, "jelly", true), Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Login(ctx, "bia", "romm"); err == nil {
		t.Fatal("senha antiga devia ter saído")
	}
	if _, err := a.Login(ctx, "bia", "jelly"); err != nil {
		t.Fatal(err)
	}
	var n int
	st.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	if n != 1 {
		t.Fatalf("%d usuários", n)
	}
}

func TestImportBcryptLoginFillsEnc(t *testing.T) {
	ctx := context.Background()
	a, _ := newAuth(t)
	h, _ := bcrypt.GenerateFromPassword([]byte("romm"), bcrypt.MinCost)
	if _, err := a.ImportUser(ctx, auth.Imported{Name: "caio", Hash: string(h)}); err != nil {
		t.Fatal(err)
	}
	if subsonicOK(a, "caio", "romm") {
		t.Fatal("sem cifrada ainda")
	}
	if _, err := a.SubsonicPassword(ctx, "caio", "romm"); err != nil {
		t.Fatal(err)
	}
	if !subsonicOK(a, "caio", "romm") {
		t.Fatal("login com p= devia gravar a cifrada")
	}
}

func TestImportNoPasswordAndBadHash(t *testing.T) {
	ctx := context.Background()
	a, _ := newAuth(t)
	res, err := a.ImportUser(ctx, auth.Imported{Name: "dani"})
	if err != nil || !res.NoPassword {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := a.Login(ctx, "dani", ""); err != auth.ErrBadLogin {
		t.Fatal("sem senha não entra")
	}
	if ok, _ := a.HasPassword(ctx, res.User.ID); ok {
		t.Fatal("HasPassword")
	}
	// fonte seguinte preenche quem está sem senha, mesmo sem Overwrite
	if res, err = a.ImportUser(ctx, auth.Imported{Name: "dani", Password: "nova"}); err != nil || !res.PassSet {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := a.Login(ctx, "dani", "nova"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ImportUser(ctx, auth.Imported{Name: "eva", Hash: "md5:abc"}); err != auth.ErrUnknownHash {
		t.Fatal(err)
	}
	for _, h := range []string{"$PBKDF2-SHA512$iterations=x$00$00", "$PBKDF2-SHA512$iterations=1$zz$00", "$OUTRO$iterations=1$00$00"} {
		if res, err := a.ImportUser(ctx, auth.Imported{Name: "fe", Hash: h, Overwrite: true}); err == nil {
			if _, err := a.Login(ctx, "fe", ""); err == nil {
				t.Fatalf("hash quebrado %q entrou (%+v)", h, res)
			}
		}
	}
}
