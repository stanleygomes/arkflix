package romm

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// disposition monta o Content-Disposition com nome UTF-8 (RFC 6266).
func disposition(kind, name string) string {
	ascii := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)
	return fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`, kind, ascii, url.PathEscape(name))
}

// serveFile manda um arquivo do disco como anexo, com Range e HEAD.
func (a *api) serveFile(w http.ResponseWriter, r *http.Request, file, name string) {
	st, err := os.Stat(file)
	if err != nil || !st.Mode().IsRegular() {
		fail(w, http.StatusNotFound, "File not found")
		return
	}
	w.Header().Set("Content-Disposition", disposition("attachment", name))
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" && !strings.HasPrefix(ct, "text/") {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	a.d.Media.ServeFile(w, r, file)
}

// zipEntry é um arquivo do zip montado na hora.
type zipEntry struct {
	name string // dentro do zip
	file string // no disco ("" = data)
	data []byte
	mod  time.Time
}

// serveZip manda os arquivos num zip sem compressão, escrito enquanto vai
// (ROM já é comprimida; e não precisa de espaço em disco).
func serveZip(w http.ResponseWriter, r *http.Request, name string, entries []zipEntry) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", disposition("attachment", name))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	// confere antes de começar: no meio do zip não dá mais pra mandar 404
	for _, e := range entries {
		if e.file == "" {
			continue
		}
		if st, err := os.Stat(e.file); err != nil || !st.Mode().IsRegular() {
			fail(w, http.StatusNotFound, "File "+path.Base(e.name)+" not found on disk")
			return
		}
	}
	zw := zip.NewWriter(w)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Store, Modified: e.mod}
		h.SetMode(0o600)
		out, err := zw.CreateHeader(h)
		if err != nil {
			return
		}
		if e.file == "" {
			out.Write(e.data)
			continue
		}
		f, err := os.Open(e.file)
		if err != nil {
			slog.Error("romm: zip", "file", e.file, "err", err)
			return
		}
		_, err = io.Copy(out, f)
		f.Close()
		if err != nil {
			return // cliente foi embora
		}
	}
	zw.Close()
}

// Discos pro .m3u (utils/m3u.py): a folha (.cue/.gdi...) é o disco; as
// faixas que ela aponta (.bin...) não.
var (
	descriptorExts = []string{"cue", "gdi", "ccd", "mds"}
	companionExts  = []string{"bin", "raw", "img", "sub", "mdf", "wav", "ogg", "flac", "mp3"}
	discTagRe      = regexp.MustCompile(`(?i)\((?:disc|disk|cd|disque)\s*([0-9]+)`)
)

// m3u devolve o .m3u gerado pros arquivos (nil se não precisa: já tem um,
// ou nenhum é disco).
func m3u(names []string) []byte {
	var discs []string
	hasDescriptor := false
	for _, n := range names {
		ext := strings.ToLower(nameExt(n))
		if ext == "m3u" {
			return nil
		}
		discs = append(discs, n)
		hasDescriptor = hasDescriptor || slices.Contains(descriptorExts, ext)
	}
	if hasDescriptor {
		discs = slices.DeleteFunc(discs, func(n string) bool { return slices.Contains(companionExts, strings.ToLower(nameExt(n))) })
	}
	if len(discs) == 0 {
		return nil
	}
	num := func(n string) int {
		if m := discTagRe.FindStringSubmatch(n); m != nil {
			v, _ := strconv.Atoi(m[1])
			return v
		}
		return 0
	}
	slices.SortStableFunc(discs, func(x, y string) int {
		if a, b := num(x), num(y); a != b {
			return a - b
		}
		return strings.Compare(x, y)
	})
	return []byte(strings.Join(discs, "\n"))
}

// romContent é o GET/HEAD /api/roms/{id}/content/{file_name}: um arquivo
// vai direto (com Range); vários viram zip.
func (a *api) romContent(w http.ResponseWriter, r *http.Request, c *caller) {
	rom, _, ok := a.romItem(w, r)
	if !ok {
		return
	}
	files := sortedFiles(rom)
	q := r.URL.Query()
	if v := q.Get("file_ids"); v != "" {
		want, err := queryInts(r, "file_ids")
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		rels := make([]string, len(files))
		for i, f := range files {
			rels[i] = f.Rel
		}
		ids, err := a.fileIDs(r.Context(), rom.RID, rels)
		if err != nil {
			internal(w, err)
			return
		}
		files = slices.DeleteFunc(files, func(f store.RomFile) bool { return !slices.Contains(want, ids[f.Rel]) })
	}
	if names := queryStrings(r, "files"); len(names) > 0 { // versões antigas: por nome
		files = slices.DeleteFunc(files, func(f store.RomFile) bool { return !slices.Contains(names, f.Name) })
	}
	if len(files) == 0 {
		fail(w, http.StatusNotFound, "No files found for ROM "+r.PathValue("id"))
		return
	}
	if len(files) == 1 {
		p, ok := fileOnDisk(rom, files[0])
		if !ok {
			fail(w, http.StatusNotFound, "File not found")
			return
		}
		a.serveFile(w, r, p, files[0].Name)
		return
	}
	paths := pathsOf(rom)
	hidden, _ := queryBool(r, "hidden_folder")
	zipName := r.PathValue("file_name")
	var entries []zipEntry
	var names []string
	mod := time.Unix(rom.MTime, 0)
	for _, f := range files {
		p, ok := fileOnDisk(rom, f)
		if !ok {
			fail(w, http.StatusNotFound, "File not found")
			return
		}
		// como file_name_for_download: relativo à pasta da ROM
		name := strings.TrimPrefix(paths.fileFull(f), paths.fullPath+"/")
		if hidden != nil && *hidden {
			name = ".hidden/" + name
		}
		names = append(names, name)
		entries = append(entries, zipEntry{name: name, file: p, mod: mod})
	}
	if list := m3u(names); list != nil {
		entries = append(entries, zipEntry{name: zipName + ".m3u", data: list, mod: mod})
	}
	serveZip(w, r, zipName+".zip", entries)
}

// romFileContent é o GET /api/roms/{id}/files/content/{file_name}: o {id}
// é o do arquivo.
func (a *api) romFileContent(w http.ResponseWriter, r *http.Request, c *caller) {
	rom, f, ok := a.fileItem(w, r)
	if !ok {
		return
	}
	p, ok := fileOnDisk(rom, f)
	if !ok {
		fail(w, http.StatusNotFound, "File not found")
		return
	}
	a.serveFile(w, r, p, f.Name)
}

// bulkDownload é o GET /api/roms/download?rom_ids=1,2 (ou platform_id):
// tudo num zip, cada arquivo com o caminho da biblioteca.
func (a *api) bulkDownload(w http.ResponseWriter, r *http.Request, c *caller) {
	ctx := r.Context()
	var roms []*store.Item
	if pid := r.URL.Query().Get("platform_id"); pid != "" {
		ids, err := queryInts(r, "platform_id")
		if err != nil || len(ids) != 1 {
			fail(w, http.StatusUnprocessableEntity, "Invalid platform_id")
			return
		}
		roms, err = a.d.Store.Items(ctx, `SELECT `+store.ItemColumns+` FROM items i WHERE i.kind = ? AND i.parent_id IN
(SELECT id FROM items WHERE kind = ? AND rid = ?) ORDER BY i.sort_name`, store.KindRom, store.KindPlatform, ids[0])
		if err != nil {
			internal(w, err)
			return
		}
	} else if r.URL.Query().Get("rom_ids") != "" {
		ids, err := queryInts(r, "rom_ids")
		if err != nil {
			fail(w, http.StatusBadRequest, "Invalid ROM ID")
			return
		}
		for _, id := range ids {
			if it, err := a.itemOfKind(ctx, id, store.KindRom); err == nil {
				roms = append(roms, it)
			}
		}
	} else {
		fail(w, http.StatusBadRequest, "No ROM IDs or platform/collection selector provided")
		return
	}
	if len(roms) == 0 {
		fail(w, http.StatusNotFound, "No ROMs found to download")
		return
	}
	var entries []zipEntry
	for _, rom := range roms {
		paths := pathsOf(rom)
		for _, f := range sortedFiles(rom) {
			p, ok := fileOnDisk(rom, f)
			if !ok {
				continue
			}
			entries = append(entries, zipEntry{name: paths.fileFull(f), file: p, mod: time.Unix(rom.MTime, 0)})
		}
	}
	name := sanitizeName(r.URL.Query().Get("filename"))
	if name == "" {
		name = fmt.Sprintf("%d ROMs.zip", len(roms))
	}
	serveZip(w, r, name, entries)
}

// cover é a capa da ROM em /assets/romm/resources/roms/{plat}/{rom}/cover/{small|big}.png.
func (a *api) cover(w http.ResponseWriter, r *http.Request, c *caller) {
	pid, err1 := strconv.ParseInt(r.PathValue("platform_id"), 10, 64)
	rid, err2 := strconv.ParseInt(r.PathValue("rom_id"), 10, 64)
	if err1 != nil || err2 != nil {
		fail(w, http.StatusNotFound, "Not Found")
		return
	}
	ctx := r.Context()
	rom, err := a.itemOfKind(ctx, rid, store.KindRom)
	if err != nil || rom.Image == "" {
		fail(w, http.StatusNotFound, "Not Found")
		return
	}
	if p, err := a.platformOf(ctx, rom); err != nil || p.RID != pid {
		fail(w, http.StatusNotFound, "Not Found")
		return
	}
	opts := media.ImageOpts{}
	if strings.HasPrefix(r.PathValue("size"), "small") {
		opts.MaxWidth = 300
	} else {
		opts.MaxWidth = 900
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	a.d.Media.Image(w, r, rom.Image, opts)
}
