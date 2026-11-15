package romm_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/lists"
)

type collT struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	RomIDs          []int64  `json:"rom_ids"`
	RomCount        int      `json:"rom_count"`
	PathCoverSmall  *string  `json:"path_cover_small"`
	PathCoversSmall []string `json:"path_covers_small"`
	URLCover        *string  `json:"url_cover"`
	IsPublic        bool     `json:"is_public"`
	IsVirtual       bool     `json:"is_virtual"`
	IsSmart         bool     `json:"is_smart"`
	UserID          int64    `json:"user_id"`
	OwnerUsername   string   `json:"owner_username"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// form manda um multipart só com campos (como o front do RomM).
func (e *env) form(method, path, auth string, kv ...string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for i := 0; i+1 < len(kv); i += 2 {
		w.WriteField(kv[i], kv[i+1])
	}
	w.Close()
	return e.do(method, path, auth, &buf, w.FormDataContentType())
}

func ids(v ...int64) string {
	var s []string
	for _, n := range v {
		s = append(s, strconv.FormatInt(n, 10))
	}
	return "[" + strings.Join(s, ",") + "]"
}

func TestCollections(t *testing.T) {
	e := newEnv(t)
	admin, nina := e.admin(), e.nina()
	super, outro, disco := e.rid(e.ids.RomSuper), e.rid(e.ids.RomOutro), e.rid(e.ids.RomDisco)

	var c collT
	r := e.form(http.MethodPost, "/api/collections?is_public=false", admin, "name", "Favoritos", "description", "os bons")
	r.json(t, &c)
	if r.status != http.StatusOK || c.ID == 0 || c.Name != "Favoritos" || c.Description != "os bons" || c.RomCount != 0 ||
		c.RomIDs == nil || c.PathCoverSmall != nil || c.URLCover != nil || c.IsPublic || c.IsVirtual || c.IsSmart ||
		c.OwnerUsername != "stanley" || c.UserID == 0 || !isoRe.MatchString(c.CreatedAt) {
		t.Fatalf("criar: %d %s", r.status, r.body)
	}
	id := strconv.FormatInt(c.ID, 10)
	if r := e.form(http.MethodPost, "/api/collections", admin, "name", "favoritos"); r.status != http.StatusConflict {
		t.Fatalf("nome repetido: %d", r.status)
	}
	if r := e.form(http.MethodPost, "/api/collections", admin, "description", "sem nome"); r.status != http.StatusBadRequest {
		t.Fatalf("sem nome: %d", r.status)
	}

	// PUT troca as ROMs (rom_ids em JSON) e o nome
	r = e.form(http.MethodPut, "/api/collections/"+id, admin, "rom_ids", ids(disco, super), "name", "Jogos bons")
	r.json(t, &c)
	if r.status != http.StatusOK || c.Name != "Jogos bons" || c.Description != "os bons" || c.RomCount != 2 || ids(c.RomIDs...) != ids(disco, super) {
		t.Fatalf("put: %d %s", r.status, r.body)
	}
	if r := e.form(http.MethodPut, "/api/collections/"+id, admin, "rom_ids", "nada"); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("rom_ids ruim: %d", r.status)
	}
	// POST/DELETE .../roms
	r = e.postJSON("/api/collections/"+id+"/roms", admin, map[string]any{"rom_ids": []int64{outro, super}})
	r.json(t, &c)
	if ids(c.RomIDs...) != ids(disco, super, outro) {
		t.Fatalf("pôr roms: %s", r.body)
	}
	r = e.do(http.MethodDelete, "/api/collections/"+id+"/roms", admin, strings.NewReader(`{"rom_ids":[`+strconv.FormatInt(disco, 10)+`]}`), "application/json")
	r.json(t, &c)
	if ids(c.RomIDs...) != ids(super, outro) {
		t.Fatalf("tirar roms: %s", r.body)
	}

	// lista, ids, uma só, filtro de ROMs
	var all []collT
	e.get("/api/collections", admin).json(t, &all)
	if len(all) != 1 || all[0].ID != c.ID || all[0].RomCount != 2 {
		t.Fatalf("lista: %+v", all)
	}
	if r := e.get("/api/collections/identifiers", admin); strings.TrimSpace(string(r.body)) != "["+id+"]" {
		t.Fatalf("ids: %s", r.body)
	}
	e.get("/api/collections/"+id, admin).json(t, &c)
	if c.RomCount != 2 {
		t.Fatalf("uma: %+v", c)
	}
	var roms struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	e.get("/api/roms?collection_id="+id, admin).json(t, &roms)
	if roms.Total != 2 {
		t.Fatalf("roms da coleção: %+v", roms)
	}
	if r := e.get("/api/roms?collection_id=x", admin); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("collection_id ruim: %d", r.status)
	}

	// a nina: não vê a privada, nem filtra por ela, nem mexe
	if r := e.get("/api/collections/"+id, nina); r.status != http.StatusForbidden {
		t.Fatalf("nina vê: %d", r.status)
	}
	e.get("/api/roms?collection_id="+id, nina).json(t, &roms)
	if roms.Total != 0 {
		t.Fatalf("nina filtra: %+v", roms)
	}
	e.get("/api/collections", nina).json(t, &all)
	if len(all) != 0 {
		t.Fatalf("nina lista: %+v", all)
	}
	// pública: vê, mas não mexe
	e.form(http.MethodPut, "/api/collections/"+id+"?is_public=true", admin, "rom_ids", ids(super, outro))
	e.get("/api/collections", nina).json(t, &all)
	if len(all) != 1 || !all[0].IsPublic {
		t.Fatalf("pública: %+v", all)
	}
	if r := e.form(http.MethodPut, "/api/collections/"+id, nina, "rom_ids", "[]"); r.status != http.StatusForbidden {
		t.Fatalf("nina edita: %d", r.status)
	}
	if r := e.do(http.MethodDelete, "/api/collections/"+id, nina, nil, ""); r.status != http.StatusForbidden {
		t.Fatalf("nina apaga: %d", r.status)
	}

	// JSON também vale; uma lista sem ROM (só filme) não é coleção
	r = e.postJSON("/api/collections", nina, map[string]any{"name": "Da Nina", "rom_ids": []int64{super}})
	r.json(t, &c)
	if r.status != http.StatusOK || c.RomCount != 1 {
		t.Fatalf("json: %d %s", r.status, r.body)
	}
	ls := lists.New(e.k.Store)
	l, _ := ls.ByRID(context.Background(), c.ID)
	ls.Add(context.Background(), l.ID, []string{e.ids.MovieBig})
	ls.Remove(context.Background(), l.ID, []string{e.ids.RomSuper})
	e.get("/api/collections", nina).json(t, &all)
	if len(all) != 1 || all[0].Name != "Jogos bons" {
		t.Fatalf("lista de filme virou coleção: %+v", all)
	}

	if r := e.do(http.MethodDelete, "/api/collections/"+id, admin, nil, ""); r.status != http.StatusOK {
		t.Fatalf("apagar: %d", r.status)
	}
	if r := e.get("/api/collections/"+id, admin); r.status != http.StatusNotFound {
		t.Fatalf("apagada: %d", r.status)
	}
	if r := e.get("/api/collections/"+url.PathEscape("x"), admin); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("id ruim: %d", r.status)
	}
}
