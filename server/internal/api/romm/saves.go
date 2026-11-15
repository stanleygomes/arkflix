package romm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// assetKind é um tipo de arquivo do usuário: saves, states ou screenshots.
type assetKind struct {
	name  string // saves | states | screenshots (rota e coluna kind)
	field string // campo do multipart (saveFile, stateFile, screenshotFile)
	label string // "Save", "State", "Screenshot" (mensagens)
}

var (
	kindSaves       = assetKind{"saves", "saveFile", "Save"}
	kindStates      = assetKind{"states", "stateFile", "State"}
	kindScreenshots = assetKind{"screenshots", "screenshotFile", "Screenshot"}
	kinds           = []assetKind{kindSaves, kindStates, kindScreenshots}
)

// maxUpload é o teto de um upload (state de PS2/Switch passa de 100 MB).
const maxUpload = 1 << 30

func (a *api) assetRoutes(mux *http.ServeMux, k assetKind) {
	read, write := []string{scopeAssetsRead}, []string{scopeAssetsWrite}
	base := "/api/" + k.name
	a.handle(mux, "GET "+base, read, func(w http.ResponseWriter, r *http.Request, c *caller) { a.listAssets(w, r, c, k) })
	a.handle(mux, "POST "+base, write, func(w http.ResponseWriter, r *http.Request, c *caller) { a.uploadAsset(w, r, c, k) })
	a.handle(mux, "GET "+base+"/identifiers", read, func(w http.ResponseWriter, r *http.Request, c *caller) { a.assetIDs(w, r, c, k) })
	a.handle(mux, "POST "+base+"/delete", write, func(w http.ResponseWriter, r *http.Request, c *caller) { a.deleteAssets(w, r, c, k) })
	a.handle(mux, "GET "+base+"/{id}", read, func(w http.ResponseWriter, r *http.Request, c *caller) { a.getAsset(w, r, c, k) })
	a.handle(mux, "PUT "+base+"/{id}", write, func(w http.ResponseWriter, r *http.Request, c *caller) { a.updateAsset(w, r, c, k) })
	a.handle(mux, "DELETE "+base+"/{id}", write, func(w http.ResponseWriter, r *http.Request, c *caller) { a.deleteAsset(w, r, c, k) })
	a.handle(mux, "GET "+base+"/{id}/content", read, func(w http.ResponseWriter, r *http.Request, c *caller) { a.assetContent(w, r, c, k) })
	a.handle(mux, "GET "+base+"/{id}/content/{file_name}", read, func(w http.ResponseWriter, r *http.Request, c *caller) { a.assetContent(w, r, c, k) })
	if k == kindSaves {
		a.handle(mux, "GET "+base+"/summary", read, a.savesSummary)
	}
}

// sanitizeName limpa um nome de arquivo ("" se não sobrar nada usável).
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if strings.ContainsAny(name, "/\\\x00") || name == "." || name == ".." || len(name) > 255 {
		return ""
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return ""
	}
	return name
}

// emulatorOK confere o nome do emulador (vira pasta).
func emulatorOK(e string) bool { return e == "" || sanitizeName(e) == e }

func (a *api) diskPath(s *asset) string {
	return filepath.Join(a.d.Config.Data, filepath.FromSlash(s.FilePath), s.FileName)
}

func stem(name string) string { return strings.TrimSuffix(name, path.Ext(name)) }

// pyTime é o str(datetime) do Python (vai no download_path).
func pyTime(t time.Time) string { return strings.Replace(isoformat(t), "T", " ", 1) }

// assetSchema monta o SaveSchema/StateSchema/ScreenshotSchema.
func (a *api) assetSchema(ctx context.Context, s *asset, userRID int64) (assetJSON, error) {
	_, err := os.Stat(a.diskPath(s))
	j := assetJSON{
		"id": s.ID, "rom_id": s.RomRID, "user_id": userRID, "file_name": s.FileName,
		"file_name_no_tags": nameNoTags(s.FileName), "file_name_no_ext": nameNoExt(s.FileName),
		"file_extension": nameExt(s.FileName), "file_path": s.FilePath, "file_size_bytes": s.Size,
		"full_path": s.FilePath + "/" + s.FileName,
		"download_path": fmt.Sprintf("/api/%s/%d/content?timestamp=%s&name=%s", s.Kind, s.ID,
			url.QueryEscape(pyTime(s.Updated)), url.PathEscape(s.FileName)),
		"missing_from_fs": err != nil, "created_at": ts(s.Created), "updated_at": ts(s.Updated), "is_public": s.IsPublic,
	}
	if s.Kind == kindScreenshots.name {
		j["is_gallery"] = s.IsGallery
		return j, nil
	}
	j["emulator"] = orNil(s.Emulator)
	j["is_favorite"] = false
	j["labels"] = []string{}
	j["channel_id"] = nil
	j["origin_device_id"] = nil
	j["screenshot"] = nil
	shots, err := a.assets(ctx, `kind = 'screenshots' AND user_id = ? AND rom_rid = ? ORDER BY updated_at DESC`, s.UserID, s.RomRID)
	if err != nil {
		return nil, err
	}
	for _, sh := range shots {
		if stem(sh.FileName) == stem(s.FileName) {
			if j["screenshot"], err = a.assetSchema(ctx, sh, userRID); err != nil {
				return nil, err
			}
			break
		}
	}
	if s.Kind == kindSaves.name {
		j["slot"] = nil
		j["content_hash"] = orNil(s.ContentHash)
		j["device_syncs"] = []any{}
	} else {
		j["core"] = nil
	}
	return j, nil
}

// ownAsset busca o asset {id} do usuário (404 se for de outro).
func (a *api) ownAsset(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) (*asset, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return nil, false
	}
	s, err := a.assetByID(r.Context(), k.name, id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && s.UserID != c.user.ID) {
		fail(w, http.StatusNotFound, k.label+" with ID "+r.PathValue("id")+" not found")
		return nil, false
	}
	if err != nil {
		internal(w, err)
		return nil, false
	}
	return s, true
}

func (a *api) listAssets(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	ctx := r.Context()
	where, args := `kind = ? AND user_id = ?`, []any{k.name, c.user.ID}
	romIDs, err := queryInts(r, "rom_id", "rom_ids")
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if len(romIDs) > 0 {
		where += ` AND rom_rid IN (` + placeholders(len(romIDs)) + `)`
		args = append(args, anys(romIDs)...)
	}
	pids, err := queryInts(r, "platform_id")
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if len(pids) > 0 {
		where += ` AND rom_rid IN (SELECT r.rid FROM items r JOIN items p ON p.id = r.parent_id WHERE r.kind = 'rom' AND p.rid IN (` +
			placeholders(len(pids)) + `))`
		args = append(args, anys(pids)...)
	}
	list, err := a.assets(ctx, where+` ORDER BY updated_at DESC, id DESC`, args...)
	if err != nil {
		internal(w, err)
		return
	}
	out := []assetJSON{}
	for _, s := range list {
		j, err := a.assetSchema(ctx, s, c.user.RID)
		if err != nil {
			internal(w, err)
			return
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) assetIDs(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	list, err := a.assets(r.Context(), `kind = ? AND user_id = ? ORDER BY id`, k.name, c.user.ID)
	if err != nil {
		internal(w, err)
		return
	}
	ids := []int64{}
	for _, s := range list {
		ids = append(ids, s.ID)
	}
	writeJSON(w, http.StatusOK, ids)
}

func (a *api) getAsset(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	s, ok := a.ownAsset(w, r, c, k)
	if !ok {
		return
	}
	j, err := a.assetSchema(r.Context(), s, c.user.RID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (a *api) assetContent(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	s, ok := a.ownAsset(w, r, c, k)
	if !ok {
		return
	}
	a.serveFile(w, r, a.diskPath(s), s.FileName)
}

// badRequest é erro do cliente no upload (vira 400).
type badRequest string

func (e badRequest) Error() string { return string(e) }

// parseUpload lê o multipart com teto de tamanho.
func parseUpload(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			fail(w, http.StatusRequestEntityTooLarge, "Upload too large")
		} else {
			fail(w, http.StatusBadRequest, "Invalid multipart body")
		}
		return false
	}
	return true
}

// writeUpload grava o arquivo enviado em dir/name (temporário + rename:
// quem baixa nunca vê arquivo pela metade). Devolve tamanho e md5.
func writeUpload(fh *multipart.FileHeader, dir, name string) (int64, string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, "", err
	}
	src, err := fh.Open()
	if err != nil {
		return 0, "", err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return 0, "", err
	}
	h := md5.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), src)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), filepath.Join(dir, name))
	}
	if err != nil {
		os.Remove(tmp.Name())
		return 0, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

// storeAsset grava o arquivo e a linha (mesmo nome + emulador = substitui,
// como o RomM).
func (a *api) storeAsset(ctx context.Context, kind string, c *caller, rom, plat *store.Item, emulator string,
	fh *multipart.FileHeader, gallery bool) (*asset, error) {
	name := sanitizeName(fh.Filename)
	if name == "" {
		return nil, badRequest("Invalid " + strings.TrimSuffix(kind, "s") + " filename")
	}
	parts := []string{"romm", kind, strconv.FormatInt(c.user.RID, 10), plat.Platform, strconv.FormatInt(rom.RID, 10)}
	if emulator != "" {
		parts = append(parts, emulator)
	}
	rel := path.Join(parts...)
	size, hash, err := writeUpload(fh, filepath.Join(a.d.Config.Data, filepath.FromSlash(rel)), name)
	if err != nil {
		return nil, err
	}
	now := store.Now()
	_, err = a.d.Store.DB.ExecContext(ctx, `
INSERT INTO romm_assets (kind, user_id, rom_rid, emulator, file_name, file_path, size, content_hash, is_gallery, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(kind, user_id, rom_rid, emulator, file_name) DO UPDATE SET file_path=excluded.file_path, size=excluded.size,
  content_hash=excluded.content_hash, is_gallery=MAX(romm_assets.is_gallery, excluded.is_gallery), updated_at=excluded.updated_at`,
		kind, c.user.ID, rom.RID, emulator, name, rel, size, hash, gallery, now, now)
	if err != nil {
		return nil, err
	}
	return scanAsset(a.d.Store.DB.QueryRowContext(ctx, `SELECT `+assetCols+` FROM romm_assets
WHERE kind = ? AND user_id = ? AND rom_rid = ? AND emulator = ? AND file_name = ?`, kind, c.user.ID, rom.RID, emulator, name))
}

func (a *api) touchPlayed(ctx context.Context, c *caller, rom *store.Item) {
	a.d.Store.UpdateUserData(ctx, c.user.ID, rom.ID, func(d *store.UserData) { d.LastPlayed = time.Now().UTC() })
}

func uploadFailed(w http.ResponseWriter, err error) {
	var bad badRequest
	if errors.As(err, &bad) {
		fail(w, http.StatusBadRequest, bad.Error())
		return
	}
	internal(w, err)
}

// uploadAsset é o POST /api/{saves|states|screenshots}?rom_id=&emulator=.
// Campo do arquivo: saveFile/stateFile/screenshotFile (+ screenshotFile
// opcional junto do save/state). O campo antigo no plural (saves=...)
// aceita vários arquivos e responde no formato velho {uploaded, saves}.
func (a *api) uploadAsset(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	ctx := r.Context()
	ids, err := queryInts(r, "rom_id")
	if err != nil || len(ids) != 1 {
		fail(w, http.StatusUnprocessableEntity, "Field required: rom_id")
		return
	}
	rom, err := a.itemOfKind(ctx, ids[0], store.KindRom)
	var plat *store.Item
	if err == nil {
		plat, err = a.platformOf(ctx, rom)
	}
	if err != nil {
		fail(w, http.StatusNotFound, "Rom with ID "+strconv.FormatInt(ids[0], 10)+" not found")
		return
	}
	emulator := strings.TrimSpace(r.URL.Query().Get("emulator"))
	if k == kindScreenshots {
		emulator = ""
	}
	if !emulatorOK(emulator) {
		fail(w, http.StatusBadRequest, "Invalid emulator name")
		return
	}
	if !parseUpload(w, r) {
		return
	}
	defer r.MultipartForm.RemoveAll()
	main, legacy := r.MultipartForm.File[k.field], r.MultipartForm.File[k.name]
	if len(main) == 0 && len(legacy) == 0 {
		fail(w, http.StatusUnprocessableEntity, "Field required: "+k.field)
		return
	}
	gallery := k == kindScreenshots
	if len(main) > 0 {
		s, err := a.storeAsset(ctx, k.name, c, rom, plat, emulator, main[0], gallery)
		if err != nil {
			uploadFailed(w, err)
			return
		}
		if k != kindScreenshots {
			if shots := r.MultipartForm.File[kindScreenshots.field]; len(shots) > 0 {
				if _, err := a.storeAsset(ctx, kindScreenshots.name, c, rom, plat, "", shots[0], false); err != nil {
					uploadFailed(w, err)
					return
				}
			}
			a.touchPlayed(ctx, c, rom)
		}
		j, err := a.assetSchema(ctx, s, c.user.RID)
		if err != nil {
			internal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, j)
		return
	}
	out := []assetJSON{}
	for _, fh := range legacy {
		s, err := a.storeAsset(ctx, k.name, c, rom, plat, emulator, fh, gallery)
		if err != nil {
			uploadFailed(w, err)
			return
		}
		j, err := a.assetSchema(ctx, s, c.user.RID)
		if err != nil {
			internal(w, err)
			return
		}
		out = append(out, j)
	}
	if k != kindScreenshots {
		a.touchPlayed(ctx, c, rom)
	}
	writeJSON(w, http.StatusOK, map[string]any{"uploaded": len(out), k.name: out})
}

// updateAsset é o PUT /api/{tipo}/{id}: troca o conteúdo (multipart) ou,
// no screenshot, muda a visibilidade ({"is_public": true}).
func (a *api) updateAsset(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	s, ok := a.ownAsset(w, r, c, k)
	if !ok {
		return
	}
	ctx := r.Context()
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var in struct {
			IsPublic *bool `json:"is_public"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil || in.IsPublic == nil {
			fail(w, http.StatusUnprocessableEntity, "Field required: is_public")
			return
		}
		if _, err := a.d.Store.DB.ExecContext(ctx, `UPDATE romm_assets SET is_public = ? WHERE id = ?`, *in.IsPublic, s.ID); err != nil {
			internal(w, err)
			return
		}
	} else {
		if !parseUpload(w, r) {
			return
		}
		defer r.MultipartForm.RemoveAll()
		if files := r.MultipartForm.File[k.field]; len(files) > 0 {
			size, hash, err := writeUpload(files[0], filepath.Dir(a.diskPath(s)), s.FileName)
			if err != nil {
				internal(w, err)
				return
			}
			if _, err := a.d.Store.DB.ExecContext(ctx, `UPDATE romm_assets SET size = ?, content_hash = ?, updated_at = ? WHERE id = ?`,
				size, hash, store.Now(), s.ID); err != nil {
				internal(w, err)
				return
			}
		}
		rom, err := a.itemOfKind(ctx, s.RomRID, store.KindRom)
		if err == nil && k != kindScreenshots {
			if shots := r.MultipartForm.File[kindScreenshots.field]; len(shots) > 0 {
				if plat, err := a.platformOf(ctx, rom); err == nil {
					if _, err := a.storeAsset(ctx, kindScreenshots.name, c, rom, plat, "", shots[0], false); err != nil {
						uploadFailed(w, err)
						return
					}
				}
			}
			a.touchPlayed(ctx, c, rom)
		}
	}
	s, err := a.assetByID(ctx, k.name, s.ID)
	if err != nil {
		internal(w, err)
		return
	}
	j, err := a.assetSchema(ctx, s, c.user.RID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (a *api) removeAsset(ctx context.Context, s *asset) error {
	if err := os.Remove(a.diskPath(s)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_, err := a.d.Store.DB.ExecContext(ctx, `DELETE FROM romm_assets WHERE id = ?`, s.ID)
	return err
}

func (a *api) deleteAsset(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	s, ok := a.ownAsset(w, r, c, k)
	if !ok {
		return
	}
	if err := a.removeAsset(r.Context(), s); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// deleteAssets é o POST /api/{tipo}/delete {"saves": [1, 2]}: confere
// todos antes de apagar qualquer um.
func (a *api) deleteAssets(w http.ResponseWriter, r *http.Request, c *caller, k assetKind) {
	var in map[string][]int64
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		fail(w, http.StatusUnprocessableEntity, "Invalid body")
		return
	}
	ids := in[k.name]
	if len(ids) == 0 {
		fail(w, http.StatusBadRequest, "No "+k.name+" were provided")
		return
	}
	ctx := r.Context()
	var list []*asset
	for _, id := range ids {
		s, err := a.assetByID(ctx, k.name, id)
		if err != nil || s.UserID != c.user.ID {
			fail(w, http.StatusNotFound, k.label+" with ID "+strconv.FormatInt(id, 10)+" not found")
			return
		}
		list = append(list, s)
	}
	for _, s := range list {
		if err := a.removeAsset(ctx, s); err != nil {
			internal(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, ids)
}

// savesSummary é o GET /api/saves/summary?rom_id= (tudo num slot só: null).
func (a *api) savesSummary(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	ids, err := queryInts(r, "rom_id")
	if err != nil || len(ids) != 1 {
		fail(w, http.StatusUnprocessableEntity, "Field required: rom_id")
		return
	}
	list, err := a.assets(ctx, `kind = 'saves' AND user_id = ? AND rom_rid = ? ORDER BY updated_at DESC, id DESC`, c.user.ID, ids[0])
	if err != nil {
		internal(w, err)
		return
	}
	slots := []any{}
	if len(list) > 0 {
		latest, err := a.assetSchema(ctx, list[0], c.user.RID)
		if err != nil {
			internal(w, err)
			return
		}
		slots = append(slots, map[string]any{"slot": nil, "count": len(list), "latest": latest})
	}
	writeJSON(w, http.StatusOK, map[string]any{"total_count": len(list), "slots": slots})
}
