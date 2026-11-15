package romm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// romPaths junta os caminhos de uma ROM como o RomM mostra (relativos à
// raiz da biblioteca: roms/<console>/...).
type romPaths struct {
	fsPath   string // roms/snes
	fsName   string // arquivo ou pasta
	fullPath string // roms/snes/<fs_name>
}

func pathsOf(it *store.Item) romPaths {
	rel := filepath.ToSlash(it.RelPath)
	p := romPaths{fsPath: path.Dir(rel), fsName: extraString(it, "fs_name"), fullPath: rel}
	if p.fsName == "" {
		p.fsName = path.Base(rel)
	}
	return p
}

// fileFull é o caminho do arquivo relativo à biblioteca.
func (p romPaths) fileFull(f store.RomFile) string { return path.Join(p.fsPath, f.Rel) }

// filePath devolve onde o arquivo está no disco, sem sair da pasta do
// console (Rel vem do scanner, mas não custa conferir).
func fileOnDisk(rom *store.Item, f store.RomFile) (string, bool) {
	base := filepath.Dir(rom.Path)
	p := filepath.Join(base, filepath.FromSlash(f.Rel))
	if !strings.HasPrefix(p, base+string(filepath.Separator)) {
		return "", false
	}
	return p, true
}

func (p romPaths) isTopLevel(f store.RomFile) bool {
	full := p.fileFull(f)
	return full == p.fullPath || path.Dir(full) == p.fullPath
}

func sortedFiles(it *store.Item) []store.RomFile {
	files := it.Files()
	slices.SortFunc(files, func(x, y store.RomFile) int { return strings.Compare(x.Name, y.Name) })
	return files
}

func extraStrings(it *store.Item, key string) []string {
	out := []string{}
	if list, ok := it.Extra[key].([]any); ok {
		for _, v := range list {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func coverPath(platformRID int64, it *store.Item, size string) string {
	if it.Image == "" {
		return ""
	}
	return fmt.Sprintf("/assets/romm/resources/roms/%d/%d/cover/%s.png?ts=%d", platformRID, it.RID, size, it.UpdatedAt.Unix())
}

// romFiles monta os RomFileSchema (com id) da ROM.
func (a *api) romFiles(ctx context.Context, it *store.Item) ([]romFileSchema, error) {
	files := sortedFiles(it)
	rels := make([]string, len(files))
	for i, f := range files {
		rels[i] = f.Rel
	}
	ids, err := a.fileIDs(ctx, it.RID, rels)
	if err != nil {
		return nil, err
	}
	p := pathsOf(it)
	out := make([]romFileSchema, 0, len(files))
	for _, f := range files {
		full := p.fileFull(f)
		s := romFileSchema{ID: ids[f.Rel], RomID: it.RID, FileName: f.Name, FilePath: path.Dir(full), FileSizeBytes: f.Size,
			FullPath: full, IsTopLevel: p.isTopLevel(f), CreatedAt: ts(it.AddedAt), UpdatedAt: ts(it.UpdatedAt),
			CRCHash: orNil(f.CRC), MD5Hash: orNil(f.MD5), SHA1Hash: orNil(f.SHA1)}
		if it.MTime > 0 {
			s.LastModified = ts(time.Unix(it.MTime, 0))
		}
		if s.IsTopLevel {
			game := "game"
			s.Category = &game
		}
		out = append(out, s)
	}
	return out, nil
}

// romUserFor monta o rom_user do usuário (padrão do RomM quando não tem).
func (a *api) romUserFor(ctx context.Context, c *caller, it *store.Item) (romUserSchema, error) {
	ud, err := a.d.Store.UserData(ctx, c.user.ID, it.ID)
	if err != nil {
		return romUserSchema{}, err
	}
	ru, err := a.romUser(ctx, c.user.ID, it.RID)
	if err != nil {
		return romUserSchema{}, err
	}
	if ru == nil {
		now := ts(time.Now())
		return romUserSchema{ID: -1, UserID: c.user.RID, RomID: it.RID, CreatedAt: now, UpdatedAt: now, LastPlayed: ts(ud.LastPlayed)}, nil
	}
	s := romUserSchema{ID: ru.ID, UserID: c.user.RID, RomID: it.RID, CreatedAt: ts(ru.Created), UpdatedAt: ts(ru.Updated),
		LastPlayed: ts(ud.LastPlayed), IsMainSibling: ru.IsMainSibling, Backlogged: ru.Backlogged, NowPlaying: ru.NowPlaying,
		Hidden: ru.Hidden, Rating: ru.Rating, Difficulty: ru.Difficulty, Completion: ru.Completion}
	if ru.Status.Valid {
		s.Status = &ru.Status.String
	}
	if ru.PinnedMedia.Valid {
		json.Unmarshal([]byte(ru.PinnedMedia.String), &s.PinnedMedia)
	}
	return s, nil
}

// newRomSchema monta o RomSchema (files só com withFiles).
func (a *api) newRomSchema(ctx context.Context, c *caller, it, plat *store.Item, withFiles bool) (romSchema, error) {
	p := pathsOf(it)
	files := it.Files()
	multi, _ := it.Extra["multi"].(bool)
	top := 0
	for _, f := range files {
		if p.isTopLevel(f) {
			top++
		}
	}
	slug := plat.Platform
	s := romSchema{ID: it.RID, PlatformID: plat.RID, PlatformSlug: slug, PlatformFsSlug: slug, PlatformName: plat.Name,
		PlatformDisplay: plat.Name, FsName: p.fsName, FsNameNoTags: nameNoTags(p.fsName), FsNameNoExt: nameNoExt(p.fsName),
		FsExtension: nameExt(p.fsName), FsPath: p.fsPath, FsSizeBytes: it.Size, Name: it.Name,
		NameSortKey: it.SortName, Summary: orNil(it.Overview), AlternativeNames: []string{},
		Metadatum: romMetadataSchema{RomID: it.RID, Genres: it.Genres, Franchises: []string{}, Collections: []string{},
			Companies: []string{}, Publishers: []string{}, Developers: []string{}, GameModes: []string{}, AgeRatings: []string{}},
		PathCoverSmall: coverPath(plat.RID, it, "small"), PathCoverLarge: coverPath(plat.RID, it, "big"),
		IsUnidentified: true, Regions: extraStrings(it, "regions"), Languages: extraStrings(it, "languages"),
		Tags: extraStrings(it, "tags"), HasSimpleSingle: !multi && top == 1, HasNestedSingle: multi && top == 1,
		HasMultipleFiles: top > 1, FullPath: p.fullPath, CreatedAt: ts(it.AddedAt), UpdatedAt: ts(it.UpdatedAt),
		HasFileOnDisk: true, MergedScreenshots: []string{}, Files: []romFileSchema{}, SiblingRoms: []any{}, Multi: multi}
	if it.Year > 0 {
		ms := time.Date(it.Year, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
		s.Metadatum.FirstReleaseDate = &ms
	}
	if len(files) == 1 {
		s.CRCHash, s.MD5Hash, s.SHA1Hash = orNil(files[0].CRC), orNil(files[0].MD5), orNil(files[0].SHA1)
	}
	var err error
	if s.RomUser, err = a.romUserFor(ctx, c, it); err != nil {
		return s, err
	}
	if withFiles {
		if s.Files, err = a.romFiles(ctx, it); err != nil {
			return s, err
		}
	}
	return s, nil
}

// romItem busca a ROM do {id} e o console dela (404 já respondido).
func (a *api) romItem(w http.ResponseWriter, r *http.Request) (*store.Item, *store.Item, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return nil, nil, false
	}
	ctx := r.Context()
	it, err := a.itemOfKind(ctx, id, store.KindRom)
	if err == nil {
		var p *store.Item
		if p, err = a.platformOf(ctx, it); err == nil {
			return it, p, true
		}
	}
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "Rom with ID "+r.PathValue("id")+" not found")
	} else {
		internal(w, err)
	}
	return nil, nil, false
}

func (a *api) romSimple(w http.ResponseWriter, r *http.Request, c *caller) {
	it, plat, ok := a.romItem(w, r)
	if !ok {
		return
	}
	s, err := a.newRomSchema(r.Context(), c, it, plat, true)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, simpleRomSchema{romSchema: s})
}

// rom é o GET /api/roms/{id} (DetailedRomSchema).
func (a *api) rom(w http.ResponseWriter, r *http.Request, c *caller) {
	it, plat, ok := a.romItem(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	base, err := a.newRomSchema(ctx, c, it, plat, true)
	if err != nil {
		internal(w, err)
		return
	}
	d := detailedRomSchema{romSchema: base, UserChannels: []any{}, SnapshotSaveIDs: []int64{}, UserCollections: []any{},
		AllUserNotes: []any{}, DownloadFormats: []string{}, UserNotes: []any{}}
	list := func(kind string, gallery bool) ([]assetJSON, []assetJSON, error) {
		where := `user_id = ? AND rom_rid = ? AND kind = ?`
		if gallery {
			where += ` AND is_gallery = 1`
		}
		rows, err := a.assets(ctx, where+` ORDER BY updated_at DESC, id DESC`, c.user.ID, it.RID, kind)
		if err != nil {
			return nil, nil, err
		}
		own, all := []assetJSON{}, []assetJSON{}
		for _, s := range rows {
			j, err := a.assetSchema(ctx, s, c.user.RID)
			if err != nil {
				return nil, nil, err
			}
			own = append(own, j)
			all = append(all, withOwner(j, c))
		}
		return own, all, nil
	}
	if d.UserSaves, d.AllUserSaves, err = list("saves", false); err == nil {
		if d.UserStates, d.AllUserStates, err = list("states", false); err == nil {
			if d.UserScreenshots, _, err = list("screenshots", false); err == nil {
				_, d.AllUserScreenshots, err = list("screenshots", true)
			}
		}
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// withOwner é o User*Schema: o asset com o nome de quem é dono.
func withOwner(j assetJSON, c *caller) assetJSON {
	out := make(assetJSON, len(j)+3)
	for k, v := range j {
		out[k] = v
	}
	out["username"] = c.user.Name
	out["user_avatar_path"] = ""
	out["user_updated_at"] = ts(c.user.CreatedAt)
	return out
}

// romQuery é o filtro/ordem do GET /api/roms em SQL.
type romQuery struct {
	where   []string
	args    []any
	order   string
	ordArgs []any
}

var romUserStatuses = []string{"incomplete", "finished", "completed_100", "retired", "never_playing"}

// parseRomQuery lê os filtros do RomM (RomFilterParams) e a ordem.
func parseRomQuery(r *http.Request, c *caller) (*romQuery, error) {
	q := &romQuery{where: []string{`i.kind = ?`}, args: []any{store.KindRom}}
	add := func(cond string, args ...any) {
		q.where = append(q.where, cond)
		q.args = append(q.args, args...)
	}
	pids, err := queryInts(r, "platform_ids", "platform_id")
	if err != nil {
		return nil, err
	}
	if len(pids) > 0 {
		add(`i.parent_id IN (SELECT id FROM items WHERE kind = 'platform' AND rid IN (`+placeholders(len(pids))+`))`, anys(pids)...)
	}
	if term := strings.TrimSpace(r.URL.Query().Get("search_term")); term != "" {
		add(`(instr(lower(i.name), lower(?)) > 0 OR instr(lower(i.rel_path), lower(?)) > 0)`, term, term)
	}
	// coleções não existem aqui: filtrar por uma = nada
	for _, k := range []string{"collection_id", "virtual_collection_id", "smart_collection_id"} {
		if r.URL.Query().Get(k) != "" {
			add(`0 = 1`)
		}
	}
	bools := map[string]string{
		"favorite":    `EXISTS (SELECT 1 FROM userdata u WHERE u.user_id = ? AND u.item_id = i.id AND u.favorite = 1)`,
		"favourite":   `EXISTS (SELECT 1 FROM userdata u WHERE u.user_id = ? AND u.item_id = i.id AND u.favorite = 1)`,
		"last_played": `EXISTS (SELECT 1 FROM userdata u WHERE u.user_id = ? AND u.item_id = i.id AND u.last_played IS NOT NULL)`,
		"has_saves":   `EXISTS (SELECT 1 FROM romm_assets s WHERE s.user_id = ? AND s.rom_rid = i.rid AND s.kind = 'saves')`,
		"has_states":  `EXISTS (SELECT 1 FROM romm_assets s WHERE s.user_id = ? AND s.rom_rid = i.rid AND s.kind = 'states')`,
	}
	for _, k := range []string{"favorite", "favourite", "last_played", "has_saves", "has_states"} {
		b, err := queryBool(r, k)
		if err != nil {
			return nil, err
		}
		if b != nil {
			cond := bools[k]
			if !*b {
				cond = `NOT ` + cond
			}
			add(cond, c.user.ID)
		}
	}
	// o que o acervo nunca tem: missing, physical, RetroAchievements, verificado...
	for _, k := range []string{"missing", "physical", "has_ra", "verified", "matched", "duplicate", "has_soundtrack"} {
		b, err := queryBool(r, k)
		if err != nil {
			return nil, err
		}
		if b != nil && *b {
			add(`0 = 1`)
		}
	}
	for _, k := range []string{"regions", "languages", "tags"} {
		if vals := queryStrings(r, k); len(vals) > 0 {
			add(`EXISTS (SELECT 1 FROM json_each(i.extra, '$.`+k+`') j WHERE j.value IN (`+placeholders(len(vals))+`))`, anys(vals)...)
		}
	}
	if vals := queryStrings(r, "genres"); len(vals) > 0 {
		add(`EXISTS (SELECT 1 FROM json_each(i.genres) j WHERE j.value IN (`+placeholders(len(vals))+`))`, anys(vals)...)
	}
	if vals := queryStrings(r, "statuses"); len(vals) > 0 {
		add(`EXISTS (SELECT 1 FROM romm_rom_users ru WHERE ru.user_id = ? AND ru.rom_rid = i.rid AND ru.status IN (`+placeholders(len(vals))+`))`,
			append([]any{c.user.ID}, anys(vals)...)...)
	}
	if v := r.URL.Query().Get("updated_after"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, strings.Replace(v, " ", "+", 1))
		if err != nil {
			return nil, errors.New("Invalid updated_after")
		}
		add(`i.updated_at > ?`, t.UTC().Format(time.RFC3339Nano))
	}

	dir := "ASC"
	switch strings.ToLower(r.URL.Query().Get("order_dir")) {
	case "", "asc":
	case "desc":
		dir = "DESC"
	default:
		return nil, errors.New("Invalid order_dir")
	}
	col := `i.sort_name`
	switch strings.ToLower(r.URL.Query().Get("order_by")) {
	case "fs_name":
		col = `lower(i.rel_path)`
	case "fs_size_bytes":
		col = `i.size`
	case "created_at":
		col = `i.added_at`
	case "updated_at":
		col = `i.updated_at`
	case "id":
		col = `i.rid`
	case "first_release_date":
		col = `i.year`
	case "last_played":
		col = `(SELECT u.last_played FROM userdata u WHERE u.user_id = ? AND u.item_id = i.id)`
		q.ordArgs = append(q.ordArgs, c.user.ID)
	case "rating":
		col = `(SELECT ru.rating FROM romm_rom_users ru WHERE ru.user_id = ? AND ru.rom_rid = i.rid)`
		q.ordArgs = append(q.ordArgs, c.user.ID)
	case "platform_name", "platform":
		col = `(SELECT p.sort_name FROM items p WHERE p.id = i.parent_id)`
	}
	q.order = col + ` ` + dir + `, i.sort_name ` + dir + `, i.rid ` + dir
	return q, nil
}

func (q *romQuery) whereSQL() string { return strings.Join(q.where, ` AND `) }

func placeholders(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }

func anys[T any](v []T) []any {
	out := make([]any, len(v))
	for i := range v {
		out[i] = v[i]
	}
	return out
}

// roms é o GET /api/roms paginado (CustomLimitOffsetPage[SimpleRomSchema]).
func (a *api) roms(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	q, err := parseRomQuery(r, c)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	limit, offset := int64(50), int64(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err = strconv.ParseInt(v, 10, 64); err != nil || limit < 1 || limit > 10000 {
			fail(w, http.StatusUnprocessableEntity, "Invalid limit")
			return
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err = strconv.ParseInt(v, 10, 64); err != nil || offset < 0 {
			fail(w, http.StatusUnprocessableEntity, "Invalid offset")
			return
		}
	}
	opt := func(name string, def bool) bool {
		b, _ := queryBool(r, name)
		if b == nil {
			return def
		}
		return *b
	}
	withChar, withFilters, withIndex := opt("with_char_index", true), opt("with_filter_values", true), opt("with_rom_id_index", true)
	withTotal, withFiles := opt("with_total", true), opt("with_files", false)

	where := q.whereSQL()
	ordered := append(append([]any{}, q.args...), q.ordArgs...)
	page, err := a.d.Store.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE `+where+` ORDER BY `+q.order+` LIMIT ? OFFSET ?`,
		append(ordered, limit, offset)...)
	if err != nil {
		internal(w, err)
		return
	}
	var total any
	if withTotal || withIndex {
		var n int64
		if err := a.d.Store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM items i WHERE `+where, q.args...).Scan(&n); err != nil {
			internal(w, err)
			return
		}
		total = n
	}
	index := []int64{}
	charIndex := map[string]int{}
	if withIndex || withChar {
		rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT i.rid, i.sort_name FROM items i WHERE `+where+` ORDER BY `+q.order, ordered...)
		if err != nil {
			internal(w, err)
			return
		}
		pos := 0
		byName := r.URL.Query().Get("order_by") == "" || strings.EqualFold(r.URL.Query().Get("order_by"), "name")
		for rows.Next() {
			var rid int64
			var sortName string
			if err := rows.Scan(&rid, &sortName); err != nil {
				rows.Close()
				internal(w, err)
				return
			}
			if withIndex {
				index = append(index, rid)
			}
			if withChar && byName && sortName != "" {
				ch, _ := utf8.DecodeRuneInString(sortName)
				key := strings.ToLower(string(ch))
				if _, ok := charIndex[key]; !ok {
					charIndex[key] = pos
				}
			}
			pos++
		}
		rows.Close()
	}
	plats, err := a.platformsByID(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	items := []simpleRomSchema{}
	for _, it := range page {
		p := plats[it.ParentID]
		if p == nil {
			continue
		}
		s, err := a.newRomSchema(ctx, c, it, p.it, withFiles)
		if err != nil {
			internal(w, err)
			return
		}
		items = append(items, simpleRomSchema{romSchema: s})
	}
	filters := emptyFilterValues()
	if withFilters {
		if filters, err = a.filterValues(ctx); err != nil {
			internal(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset,
		"char_index": charIndex, "rom_id_index": index, "filter_values": filters})
}

func emptyFilterValues() map[string]any {
	out := map[string]any{"platforms": []int64{}}
	for _, k := range []string{"genres", "franchises", "collections", "companies", "publishers", "developers", "game_modes",
		"age_ratings", "player_counts", "regions", "languages", "tags"} {
		out[k] = []string{}
	}
	return out
}

// filterValues junta os valores possíveis dos filtros (RomFiltersDict).
func (a *api) filterValues(ctx context.Context) (map[string]any, error) {
	out := emptyFilterValues()
	for _, k := range []string{"regions", "languages", "tags"} {
		vals, err := a.distinct(ctx, `SELECT DISTINCT j.value FROM items i, json_each(i.extra, '$.`+k+`') j WHERE i.kind = ? ORDER BY 1`)
		if err != nil {
			return nil, err
		}
		out[k] = vals
	}
	genres, err := a.distinct(ctx, `SELECT DISTINCT j.value FROM items i, json_each(i.genres) j WHERE i.kind = ? ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	out["genres"] = genres
	rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT DISTINCT p.rid FROM items i JOIN items p ON p.id = i.parent_id WHERE i.kind = ? ORDER BY 1`, store.KindRom)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	out["platforms"] = ids
	return out, rows.Err()
}

func (a *api) distinct(ctx context.Context, query string) ([]string, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, query, store.KindRom)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var v any
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

func (a *api) romFilters(w http.ResponseWriter, r *http.Request, c *caller) {
	f, err := a.filterValues(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (a *api) romIDs(w http.ResponseWriter, r *http.Request, c *caller) {
	rows, err := a.d.Store.DB.QueryContext(r.Context(), `SELECT rid FROM items WHERE kind = ? ORDER BY sort_name, rid`, store.KindRom)
	if err != nil {
		internal(w, err)
		return
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			internal(w, err)
			return
		}
		ids = append(ids, id)
	}
	writeJSON(w, http.StatusOK, ids)
}

// romFile é o GET /api/roms/{id}/files: o {id} aqui é o do arquivo.
func (a *api) romFile(w http.ResponseWriter, r *http.Request, c *caller) {
	rom, f, ok := a.fileItem(w, r)
	if !ok {
		return
	}
	files, err := a.romFiles(r.Context(), rom)
	if err != nil {
		internal(w, err)
		return
	}
	for _, s := range files {
		if s.FileName == f.Name && s.FullPath == pathsOf(rom).fileFull(f) {
			writeJSON(w, http.StatusOK, s)
			return
		}
	}
	fail(w, http.StatusNotFound, "File not found")
}

// fileItem acha a ROM e o arquivo de um id de arquivo.
func (a *api) fileItem(w http.ResponseWriter, r *http.Request) (*store.Item, store.RomFile, bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return nil, store.RomFile{}, false
	}
	ctx := r.Context()
	romRID, rel, err := a.fileByID(ctx, id)
	if err == nil {
		var rom *store.Item
		if rom, err = a.itemOfKind(ctx, romRID, store.KindRom); err == nil {
			for _, f := range rom.Files() {
				if f.Rel == rel {
					return rom, f, true
				}
			}
			err = store.ErrNotFound
		}
	}
	if errors.Is(err, store.ErrNotFound) {
		fail(w, http.StatusNotFound, "File not found")
	} else {
		internal(w, err)
	}
	return nil, store.RomFile{}, false
}
