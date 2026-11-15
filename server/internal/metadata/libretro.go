package metadata

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// libretroSystems: slug do RomM → nome da playlist no libretro-thumbnails.
var libretroSystems = map[string]string{
	"3do": "The 3DO Company - 3DO", "amiga": "Commodore - Amiga", "amiga-cd32": "Commodore - CD32",
	"acpc": "Amstrad - CPC", "appleii": "Apple - II", "arcade": "MAME", "mame": "MAME",
	"atari2600": "Atari - 2600", "atari5200": "Atari - 5200", "atari7800": "Atari - 7800",
	"atari8bit": "Atari - 8-bit", "atari-st": "Atari - ST", "jaguar": "Atari - Jaguar", "lynx": "Atari - Lynx",
	"c64": "Commodore - 64", "colecovision": "Coleco - ColecoVision", "intellivision": "Mattel - Intellivision",
	"vectrex": "GCE - Vectrex", "odyssey-2": "Magnavox - Odyssey2",
	"dc": "Sega - Dreamcast", "dos": "DOS", "xbox": "Microsoft - Xbox",
	"nes": "Nintendo - Nintendo Entertainment System", "famicom": "Nintendo - Nintendo Entertainment System",
	"fds":  "Nintendo - Family Computer Disk System",
	"snes": "Nintendo - Super Nintendo Entertainment System", "sfam": "Nintendo - Super Nintendo Entertainment System",
	"n64": "Nintendo - Nintendo 64", "ngc": "Nintendo - GameCube", "wii": "Nintendo - Wii",
	"wiiu": "Nintendo - Wii U", "switch": "Nintendo - Switch",
	"gb": "Nintendo - Game Boy", "gbc": "Nintendo - Game Boy Color", "gba": "Nintendo - Game Boy Advance",
	"nds": "Nintendo - Nintendo DS", "3ds": "Nintendo - Nintendo 3DS", "virtualboy": "Nintendo - Virtual Boy",
	"pokemon-mini": "Nintendo - Pokemon Mini",
	"sms":          "Sega - Master System - Mark III", "genesis": "Sega - Mega Drive - Genesis",
	"segacd": "Sega - Mega-CD - Sega CD", "sega32": "Sega - 32X", "saturn": "Sega - Saturn",
	"gamegear": "Sega - Game Gear", "sg1000": "Sega - SG-1000",
	"psx": "Sony - PlayStation", "ps2": "Sony - PlayStation 2", "ps3": "Sony - PlayStation 3",
	"psp": "Sony - PlayStation Portable", "psvita": "Sony - PlayStation Vita",
	"neogeoaes": "SNK - Neo Geo", "neogeomvs": "SNK - Neo Geo", "neo-geo-cd": "SNK - Neo Geo CD",
	"neo-geo-pocket": "SNK - Neo Geo Pocket", "neo-geo-pocket-color": "SNK - Neo Geo Pocket Color",
	"tg16": "NEC - PC Engine - TurboGrafx 16", "turbografx-cd": "NEC - PC Engine CD - TurboGrafx-CD",
	"supergrafx": "NEC - PC Engine SuperGrafx", "pc-fx": "NEC - PC-FX",
	"wonderswan": "Bandai - WonderSwan", "wonderswan-color": "Bandai - WonderSwan Color",
	"msx": "Microsoft - MSX", "msx2": "Microsoft - MSX2", "zxs": "Sinclair - ZX Spectrum",
	"sharp-x68000": "Sharp - X68000",
}

// libretroSafe troca os caracteres que o libretro não usa em nome de
// arquivo (&*/:`<>?\|") por "_".
var libretroSafe = strings.NewReplacer("&", "_", "*", "_", "/", "_", ":", "_", "`", "_",
	"<", "_", ">", "_", "?", "_", "\\", "_", "|", "_", `"`, "_")

var (
	reSquare   = regexp.MustCompile(`\s*\[[^\]]*\]`)
	reFirstTag = regexp.MustCompile(`^([^(]*\([^)]*\))`)
)

// libretroRom: capa (Named_Boxarts) e fundo (Named_Snaps, Named_Titles)
// pelo nome No-Intro/Redump da ROM.
func (f *Fetcher) libretroRom(ctx context.Context, it *store.Item) (*patch, error) {
	sys := libretroSystems[library.PlatformSlug(it.Platform)]
	if sys == "" {
		return nil, fmt.Errorf("%w: console %q sem nome no libretro", errNotFound, it.Platform)
	}
	names := romNames(it)
	base := f.urls.libretro + "/" + url.PathEscape(sys) + "/"
	at := func(dir string) []string {
		var out []string
		for _, n := range names {
			out = append(out, base+dir+"/"+url.PathEscape(libretroSafe.Replace(n))+".png")
		}
		return out
	}
	return &patch{source: "libretro", needImage: true, images: map[string][]string{
		"image":    at("Named_Boxarts"),
		"backdrop": append(at("Named_Snaps"), at("Named_Titles")...),
	}}, nil
}

// romNames são os nomes a tentar: o do arquivo, sem [tags], só com a
// primeira (região) e o título com as regiões mais comuns.
func romNames(it *store.Item) []string {
	fs, _ := it.Extra["fs_name"].(string)
	if fs == "" {
		fs = filepath.Base(it.Path)
	}
	// ROM de arquivo único: tira a extensão; pasta (multidisco) fica igual
	if files := it.Files(); len(files) == 1 && files[0].Rel == fs {
		fs = strings.TrimSuffix(fs, filepath.Ext(fs))
	}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		for _, o := range out {
			if o == s {
				return
			}
		}
		if s != "" {
			out = append(out, s)
		}
	}
	add(fs)
	noSquare := reSquare.ReplaceAllString(fs, "")
	add(noSquare)
	if m := reFirstTag.FindString(noSquare); m != "" {
		add(m)
	}
	for _, r := range []string{"USA", "World", "USA, Europe", "Europe", "Japan"} {
		add(it.Name + " (" + r + ")")
	}
	return out
}
