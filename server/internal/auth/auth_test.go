package auth_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

func TestLoginAndTokens(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a := auth.NewWithKey(st, make([]byte, 32))
	ctx := context.Background()
	u, err := a.CreateUser(ctx, "Stanley", "senha123", true)
	if err != nil || !u.Admin || len(u.ID) != 32 || u.RID == 0 {
		t.Fatalf("%+v %v", u, err)
	}
	if _, err := a.CreateUser(ctx, "stanley", "x", false); err != auth.ErrExists {
		t.Fatalf("nome repetido (sem caixa): %v", err)
	}
	if _, err := a.Login(ctx, "stanley", "errada"); err != auth.ErrBadLogin {
		t.Fatal(err)
	}
	if _, err := a.Login(ctx, "ninguem", "x"); err != auth.ErrBadLogin {
		t.Fatal(err)
	}
	if got, err := a.Login(ctx, "STANLEY", "senha123"); err != nil || got.ID != u.ID {
		t.Fatalf("%+v %v", got, err)
	}
	var raw string
	st.DB.QueryRow(`SELECT pass_hash || pass_enc FROM users`).Scan(&raw)
	if strings.Contains(raw, "senha123") {
		t.Fatal("senha em texto puro no banco")
	}

	sum := md5.Sum([]byte("senha123" + "sal"))
	if got, err := a.SubsonicToken(ctx, "stanley", hex.EncodeToString(sum[:]), "sal"); err != nil || got.ID != u.ID {
		t.Fatalf("token subsonic: %v", err)
	}
	if _, err := a.SubsonicToken(ctx, "stanley", hex.EncodeToString(sum[:]), "outro"); err == nil {
		t.Fatal("sal errado passou")
	}
	if _, err := a.SubsonicPassword(ctx, "stanley", "enc:"+hex.EncodeToString([]byte("senha123"))); err != nil {
		t.Fatal(err)
	}

	tok, err := a.NewToken(ctx, auth.Token{UserID: u.ID, Kind: auth.KindJellyfin, DeviceID: "d1"}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, _, err := a.CheckToken(ctx, tok.Token, auth.KindJellyfin); err != nil || got.ID != u.ID {
		t.Fatal(err)
	}
	if _, _, err := a.CheckToken(ctx, tok.Token, auth.KindRomM); err == nil {
		t.Fatal("token de outro tipo passou")
	}
	short, _ := a.NewToken(ctx, auth.Token{UserID: u.ID, Kind: auth.KindRomM}, "", time.Nanosecond)
	time.Sleep(time.Millisecond)
	if _, _, err := a.CheckToken(ctx, short.Token); err == nil {
		t.Fatal("token vencido passou")
	}
	a.RevokeDevice(ctx, u.ID, auth.KindJellyfin, "d1")
	if _, _, err := a.CheckToken(ctx, tok.Token); err == nil {
		t.Fatal("token revogado passou")
	}
	a.DeleteUser(ctx, u.ID)
	if _, err := a.Login(ctx, "stanley", "senha123"); err == nil {
		t.Fatal("usuário apagado")
	}
}
