package romm_test

// Este teste repete exatamente as chamadas do cliente RomM do app arkflix
// (app/internal/romm/romm.go) e decodifica nos mesmos structs (mesmas tags
// JSON). Se mudar lá, mude aqui.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"
)

// cópia de apiRom do app
type appRom struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FsName        string `json:"fs_name"`
	Platform      string `json:"platform_fs_slug"`
	Size          int64  `json:"fs_size_bytes"`
	Multi         bool   `json:"has_multiple_files"`
	Summary       string `json:"summary"`
	Cover         string `json:"path_cover_large"`
	MissingFromFS bool   `json:"missing_from_fs"`
	Metadatum     struct {
		Genres  []string `json:"genres"`
		Release int64    `json:"first_release_date"` // ms
		Players string   `json:"player_count"`
	} `json:"metadatum"`
	Files []struct {
		ID   int    `json:"id"`
		Name string `json:"file_name"`
		Path string `json:"file_path"`
		Size int64  `json:"file_size_bytes"`
	} `json:"files"`
}

func TestArkflixClient(t *testing.T) {
	e := newEnv(t)
	scopes := []string{"me.read", "roms.read", "platforms.read", "assets.read", "assets.write", "firmware.read"}

	// Login: senha → access token com me.write
	form := url.Values{"grant_type": {"password"}, "username": {"stanley"}, "password": {"errada"},
		"scope": {strings.Join(append([]string{"me.write"}, scopes...), " ")}}
	r := e.do(http.MethodPost, "/api/token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if r.status != http.StatusUnauthorized && r.status != http.StatusForbidden {
		t.Fatalf("senha errada: %d", r.status)
	}
	form.Set("password", "senha123")
	r = e.do(http.MethodPost, "/api/token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	var tok struct {
		Access string `json:"access_token"`
	}
	r.json(t, &tok)
	if r.status >= 300 || tok.Access == "" {
		t.Fatalf("token: %d %s", r.status, r.body)
	}
	// ... e troca por um token de dispositivo
	body, _ := json.Marshal(map[string]any{"name": "pc de teste", "scopes": scopes})
	r = e.do(http.MethodPost, "/api/client-tokens", "Bearer "+tok.Access, bytes.NewReader(body), "application/json")
	var ct struct {
		ID  int    `json:"id"`
		Raw string `json:"raw_token"`
	}
	r.json(t, &ct)
	if r.status >= 300 || ct.Raw == "" || ct.ID == 0 {
		t.Fatalf("client token: %d %s", r.status, r.body)
	}
	auth := "Bearer " + ct.Raw

	// Me
	var me struct {
		Username string `json:"username"`
	}
	e.get("/api/users/me", auth).json(t, &me)
	if me.Username != "stanley" {
		t.Fatalf("me: %+v", me)
	}

	// Roms (paginado de 200)
	var all []appRom
	for offset := 0; ; {
		var page struct {
			Items []appRom `json:"items"`
			Total int      `json:"total"`
		}
		r := e.get(fmt.Sprintf("/api/roms?limit=200&offset=%d&with_char_index=false&with_filter_values=false", offset), auth)
		if r.status >= 300 {
			t.Fatalf("roms: %d %s", r.status, r.body)
		}
		r.json(t, &page)
		for _, it := range page.Items {
			if !it.MissingFromFS {
				all = append(all, it)
			}
		}
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}
	if len(all) != 3 {
		t.Fatalf("roms: %+v", all)
	}
	var single, multi appRom
	for _, it := range all {
		switch it.FsName {
		case "Super Teste (USA).sfc":
			single = it
		case "Jogo Disco (USA)":
			multi = it
		}
	}
	if single.Platform != "snes" || single.Multi || !multi.Multi || multi.Platform != "psx" || single.Size == 0 {
		t.Fatalf("single/multi: %+v %+v", single, multi)
	}

	// Download do jogo de um arquivo (com retomada por Range)
	p := fmt.Sprintf("/api/roms/%d/content/%s", single.ID, url.PathEscape(single.FsName))
	r = e.get(p, auth)
	if r.status != http.StatusOK || int64(len(r.body)) != single.Size {
		t.Fatalf("download: %d %d", r.status, len(r.body))
	}
	r = e.do(http.MethodGet, p, auth, nil, "", "Range", "bytes=4-")
	if r.status != http.StatusPartialContent || int64(len(r.body)) != single.Size-4 {
		t.Fatalf("retomar: %d", r.status)
	}

	// Rom multi: arquivos com caminho dentro de roms/<console>/<fs_name>
	var full appRom
	e.get(fmt.Sprintf("/api/roms/%d", multi.ID), auth).json(t, &full)
	if len(full.Files) != 4 {
		t.Fatalf("arquivos: %+v", full)
	}
	prefix := path.Join("roms", full.Platform, full.FsName)
	var total int64
	for _, f := range full.Files {
		rel, inside := strings.CutPrefix(path.Clean(f.Path), prefix)
		if !inside || (rel != "" && !strings.HasPrefix(rel, "/")) || f.Name == "" || strings.ContainsAny(f.Name, `/\`) {
			t.Fatalf("caminho fora do jogo: %+v", f)
		}
		r := e.get(fmt.Sprintf("/api/roms/%d/files/content/%s", f.ID, url.PathEscape(f.Name)), auth)
		if r.status != http.StatusOK || int64(len(r.body)) != f.Size {
			t.Fatalf("arquivo %s: %d", f.Name, r.status)
		}
		total += f.Size
	}
	if total != multi.Size {
		t.Fatalf("soma dos arquivos %d != fs_size_bytes %d", total, multi.Size)
	}

	// Saves: upload (overwrite=true), lista, download; states vazio
	upload := func(kind, field, name, data string) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile(field, name)
		fw.Write([]byte(data))
		w.Close()
		q := url.Values{"rom_id": {fmt.Sprint(single.ID)}, "emulator": {"snes9x"}}
		if kind == "saves" {
			q.Set("overwrite", "true")
		}
		r := e.do(http.MethodPost, fmt.Sprintf("/api/%s?%s", kind, q.Encode()), auth, &buf, w.FormDataContentType())
		if r.status >= 300 {
			t.Fatalf("upload %s: %d %s", kind, r.status, r.body)
		}
	}
	type appAsset struct {
		ID       int       `json:"id"`
		FileName string    `json:"file_name"`
		Emulator string    `json:"emulator"`
		Size     int64     `json:"file_size_bytes"`
		Updated  time.Time `json:"updated_at"`
	}
	list := func(kind string) []appAsset {
		var raw []appAsset
		r := e.get(fmt.Sprintf("/api/%s?rom_id=%d", kind, single.ID), auth)
		if r.status >= 300 {
			t.Fatalf("lista %s: %d", kind, r.status)
		}
		r.json(t, &raw)
		return raw
	}
	upload("saves", "saveFile", "Super Teste (USA).srm", "save 1")
	upload("saves", "saveFile", "Super Teste (USA).srm", "save 2, maior")
	saves := list("saves")
	if len(saves) != 1 || saves[0].Size != int64(len("save 2, maior")) || saves[0].Updated.IsZero() || saves[0].Emulator != "snes9x" {
		t.Fatalf("saves: %+v", saves)
	}
	if r := e.get(fmt.Sprintf("/api/saves/%d/content", saves[0].ID), auth); string(r.body) != "save 2, maior" {
		t.Fatalf("baixar save: %q", r.body)
	}
	if states := list("states"); len(states) != 0 {
		t.Fatalf("states: %+v", states)
	}
	upload("states", "stateFile", "Super Teste (USA).state1", "state")
	if states := list("states"); len(states) != 1 || states[0].FileName != "Super Teste (USA).state1" {
		t.Fatalf("states: %+v", states)
	}

	// Firmwares: console de cada BIOS pelo /api/platforms
	var plats []struct {
		ID     int    `json:"id"`
		FsSlug string `json:"fs_slug"`
	}
	e.get("/api/platforms", auth).json(t, &plats)
	slug := map[int]string{}
	for _, p := range plats {
		slug[p.ID] = p.FsSlug
	}
	var fws []struct {
		ID         int    `json:"id"`
		PlatformID int    `json:"platform_id"`
		FileName   string `json:"file_name"`
		Size       int64  `json:"file_size_bytes"`
		MD5        string `json:"md5_hash"`
	}
	e.get("/api/firmware", auth).json(t, &fws)
	if len(fws) != 1 || slug[fws[0].PlatformID] != "psx" {
		t.Fatalf("firmware: %+v %v", fws, slug)
	}
	r = e.get(fmt.Sprintf("/api/firmware/%d/content/%s", fws[0].ID, url.PathEscape(fws[0].FileName)), auth)
	if r.status != http.StatusOK || int64(len(r.body)) != fws[0].Size {
		t.Fatalf("baixar bios: %d", r.status)
	}

	// token apagado (o app revoga ao desconectar)
	if r := e.do(http.MethodDelete, fmt.Sprintf("/api/client-tokens/%d", ct.ID), auth, nil, ""); r.status >= 300 {
		t.Fatalf("revogar: %d", r.status)
	}
	if r := e.get("/api/users/me", auth); r.status != http.StatusUnauthorized && r.status != http.StatusForbidden {
		t.Fatalf("token revogado: %d", r.status)
	}
}
