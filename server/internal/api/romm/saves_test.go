package romm_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type assetJSON struct {
	ID             int64           `json:"id"`
	RomID          int64           `json:"rom_id"`
	UserID         int64           `json:"user_id"`
	FileName       string          `json:"file_name"`
	FileNameNoTags string          `json:"file_name_no_tags"`
	FileNameNoExt  string          `json:"file_name_no_ext"`
	FileExtension  string          `json:"file_extension"`
	FilePath       string          `json:"file_path"`
	FullPath       string          `json:"full_path"`
	DownloadPath   string          `json:"download_path"`
	Size           int64           `json:"file_size_bytes"`
	Emulator       *string         `json:"emulator"`
	MissingFromFS  bool            `json:"missing_from_fs"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
	IsGallery      *bool           `json:"is_gallery"`
	Screenshot     *assetJSON      `json:"screenshot"`
	Labels         []string        `json:"labels"`
	Raw            map[string]bool `json:"-"`
}

func TestAssetsUploadListDownloadReplaceDelete(t *testing.T) {
	for _, tc := range []struct{ kind, field, ext string }{
		{"saves", "saveFile", "srm"}, {"states", "stateFile", "state"}, {"screenshots", "screenshotFile", "png"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			e := newEnv(t)
			admin, nina := e.admin(), e.nina()
			rom := itoa(e.rid(e.ids.RomSuper))
			name := "Super Teste (USA)." + tc.ext
			parts := []part{{tc.field, name, "conteúdo 1"}}
			if tc.kind != "screenshots" {
				parts = append(parts, part{"screenshotFile", "Super Teste (USA).png", "png!"})
			}
			body, ctype := multipartBody(t, parts...)
			r := e.do(http.MethodPost, "/api/"+tc.kind+"?rom_id="+rom+"&emulator=snes9x", admin, body, ctype)
			var a assetJSON
			r.json(t, &a)
			if r.status != http.StatusOK || a.ID == 0 || a.FileName != name || a.Size != int64(len("conteúdo 1")) ||
				a.UserID != e.k.AdminUser.RID || a.RomID != e.rid(e.ids.RomSuper) || a.FileExtension != tc.ext ||
				a.FileNameNoTags != "Super Teste" || a.MissingFromFS || !isoRe.MatchString(a.UpdatedAt) ||
				!strings.HasPrefix(a.DownloadPath, "/api/"+tc.kind+"/"+itoa(a.ID)+"/content?timestamp=") {
				t.Fatalf("upload: %d %s", r.status, r.body)
			}
			if tc.kind == "screenshots" {
				if a.IsGallery == nil || !*a.IsGallery || a.Emulator != nil {
					t.Fatalf("screenshot: %s", r.body)
				}
			} else {
				wantPath := "romm/" + tc.kind + "/" + itoa(e.k.AdminUser.RID) + "/snes/" + rom + "/snes9x"
				if a.Emulator == nil || *a.Emulator != "snes9x" || a.FilePath != wantPath || a.Screenshot == nil ||
					a.Screenshot.FileName != "Super Teste (USA).png" || a.Labels == nil {
					t.Fatalf("save/state: %s", r.body)
				}
				if _, err := os.Stat(filepath.Join(e.k.Config.Data, filepath.FromSlash(wantPath), name)); err != nil {
					t.Fatal(err)
				}
			}
			// mesmo nome (e emulador) substitui
			body, ctype = multipartBody(t, part{tc.field, name, "conteúdo 2, maior"})
			r = e.do(http.MethodPost, "/api/"+tc.kind+"?rom_id="+rom+"&emulator=snes9x", admin, body, ctype)
			var a2 assetJSON
			r.json(t, &a2)
			if a2.ID != a.ID || a2.Size != int64(len("conteúdo 2, maior")) {
				t.Fatalf("substituir: %s", r.body)
			}
			var list []assetJSON
			e.get("/api/"+tc.kind+"?rom_id="+rom, admin).json(t, &list)
			if len(list) != 1 || list[0].ID != a.ID {
				t.Fatalf("lista: %+v", list)
			}
			e.get("/api/"+tc.kind+"?platform_id="+itoa(e.rid(e.ids.PSX)), admin).json(t, &list)
			if len(list) != 0 {
				t.Fatalf("lista de outro console: %+v", list)
			}
			r = e.get("/api/"+tc.kind+"/"+itoa(a.ID)+"/content", admin)
			if r.status != http.StatusOK || string(r.body) != "conteúdo 2, maior" || !strings.Contains(r.header.Get("Content-Disposition"), name) {
				t.Fatalf("baixar: %d %q", r.status, r.body)
			}
			if r := e.get("/api/"+tc.kind+"/"+itoa(a.ID)+"/content/"+name, admin); string(r.body) != "conteúdo 2, maior" {
				t.Fatalf("baixar com nome: %q", r.body)
			}
			if r := e.get("/api/"+tc.kind+"/"+itoa(a.ID), admin); r.status != http.StatusOK {
				t.Fatalf("get: %d", r.status)
			}

			// a nina não enxerga nada do stanley
			e.get("/api/"+tc.kind+"?rom_id="+rom, nina).json(t, &list)
			if len(list) != 0 {
				t.Fatalf("nina listou: %+v", list)
			}
			for _, p := range []string{"", "/content", "/content/" + name} {
				if r := e.get("/api/"+tc.kind+"/"+itoa(a.ID)+p, nina); r.status != http.StatusNotFound {
					t.Fatalf("nina leu %s: %d", p, r.status)
				}
			}
			body, ctype = multipartBody(t, part{tc.field, name, "da nina"})
			if r := e.do(http.MethodPut, "/api/"+tc.kind+"/"+itoa(a.ID), nina, body, ctype); r.status != http.StatusNotFound {
				t.Fatalf("nina trocou: %d", r.status)
			}
			if r := e.postJSON("/api/"+tc.kind+"/delete", nina, map[string]any{tc.kind: []int64{a.ID}}); r.status != http.StatusNotFound {
				t.Fatalf("nina apagou: %d", r.status)
			}
			if r := e.do(http.MethodDelete, "/api/"+tc.kind+"/"+itoa(a.ID), nina, nil, ""); r.status != http.StatusNotFound {
				t.Fatalf("nina apagou (DELETE): %d", r.status)
			}
			// a nina tem o dela, separado
			body, ctype = multipartBody(t, part{tc.field, name, "da nina"})
			if r := e.do(http.MethodPost, "/api/"+tc.kind+"?rom_id="+rom+"&emulator=snes9x", nina, body, ctype); r.status != http.StatusOK {
				t.Fatalf("upload da nina: %d %s", r.status, r.body)
			}
			if r := e.get("/api/"+tc.kind+"/"+itoa(a.ID)+"/content", admin); string(r.body) != "conteúdo 2, maior" {
				t.Fatalf("save do stanley mudou: %q", r.body)
			}

			// PUT troca o conteúdo
			body, ctype = multipartBody(t, part{tc.field, "qualquer.bin", "trocado"})
			r = e.do(http.MethodPut, "/api/"+tc.kind+"/"+itoa(a.ID), admin, body, ctype)
			var a3 assetJSON
			r.json(t, &a3)
			if r.status != http.StatusOK || a3.FileName != name || a3.Size != int64(len("trocado")) {
				t.Fatalf("put: %d %s", r.status, r.body)
			}
			if r := e.get("/api/"+tc.kind+"/"+itoa(a.ID)+"/content", admin); string(r.body) != "trocado" {
				t.Fatalf("depois do put: %q", r.body)
			}

			// no detalhe da ROM
			var rd map[string]any
			e.get("/api/roms/"+rom, admin).json(t, &rd)
			key := map[string]string{"saves": "user_saves", "states": "user_states", "screenshots": "user_screenshots"}[tc.kind]
			if l, _ := rd[key].([]any); len(l) != 1 {
				t.Fatalf("%s no detalhe: %v", key, rd[key])
			}
			if tc.kind != "screenshots" {
				if l, _ := rd["all_"+key].([]any); len(l) != 1 || l[0].(map[string]any)["username"] != "stanley" {
					t.Fatalf("all_%s: %v", key, rd["all_"+key])
				}
			}

			// apagar
			if r := e.postJSON("/api/"+tc.kind+"/delete", admin, map[string]any{tc.kind: []int64{}}); r.status != http.StatusBadRequest {
				t.Fatalf("delete vazio: %d", r.status)
			}
			r = e.postJSON("/api/"+tc.kind+"/delete", admin, map[string]any{tc.kind: []int64{a.ID}})
			if r.status != http.StatusOK || strings.TrimSpace(string(r.body)) != "["+itoa(a.ID)+"]" {
				t.Fatalf("delete: %d %s", r.status, r.body)
			}
			if r := e.get("/api/"+tc.kind+"/"+itoa(a.ID), admin); r.status != http.StatusNotFound {
				t.Fatalf("apagado ainda existe: %d", r.status)
			}
			if _, err := os.Stat(filepath.Join(e.k.Config.Data, filepath.FromSlash(a.FullPath))); !os.IsNotExist(err) {
				t.Fatalf("arquivo ficou: %v", err)
			}
		})
	}
}

func TestUploadRejectsBadNames(t *testing.T) {
	e := newEnv(t)
	tok := e.nina()
	rom := itoa(e.rid(e.ids.RomSuper))
	post := func(query string, p part) (int, string) {
		body, ctype := multipartBody(t, p)
		r := e.do(http.MethodPost, "/api/saves?rom_id="+rom+query, tok, body, ctype)
		return r.status, string(r.body)
	}
	for _, emu := range []string{"..", "../../etc", "a/b", `a\b`} {
		if st, b := post("&emulator="+emu, part{"saveFile", "x.srm", "x"}); st != http.StatusBadRequest {
			t.Fatalf("emulador %q: %d %s", emu, st, b)
		}
	}
	for _, name := range []string{`..\..\evil.srm`, "..", "a\x01b"} {
		if st, b := post("", part{"saveFile", name, "x"}); st != http.StatusBadRequest {
			t.Fatalf("nome %q: %d %s", name, st, b)
		}
	}
	// "../" no nome some (fica só o nome-base, dentro da pasta certa)
	st, b := post("", part{"saveFile", "../../../evil.srm", "x"})
	if st != http.StatusOK || !strings.Contains(b, `"file_name":"evil.srm"`) {
		t.Fatalf("nome com ../: %d %s", st, b)
	}
	var bad []string
	filepath.Walk(e.k.Root, func(p string, info os.FileInfo, err error) error {
		if err == nil && info.Name() == "evil.srm" && !strings.HasPrefix(p, filepath.Join(e.k.Config.Data, "romm", "saves")+string(filepath.Separator)) {
			bad = append(bad, p)
		}
		return nil
	})
	if len(bad) > 0 {
		t.Fatalf("arquivo fora da pasta: %v", bad)
	}
	if st, _ := post("", part{"outroCampo", "x.srm", "x"}); st != http.StatusUnprocessableEntity {
		t.Fatalf("sem saveFile: %d", st)
	}
	body, ctype := multipartBody(t, part{"saveFile", "x.srm", "x"})
	if r := e.do(http.MethodPost, "/api/saves?rom_id=99999", tok, body, ctype); r.status != http.StatusNotFound {
		t.Fatalf("rom inexistente: %d", r.status)
	}
	if r := e.get("/api/saves/abc/content", tok); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("id ruim: %d", r.status)
	}
	// formato antigo: campo "saves" com vários arquivos
	body, ctype = multipartBody(t, part{"saves", "a.srm", "a"}, part{"saves", "b.srm", "b"})
	r := e.do(http.MethodPost, "/api/saves?rom_id="+rom, tok, body, ctype)
	if r.status != http.StatusOK || !strings.Contains(string(r.body), `"uploaded":2`) {
		t.Fatalf("formato antigo: %d %s", r.status, r.body)
	}
}

func TestProps(t *testing.T) {
	e := newEnv(t)
	tok := e.nina()
	ctx := context.Background()
	rom := itoa(e.rid(e.ids.RomOutro))
	r := e.do(http.MethodPut, "/api/roms/"+rom+"/props?update_last_played=true", tok,
		strings.NewReader(`{"rating": 8, "status": "finished", "backlogged": true}`), "application/json")
	var ru struct {
		ID         int64   `json:"id"`
		UserID     int64   `json:"user_id"`
		RomID      int64   `json:"rom_id"`
		Rating     int     `json:"rating"`
		Status     *string `json:"status"`
		Backlogged bool    `json:"backlogged"`
		Hidden     bool    `json:"hidden"`
		LastPlayed *string `json:"last_played"`
	}
	r.json(t, &ru)
	if r.status != http.StatusOK || ru.ID < 1 || ru.Rating != 8 || ru.Status == nil || *ru.Status != "finished" || !ru.Backlogged ||
		ru.LastPlayed == nil || !isoRe.MatchString(*ru.LastPlayed) || ru.UserID != e.k.PlainUser.RID {
		t.Fatalf("props: %d %s", r.status, r.body)
	}
	// last_played é o do userdata comum
	if d, _ := e.k.Store.UserData(ctx, e.k.PlainUser.ID, e.ids.RomOutro); d.LastPlayed.IsZero() {
		t.Fatal("userdata sem last_played")
	}
	// formato antigo {"data": {...}} e só muda o que veio
	r = e.do(http.MethodPut, "/api/roms/"+rom+"/props", tok, strings.NewReader(`{"data": {"hidden": true}}`), "application/json")
	r.json(t, &ru)
	if !ru.Hidden || ru.Rating != 8 || ru.LastPlayed == nil {
		t.Fatalf("parcial: %s", r.body)
	}
	var rd romJSON
	e.get("/api/roms/"+rom, tok).json(t, &rd)
	if rd.RomUser.Rating != 8 || rd.RomUser.Status == nil || rd.RomUser.LastPlayed == nil {
		t.Fatalf("rom_user no detalhe: %+v", rd.RomUser)
	}
	// é por usuário
	e.get("/api/roms/"+rom, e.admin()).json(t, &rd)
	if rd.RomUser.Rating != 0 || rd.RomUser.ID != -1 {
		t.Fatalf("props vazou: %+v", rd.RomUser)
	}
	r = e.do(http.MethodPut, "/api/roms/"+rom+"/props?remove_last_played=true", tok, strings.NewReader(`{}`), "application/json")
	r.json(t, &ru)
	if ru.LastPlayed != nil {
		t.Fatalf("remove_last_played: %s", r.body)
	}
	for _, bad := range []string{`{"rating": 11}`, `{"status": "zerado"}`, `{"completion": 101}`, `nada`} {
		if r := e.do(http.MethodPut, "/api/roms/"+rom+"/props", tok, strings.NewReader(bad), "application/json"); r.status != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d", bad, r.status)
		}
	}
	if r := e.do(http.MethodPut, "/api/roms/"+rom+"/props?update_last_played=true&remove_last_played=true", tok,
		strings.NewReader(`{}`), "application/json"); r.status != http.StatusBadRequest {
		t.Fatalf("os dois: %d", r.status)
	}
}
