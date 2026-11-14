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

	// a nova já existe (o WebKit criou): traz o que falta, nunca sobrescreve
	os.MkdirAll(filepath.Join(old, "romm"), 0o700)
	os.WriteFile(filepath.Join(old, "x"), []byte("velho"), 0o600)
	os.WriteFile(filepath.Join(dir, "x"), []byte("novo"), 0o600)
	if moved, err := settings.MoveLegacy(dir, old); !moved || err != nil {
		t.Fatalf("traz o que falta: %v %v", moved, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "romm")); err != nil {
		t.Fatal("pasta que faltava veio")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "x")); string(b) != "novo" {
		t.Fatalf("não sobrescreve: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(old, "x")); string(b) != "velho" {
		t.Fatal("o que não coube fica na antiga")
	}
	if moved, _ := settings.MoveLegacy(dir, old); moved {
		t.Fatal("de novo: nada a trazer")
	}
	os.Remove(filepath.Join(old, "x"))
	settings.MoveLegacy(dir, old)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("antiga vazia some")
	}
}
