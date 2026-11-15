package romm

import (
	"context"
	"errors"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// platformInfo é um console com o que o PlatformSchema precisa.
type platformInfo struct {
	it       *store.Item
	romCount int64
	size     int64
}

// platformsByID carrega os consoles (chave: id 32 hex do item).
func (a *api) platformsByID(ctx context.Context) (map[string]*platformInfo, error) {
	items, err := a.d.Store.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = ? ORDER BY i.sort_name, i.rid`, store.KindPlatform)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*platformInfo, len(items))
	for _, it := range items {
		out[it.ID] = &platformInfo{it: it}
	}
	rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT parent_id, COUNT(*), COALESCE(SUM(size),0) FROM items WHERE kind = ? GROUP BY parent_id`, store.KindRom)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n, size int64
		if err := rows.Scan(&id, &n, &size); err != nil {
			return nil, err
		}
		if p := out[id]; p != nil {
			p.romCount, p.size = n, size
		}
	}
	return out, rows.Err()
}

// sortedPlatforms devolve os consoles em ordem de nome.
func sortedPlatforms(m map[string]*platformInfo) []*platformInfo {
	out := make([]*platformInfo, 0, len(m))
	for _, p := range m {
		out = append(out, p)
	}
	slices.SortFunc(out, func(x, y *platformInfo) int {
		if c := strings.Compare(x.it.SortName, y.it.SortName); c != 0 {
			return c
		}
		return int(x.it.RID - y.it.RID)
	})
	return out
}

// firmwareItems lista as BIOS (de um console, ou todas com parentID "").
func (a *api) firmwareItems(ctx context.Context, parentID string) ([]*store.Item, error) {
	return a.d.Store.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = ? AND (? = '' OR i.parent_id = ?)
ORDER BY i.name, i.rid`, store.KindFirmware, parentID, parentID)
}

func extraString(it *store.Item, key string) string {
	s, _ := it.Extra[key].(string)
	return s
}

func newFirmwareSchema(it *store.Item, platformRID int64) firmwareSchema {
	rel := filepath.ToSlash(it.RelPath)
	name := path.Base(rel)
	if it.Name != "" {
		name = it.Name
	}
	return firmwareSchema{ID: it.RID, PlatformID: platformRID, FileName: name, FileNameNoTags: nameNoTags(name),
		FileNameNoExt: nameNoExt(name), FileExtension: nameExt(name), FilePath: path.Dir(rel), FileSizeBytes: it.Size,
		FullPath: rel, CRCHash: extraString(it, "crc"), MD5Hash: extraString(it, "md5"), SHA1Hash: extraString(it, "sha1"),
		CreatedAt: ts(it.AddedAt), UpdatedAt: ts(it.UpdatedAt)}
}

func (a *api) newPlatformSchema(p *platformInfo, fw []*store.Item) platformSchema {
	it := p.it
	slug := it.Platform
	if slug == "" {
		slug = path.Base(filepath.ToSlash(it.RelPath))
	}
	s := platformSchema{ID: it.RID, Slug: slug, FsSlug: slug, RomCount: p.romCount, Name: it.Name,
		LogoPath: "", Firmware: []firmwareSchema{}, CreatedAt: ts(it.AddedAt), UpdatedAt: ts(it.UpdatedAt),
		FsSizeBytes: p.size, IsUnidentified: true, AspectRatio: "2 / 3", DisplayName: it.Name,
		Abbreviation: strings.ToUpper(slug), AlternativeNames: []string{}}
	for _, f := range fw {
		if f.ParentID == it.ID {
			s.Firmware = append(s.Firmware, newFirmwareSchema(f, it.RID))
		}
	}
	slices.SortFunc(s.Firmware, func(x, y firmwareSchema) int { return strings.Compare(x.FileName, y.FileName) })
	s.FirmwareCount = len(s.Firmware)
	return s
}

func (a *api) platforms(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	plats, err := a.platformsByID(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	fw, err := a.firmwareItems(ctx, "")
	if err != nil {
		internal(w, err)
		return
	}
	out := []platformSchema{}
	for _, p := range sortedPlatforms(plats) {
		out = append(out, a.newPlatformSchema(p, fw))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) platformIDs(w http.ResponseWriter, r *http.Request, c *caller) {
	plats, err := a.platformsByID(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := []int64{}
	for _, p := range sortedPlatforms(plats) {
		out = append(out, p.it.RID)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) platform(w http.ResponseWriter, r *http.Request, c *caller) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	ctx := r.Context()
	plats, err := a.platformsByID(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	for _, p := range plats {
		if p.it.RID == id {
			fw, err := a.firmwareItems(ctx, p.it.ID)
			if err != nil {
				internal(w, err)
				return
			}
			writeJSON(w, http.StatusOK, a.newPlatformSchema(p, fw))
			return
		}
	}
	fail(w, http.StatusNotFound, "Platform not found")
}

// itemOfKind busca um item pelo id numérico conferindo o tipo.
func (a *api) itemOfKind(ctx context.Context, rid int64, kind string) (*store.Item, error) {
	it, err := a.d.Store.ItemByRID(ctx, rid)
	if err != nil {
		return nil, err
	}
	if it.Kind != kind {
		return nil, store.ErrNotFound
	}
	return it, nil
}

// platformRID devolve o id numérico do console de um item.
func (a *api) platformOf(ctx context.Context, it *store.Item) (*store.Item, error) {
	if it.ParentID == "" {
		return nil, store.ErrNotFound
	}
	return a.d.Store.Item(ctx, it.ParentID)
}

func (a *api) firmwareList(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	parent := ""
	if v := r.URL.Query().Get("platform_id"); v != "" {
		pid, err := queryInts(r, "platform_id")
		if err != nil || len(pid) != 1 {
			fail(w, http.StatusUnprocessableEntity, "Invalid platform_id")
			return
		}
		p, err := a.itemOfKind(ctx, pid[0], store.KindPlatform)
		if err != nil {
			writeJSON(w, http.StatusOK, []firmwareSchema{})
			return
		}
		parent = p.ID
	}
	items, err := a.firmwareItems(ctx, parent)
	if err != nil {
		internal(w, err)
		return
	}
	plats, err := a.platformsByID(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	out := []firmwareSchema{}
	for _, it := range items {
		if p := plats[it.ParentID]; p != nil {
			out = append(out, newFirmwareSchema(it, p.it.RID))
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *api) firmwareIDs(w http.ResponseWriter, r *http.Request, c *caller) {
	items, err := a.firmwareItems(r.Context(), "")
	if err != nil {
		internal(w, err)
		return
	}
	out := []int64{}
	for _, it := range items {
		out = append(out, it.RID)
	}
	writeJSON(w, http.StatusOK, out)
}

// firmwareItem busca a BIOS do {id} (404 já respondido se não achar).
func (a *api) firmwareItem(w http.ResponseWriter, r *http.Request) (*store.Item, *store.Item, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return nil, nil, false
	}
	ctx := r.Context()
	it, err := a.itemOfKind(ctx, id, store.KindFirmware)
	if err == nil {
		var p *store.Item
		if p, err = a.platformOf(ctx, it); err == nil {
			return it, p, true
		}
	}
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Firmware with ID "+r.PathValue("id")+" not found")
	} else {
		internal(w, err)
	}
	return nil, nil, false
}

func (a *api) firmware(w http.ResponseWriter, r *http.Request, c *caller) {
	it, p, ok := a.firmwareItem(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, newFirmwareSchema(it, p.RID))
}

func (a *api) firmwareContent(w http.ResponseWriter, r *http.Request, c *caller) {
	it, _, ok := a.firmwareItem(w, r)
	if !ok {
		return
	}
	a.serveFile(w, r, it.Path, newFirmwareSchema(it, 0).FileName)
}
