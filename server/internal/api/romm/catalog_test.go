package romm_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

type platformJSON struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	FsSlug      string `json:"fs_slug"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	RomCount    int64  `json:"rom_count"`
	FsSize      int64  `json:"fs_size_bytes"`
	CustomName  any    `json:"custom_name"`
	IGDBID      any    `json:"igdb_id"`
	Firmware    []struct {
		ID         int64  `json:"id"`
		PlatformID int64  `json:"platform_id"`
		FileName   string `json:"file_name"`
		FilePath   string `json:"file_path"`
		FullPath   string `json:"full_path"`
		Size       int64  `json:"file_size_bytes"`
		CreatedAt  string `json:"created_at"`
	} `json:"firmware"`
	MissingFromFS bool   `json:"missing_from_fs"`
	CreatedAt     string `json:"created_at"`
}

func TestPlatformsAndFirmware(t *testing.T) {
	e := newEnv(t)
	tok := e.nina()
	var plats []platformJSON
	r := e.get("/api/platforms", tok)
	r.json(t, &plats)
	if r.status != http.StatusOK || len(plats) != 2 {
		t.Fatalf("platforms: %s", r.body)
	}
	by := map[string]platformJSON{}
	for _, p := range plats {
		by[p.FsSlug] = p
		if !isoRe.MatchString(p.CreatedAt) || p.IGDBID != nil || p.DisplayName != p.Name || p.Slug != p.FsSlug {
			t.Fatalf("platform: %+v", p)
		}
	}
	snes, psx := by["snes"], by["psx"]
	if snes.ID != e.rid(e.ids.SNES) || snes.RomCount != 2 || psx.RomCount != 1 || snes.FsSize == 0 || len(snes.Firmware) != 0 {
		t.Fatalf("contagem: %+v", plats)
	}
	if len(psx.Firmware) != 1 || psx.Firmware[0].FileName != "scph1001.bin" || psx.Firmware[0].FilePath != "bios/psx" ||
		psx.Firmware[0].PlatformID != psx.ID || psx.Firmware[0].ID != e.rid(e.ids.BiosPSX) || psx.Firmware[0].Size == 0 {
		t.Fatalf("firmware: %+v", psx.Firmware)
	}
	var one platformJSON
	r = e.get("/api/platforms/"+itoa(psx.ID), tok)
	r.json(t, &one)
	if one.ID != psx.ID || len(one.Firmware) != 1 {
		t.Fatalf("platform: %s", r.body)
	}
	if r := e.get("/api/platforms/"+itoa(e.rid(e.ids.RomSuper)), tok); r.status != http.StatusNotFound {
		t.Fatalf("rom como console: %d", r.status)
	}
	if r := e.get("/api/platforms/abc", tok); r.status != http.StatusUnprocessableEntity {
		t.Fatalf("id ruim: %d", r.status)
	}

	var fws []map[string]any
	r = e.get("/api/firmware?platform_id="+itoa(snes.ID), tok)
	r.json(t, &fws)
	if len(fws) != 0 {
		t.Fatalf("firmware snes: %s", r.body)
	}
	r = e.get("/api/firmware", tok)
	r.json(t, &fws)
	if len(fws) != 1 || fws[0]["md5_hash"] != "" || fws[0]["is_verified"] != false {
		t.Fatalf("firmware: %s", r.body)
	}
	fid := itoa(e.rid(e.ids.BiosPSX))
	if r := e.get("/api/firmware/"+fid, tok); r.status != http.StatusOK {
		t.Fatalf("firmware id: %d", r.status)
	}
	r = e.get("/api/firmware/"+fid+"/content/scph1001.bin", tok)
	if r.status != http.StatusOK || string(r.body) != "rom de teste: jogos/bios/psx/scph1001.bin" ||
		!strings.Contains(r.header.Get("Content-Disposition"), "scph1001.bin") {
		t.Fatalf("download bios: %d %q %v", r.status, r.body, r.header)
	}
	if r := e.get("/api/firmware/"+itoa(e.rid(e.ids.RomSuper))+"/content/x", tok); r.status != http.StatusNotFound {
		t.Fatalf("rom como bios: %d", r.status)
	}
}

type romJSON struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	FsName           string   `json:"fs_name"`
	FsNameNoTags     string   `json:"fs_name_no_tags"`
	FsNameNoExt      string   `json:"fs_name_no_ext"`
	FsExtension      string   `json:"fs_extension"`
	FsPath           string   `json:"fs_path"`
	FullPath         string   `json:"full_path"`
	FsSize           int64    `json:"fs_size_bytes"`
	PlatformID       int64    `json:"platform_id"`
	PlatformFsSlug   string   `json:"platform_fs_slug"`
	PlatformDisplay  string   `json:"platform_display_name"`
	Regions          []string `json:"regions"`
	Languages        []string `json:"languages"`
	SimpleSingle     bool     `json:"has_simple_single_file"`
	NestedSingle     bool     `json:"has_nested_single_file"`
	Multiple         bool     `json:"has_multiple_files"`
	PathCoverSmall   string   `json:"path_cover_small"`
	PathCoverLarge   string   `json:"path_cover_large"`
	IGDBID           any      `json:"igdb_id"`
	Summary          any      `json:"summary"`
	MissingFromFS    bool     `json:"missing_from_fs"`
	CreatedAt        string   `json:"created_at"`
	SiblingRoms      []any    `json:"sibling_roms"`
	MergedScreenshot []string `json:"merged_screenshots"`
	Metadatum        struct {
		Genres  []string `json:"genres"`
		Release *int64   `json:"first_release_date"`
		Players string   `json:"player_count"`
	} `json:"metadatum"`
	RomUser struct {
		ID         int64   `json:"id"`
		Rating     int     `json:"rating"`
		Status     *string `json:"status"`
		LastPlayed *string `json:"last_played"`
	} `json:"rom_user"`
	Files []struct {
		ID         int64   `json:"id"`
		RomID      int64   `json:"rom_id"`
		FileName   string  `json:"file_name"`
		FilePath   string  `json:"file_path"`
		FullPath   string  `json:"full_path"`
		Size       int64   `json:"file_size_bytes"`
		IsTopLevel bool    `json:"is_top_level"`
		Category   *string `json:"category"`
		CRC        any     `json:"crc_hash"`
	} `json:"files"`
	UserSaves  []map[string]any `json:"user_saves"`
	UserStates []map[string]any `json:"user_states"`
}

type pageJSON struct {
	Items      []romJSON        `json:"items"`
	Total      *int64           `json:"total"`
	Limit      int64            `json:"limit"`
	Offset     int64            `json:"offset"`
	CharIndex  map[string]int   `json:"char_index"`
	RomIDIndex []int64          `json:"rom_id_index"`
	Filters    map[string][]any `json:"filter_values"`
}

func (e *env) page(path, tok string) pageJSON {
	e.t.Helper()
	r := e.get(path, tok)
	if r.status != http.StatusOK {
		e.t.Fatalf("%s: %d %s", path, r.status, r.body)
	}
	var p pageJSON
	r.json(e.t, &p)
	return p
}

func names(p pageJSON) []string {
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Name)
	}
	return out
}

func TestRomsListing(t *testing.T) {
	e := newEnv(t)
	tok := e.nina()
	// a consulta exata do app arkflix
	p := e.page("/api/roms?limit=200&offset=0&with_char_index=false&with_filter_values=false", tok)
	if *p.Total != 3 || len(p.Items) != 3 || p.Limit != 200 || len(p.CharIndex) != 0 || len(p.Filters["regions"]) != 0 {
		t.Fatalf("página: %+v", p)
	}
	if got := names(p); !slices.Equal(got, []string{"Jogo Disco", "Outro Jogo", "Super Teste"}) {
		t.Fatalf("ordem: %v", got)
	}
	disco, outro, super := p.Items[0], p.Items[1], p.Items[2]
	if !disco.Multiple || disco.SimpleSingle || disco.FsName != "Jogo Disco (USA)" || disco.PlatformFsSlug != "psx" ||
		disco.FsPath != "roms/psx" || disco.FullPath != "roms/psx/Jogo Disco (USA)" || len(disco.Files) != 0 {
		t.Fatalf("disco: %+v", disco)
	}
	if !super.SimpleSingle || super.Multiple || super.FsNameNoTags != "Super Teste" || super.FsNameNoExt != "Super Teste (USA)" ||
		super.FsExtension != "sfc" || super.FsSize != int64(len("rom de teste: jogos/roms/snes/Super Teste (USA).sfc")) ||
		super.PlatformID != e.rid(e.ids.SNES) || super.ID != e.rid(e.ids.RomSuper) || super.IGDBID != nil || super.Summary != nil ||
		super.PathCoverLarge != "" || !isoRe.MatchString(super.CreatedAt) || super.RomUser.ID != -1 || super.SiblingRoms == nil {
		t.Fatalf("super: %+v", super)
	}
	if !slices.Equal(outro.Languages, []string{"En", "Fr"}) || !slices.Equal(outro.Regions, []string{"Europe"}) {
		t.Fatalf("outro: %+v", outro)
	}

	// paginação
	p = e.page("/api/roms?limit=1&offset=1", tok)
	if *p.Total != 3 || len(p.Items) != 1 || p.Items[0].Name != "Outro Jogo" || len(p.RomIDIndex) != 3 || p.Offset != 1 {
		t.Fatalf("paginação: %+v", p)
	}
	if p.CharIndex["j"] != 0 || p.CharIndex["o"] != 1 || p.CharIndex["s"] != 2 {
		t.Fatalf("char_index: %v", p.CharIndex)
	}
	if len(p.Filters["regions"]) != 2 || len(p.Filters["platforms"]) != 2 {
		t.Fatalf("filter_values: %v", p.Filters)
	}
	// console, busca, ordem
	snes := itoa(e.rid(e.ids.SNES))
	if got := names(e.page("/api/roms?platform_ids="+snes, tok)); !slices.Equal(got, []string{"Outro Jogo", "Super Teste"}) {
		t.Fatalf("platform_ids: %v", got)
	}
	if got := names(e.page("/api/roms?platform_id="+itoa(e.rid(e.ids.PSX)), tok)); !slices.Equal(got, []string{"Jogo Disco"}) {
		t.Fatalf("platform_id: %v", got)
	}
	if got := names(e.page("/api/roms?search_term=OUTRO", tok)); !slices.Equal(got, []string{"Outro Jogo"}) {
		t.Fatalf("busca: %v", got)
	}
	if got := names(e.page("/api/roms?order_by=name&order_dir=desc", tok)); !slices.Equal(got, []string{"Super Teste", "Outro Jogo", "Jogo Disco"}) {
		t.Fatalf("desc: %v", got)
	}
	p = e.page("/api/roms?order_by=fs_size_bytes&order_dir=desc", tok)
	if p.Items[0].Name != "Jogo Disco" || len(p.CharIndex) != 0 {
		t.Fatalf("por tamanho: %v %v", names(p), p.CharIndex)
	}
	if got := names(e.page("/api/roms?regions=Europe", tok)); !slices.Equal(got, []string{"Outro Jogo"}) {
		t.Fatalf("região: %v", got)
	}
	if got := names(e.page("/api/roms?collection_id=1", tok)); len(got) != 0 {
		t.Fatalf("coleção: %v", got)
	}
	// favorito é do userdata comum (estrela do Jellyfin/Subsonic)
	if _, err := e.k.Store.SetFavorite(context.Background(), e.k.PlainUser.ID, e.ids.RomOutro, true); err != nil {
		t.Fatal(err)
	}
	if got := names(e.page("/api/roms?favorite=true", tok)); !slices.Equal(got, []string{"Outro Jogo"}) {
		t.Fatalf("favorito: %v", got)
	}
	if got := names(e.page("/api/roms?favorite=true", e.admin())); len(got) != 0 {
		t.Fatalf("favorito é por usuário: %v", got)
	}
	p = e.page("/api/roms?with_files=true&platform_id="+itoa(e.rid(e.ids.PSX)), tok)
	if len(p.Items[0].Files) != 4 {
		t.Fatalf("with_files: %+v", p.Items[0])
	}
	for _, bad := range []string{"limit=0", "limit=abc", "offset=-1", "order_dir=up", "favorite=talvez"} {
		if r := e.get("/api/roms?"+bad, tok); r.status != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d", bad, r.status)
		}
	}
	var ids []int64
	e.get("/api/roms/identifiers", tok).json(t, &ids)
	if len(ids) != 3 {
		t.Fatalf("identifiers: %v", ids)
	}
}

func TestRomDetail(t *testing.T) {
	e := newEnv(t)
	tok := e.nina()
	ctx := context.Background()
	// hashes e capa vêm do scanner quando tem
	it, _ := e.k.Store.Item(ctx, e.ids.RomSuper)
	files := it.Files()
	files[0].CRC, files[0].MD5 = "1234abcd", "d41d8cd98f00b204e9800998ecf8427e"
	it.Extra["files"] = files
	it.Image = filepath.Join(e.k.Root, "capa.png")
	it.Year, it.Overview = 1994, "Um jogo"
	if err := e.k.Store.UpsertItem(ctx, it); err != nil {
		t.Fatal(err)
	}
	r := e.get("/api/roms/"+itoa(it.RID), tok)
	var rom romJSON
	r.json(t, &rom)
	var raw map[string]any
	r.json(t, &raw)
	if r.status != http.StatusOK || len(rom.Files) != 1 || rom.Files[0].CRC != "1234abcd" || raw["crc_hash"] != "1234abcd" ||
		raw["md5_hash"] != "d41d8cd98f00b204e9800998ecf8427e" || rom.Summary != "Um jogo" || rom.Metadatum.Release == nil {
		t.Fatalf("detalhe: %s", r.body)
	}
	cover := "/assets/romm/resources/roms/" + itoa(rom.PlatformID) + "/" + itoa(rom.ID) + "/cover/big.png?ts="
	if !strings.HasPrefix(rom.PathCoverLarge, cover) || !strings.Contains(rom.PathCoverSmall, "/cover/small.png?ts=") {
		t.Fatalf("capa: %q %q", rom.PathCoverSmall, rom.PathCoverLarge)
	}
	// campos que clientes estritos exigem
	for _, k := range []string{"user_saves", "user_states", "user_screenshots", "all_user_saves", "all_user_states",
		"user_collections", "all_user_notes", "merged_screenshots", "sibling_roms", "rom_user", "metadatum",
		"igdb_metadata", "ss_metadata", "manual_metadata", "url_cover", "has_manual", "is_identified", "revision",
		"download_formats", "user_channels", "snapshot_save_ids", "channel_file_id"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("detalhe sem %s", k)
		}
	}
	f := rom.Files[0]
	if f.FileName != "Super Teste (USA).sfc" || f.FilePath != "roms/snes" || f.FullPath != "roms/snes/Super Teste (USA).sfc" ||
		!f.IsTopLevel || f.Category == nil || *f.Category != "game" || f.RomID != rom.ID || f.ID == 0 {
		t.Fatalf("arquivo: %+v", f)
	}
	// capa de ROM sem imagem: 404 (a com imagem vai pro media.Image)
	if r := e.get("/assets/romm/resources/roms/"+itoa(e.rid(e.ids.PSX))+"/"+itoa(e.rid(e.ids.RomDisco))+"/cover/big.png", ""); r.status != http.StatusNotFound {
		t.Fatalf("capa sem imagem: %d", r.status)
	}
	if r := e.get("/assets/romm/resources/roms/999/"+itoa(rom.ID)+"/cover/big.png", ""); r.status != http.StatusNotFound {
		t.Fatalf("capa com console errado: %d", r.status)
	}

	r = e.get("/api/roms/"+itoa(e.rid(e.ids.RomDisco)), tok)
	var disco romJSON
	r.json(t, &disco)
	if len(disco.Files) != 4 || !disco.Multiple || disco.SimpleSingle || disco.NestedSingle {
		t.Fatalf("disco: %s", r.body)
	}
	for _, f := range disco.Files {
		if f.FilePath != "roms/psx/Jogo Disco (USA)" || !f.IsTopLevel {
			t.Fatalf("arquivo do disco: %+v", f)
		}
	}
	// ids dos arquivos são estáveis
	var again romJSON
	e.get("/api/roms/"+itoa(e.rid(e.ids.RomDisco)), tok).json(t, &again)
	if again.Files[2].ID != disco.Files[2].ID {
		t.Fatal("id de arquivo mudou")
	}
	var file map[string]any
	e.get("/api/roms/"+itoa(disco.Files[1].ID)+"/files", tok).json(t, &file)
	if file["file_name"] != disco.Files[1].FileName {
		t.Fatalf("arquivo por id: %v", file)
	}
	if r := e.get("/api/roms/"+itoa(e.rid(e.ids.SNES)), tok); r.status != http.StatusNotFound {
		t.Fatalf("console como rom: %d", r.status)
	}
	if r := e.get("/api/roms/99999", tok); r.status != http.StatusNotFound || !strings.Contains(detail(t, r), "99999") {
		t.Fatalf("rom inexistente: %d %s", r.status, r.body)
	}
}

func TestDownloads(t *testing.T) {
	e := newEnv(t)
	tok := e.nina()
	super := itoa(e.rid(e.ids.RomSuper))
	want := "rom de teste: jogos/roms/snes/Super Teste (USA).sfc"
	r := e.get("/api/roms/"+super+"/content/Super%20Teste%20(USA).sfc", tok)
	if r.status != http.StatusOK || string(r.body) != want || r.header.Get("Accept-Ranges") != "bytes" ||
		!strings.Contains(r.header.Get("Content-Disposition"), "Super Teste (USA).sfc") {
		t.Fatalf("download: %d %q %v", r.status, r.body, r.header)
	}
	r = e.do(http.MethodGet, "/api/roms/"+super+"/content/x.sfc", tok, nil, "", "Range", "bytes=5-")
	if r.status != http.StatusPartialContent || string(r.body) != want[5:] {
		t.Fatalf("range: %d %q", r.status, r.body)
	}
	r = e.do(http.MethodHead, "/api/roms/"+super+"/content/x.sfc", tok, nil, "")
	if r.status != http.StatusOK || len(r.body) != 0 || r.header.Get("Content-Length") != itoa(int64(len(want))) {
		t.Fatalf("head: %d %v", r.status, r.header)
	}
	if r := e.get("/api/roms/"+super+"/content/x.sfc", ""); r.status != http.StatusUnauthorized {
		t.Fatalf("sem login: %d", r.status)
	}

	// multidisco: zip na hora, com .m3u dos dois discos
	disco := itoa(e.rid(e.ids.RomDisco))
	r = e.get("/api/roms/"+disco+"/content/Jogo%20Disco%20(USA)", tok)
	if r.status != http.StatusOK || r.header.Get("Content-Type") != "application/zip" ||
		!strings.Contains(r.header.Get("Content-Disposition"), "Jogo Disco (USA).zip") {
		t.Fatalf("zip: %d %v", r.status, r.header)
	}
	zr, err := zip.NewReader(bytes.NewReader(r.body), int64(len(r.body)))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
		if f.Method != zip.Store {
			t.Errorf("%s comprimido", f.Name)
		}
	}
	if len(got) != 5 || got["Jogo Disco (USA) (Disc 2).bin"] != "rom de teste: jogos/roms/psx/Jogo Disco (USA)/Jogo Disco (USA) (Disc 2).bin" ||
		got["Jogo Disco (USA).m3u"] != "Jogo Disco (USA) (Disc 1).cue\nJogo Disco (USA) (Disc 2).cue" {
		t.Fatalf("conteúdo do zip: %v", got)
	}
	if r := e.do(http.MethodHead, "/api/roms/"+disco+"/content/x", tok, nil, ""); r.status != http.StatusOK || len(r.body) != 0 ||
		r.header.Get("Content-Type") != "application/zip" {
		t.Fatalf("head zip: %d %v", r.status, r.header)
	}
	// escolhendo arquivos por id: um só vai direto
	var rom romJSON
	e.get("/api/roms/"+disco, tok).json(t, &rom)
	r = e.get("/api/roms/"+disco+"/content/x?file_ids="+itoa(rom.Files[0].ID), tok)
	if r.status != http.StatusOK || !strings.HasSuffix(string(r.body), "(Disc 1).bin") {
		t.Fatalf("file_ids: %d %q", r.status, r.body)
	}
	r = e.get("/api/roms/"+disco+"/content/x?file_ids="+itoa(rom.Files[0].ID)+","+itoa(rom.Files[1].ID), tok)
	if zr, err := zip.NewReader(bytes.NewReader(r.body), int64(len(r.body))); err != nil || len(zr.File) != 3 {
		t.Fatalf("dois arquivos + m3u: %v", err)
	}
	if r := e.get("/api/roms/"+disco+"/content/x?file_ids=999999", tok); r.status != http.StatusNotFound {
		t.Fatalf("file_ids inexistente: %d", r.status)
	}
	// por arquivo (o app baixa jogo multidisco assim)
	r = e.do(http.MethodGet, "/api/roms/"+itoa(rom.Files[3].ID)+"/files/content/"+rom.Files[3].FileName, tok, nil, "", "Range", "bytes=0-3")
	if r.status != http.StatusPartialContent || string(r.body) != "rom " {
		t.Fatalf("arquivo por id: %d %q", r.status, r.body)
	}
	if r := e.get("/api/roms/999999/files/content/x", tok); r.status != http.StatusNotFound {
		t.Fatalf("arquivo inexistente: %d", r.status)
	}
	// vários jogos num zip
	r = e.get("/api/roms/download?rom_ids="+super+","+disco, tok)
	zr, err = zip.NewReader(bytes.NewReader(r.body), int64(len(r.body)))
	if err != nil || len(zr.File) != 5 || zr.File[0].Name != "roms/snes/Super Teste (USA).sfc" {
		t.Fatalf("bulk: %d %v", r.status, err)
	}
	if r := e.get("/api/roms/download", tok); r.status != http.StatusBadRequest {
		t.Fatalf("bulk sem ids: %d", r.status)
	}
	// arquivo sumiu do disco: 404, não zip quebrado
	os.Remove(filepath.Join(e.k.Config.Games, "roms", "psx", "Jogo Disco (USA)", "Jogo Disco (USA) (Disc 2).cue"))
	if r := e.get("/api/roms/"+disco+"/content/x", tok); r.status != http.StatusNotFound {
		t.Fatalf("arquivo sumido: %d", r.status)
	}
	_ = store.KindRom
}
