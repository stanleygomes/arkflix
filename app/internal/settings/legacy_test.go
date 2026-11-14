package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix/app/internal/settings"
)

func TestMoveLegacy(t *testing.T) {
	base := t.TempDir()
	dir, old := filepath.Join(base, "arkflix"), filepath.Join(base, "arkflix-games")

	if moved, err := settings.MoveLegacy(dir, old); moved || err != nil {
		t.Fatalf("sem pasta antiga não faz nada: %v %v", moved, err)
	}
	os.MkdirAll(filepath.Join(old, "profiles", "abc"), 0o700)
	os.WriteFile(filepath.Join(old, "profiles.json"), []byte("[]"), 0o600)
	if moved, err := settings.MoveLegacy(dir, old); !moved || err != nil {
		t.Fatalf("move a antiga: %v %v", moved, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles", "abc")); err != nil {
		t.Fatal("perfis vieram junto")
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("a antiga sai do lugar")
	}

	// a nova já existe: nunca mistura nem apaga
	os.MkdirAll(old, 0o700)
	os.WriteFile(filepath.Join(old, "x"), []byte("velho"), 0o600)
	if moved, _ := settings.MoveLegacy(dir, old); moved {
		t.Fatal("não mexe quando a nova já existe")
	}
	if _, err := os.Stat(filepath.Join(old, "x")); err != nil {
		t.Fatal("a antiga fica intacta")
	}
}
