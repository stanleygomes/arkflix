package library

import "strings"

// platformNames: slug do RomM → nome que ele mostra (nomes do IGDB).
var platformNames = map[string]string{
	"3do": "3DO Interactive Multiplayer", "amiga": "Amiga", "amiga-cd32": "Amiga CD32",
	"acpc": "Amstrad CPC", "appleii": "Apple II", "arcade": "Arcade", "mame": "MAME",
	"atari2600": "Atari 2600", "atari5200": "Atari 5200", "atari7800": "Atari 7800",
	"atari8bit": "Atari 8-bit", "atari-st": "Atari ST/STE", "jaguar": "Atari Jaguar", "lynx": "Atari Lynx",
	"c64": "Commodore C64/128/MAX", "colecovision": "ColecoVision", "intellivision": "Intellivision",
	"vectrex": "Vectrex", "odyssey-2": "Odyssey 2 / Videopac G7000",
	"dc": "Dreamcast", "dos": "DOS", "win": "Windows", "xbox": "Xbox",
	"nes": "Nintendo Entertainment System", "famicom": "Family Computer", "fds": "Family Computer Disk System",
	"snes": "Super Nintendo Entertainment System", "sfam": "Super Famicom", "n64": "Nintendo 64",
	"ngc": "Nintendo GameCube", "wii": "Wii", "wiiu": "Wii U", "switch": "Nintendo Switch",
	"gb": "Game Boy", "gbc": "Game Boy Color", "gba": "Game Boy Advance", "nds": "Nintendo DS",
	"3ds": "Nintendo 3DS", "virtualboy": "Virtual Boy", "pokemon-mini": "Pokémon mini",
	"sms": "Sega Master System/Mark III", "genesis": "Sega Mega Drive/Genesis", "segacd": "Sega CD",
	"sega32": "Sega 32X", "saturn": "Sega Saturn", "gamegear": "Sega Game Gear", "sg1000": "SG-1000",
	"psx": "PlayStation", "ps2": "PlayStation 2", "ps3": "PlayStation 3", "psp": "PlayStation Portable",
	"psvita": "PlayStation Vita", "neogeoaes": "Neo Geo AES", "neogeomvs": "Neo Geo MVS", "neo-geo-cd": "Neo Geo CD",
	"neo-geo-pocket": "Neo Geo Pocket", "neo-geo-pocket-color": "Neo Geo Pocket Color",
	"tg16": "TurboGrafx-16/PC Engine", "turbografx-cd": "Turbografx-16/PC Engine CD",
	"supergrafx": "PC Engine SuperGrafx", "pc-fx": "PC-FX",
	"wonderswan": "WonderSwan", "wonderswan-color": "WonderSwan Color",
	"msx": "MSX", "msx2": "MSX2", "zxs": "ZX Spectrum", "sharp-x68000": "Sharp X68000",
}

// platformAliases: nomes de pasta (Batocera, ES-DE, RetroBat) → slug.
var platformAliases = map[string]string{
	"amiga500": "amiga", "amiga600": "amiga", "amiga1200": "amiga", "amiga4000": "amiga",
	"amigacd32": "amiga-cd32", "amstradcpc": "acpc", "apple2": "appleii",
	"atarijaguar": "jaguar", "atarilynx": "lynx", "atarist": "atari-st",
	"cps": "arcade", "cps1": "arcade", "cps2": "arcade", "cps3": "arcade", "fbneo": "arcade",
	"dreamcast": "dc", "gamecube": "ngc", "gc": "ngc", "n3ds": "3ds",
	"mastersystem": "sms", "mark3": "sms", "megadrive": "genesis", "megadrivejp": "genesis",
	"genesis-slash-megadrive": "genesis", "megacd": "segacd", "sega32x": "sega32", "sg-1000": "sg1000",
	"neogeo": "neogeoaes", "neogeocd": "neo-geo-cd", "ngp": "neo-geo-pocket", "ngpc": "neo-geo-pocket-color",
	"pce": "tg16", "pcengine": "tg16", "pcenginecd": "turbografx-cd", "tg-cd": "turbografx-cd", "pcfx": "pc-fx",
	"pokemini": "pokemon-mini", "ps": "psx", "ps1": "psx", "sfc": "sfam", "snes-msu1": "snes",
	"wswan": "wonderswan", "wswanc": "wonderswan-color", "wonderswancolor": "wonderswan-color",
	"msx1": "msx", "x68000": "sharp-x68000", "zxspectrum": "zxs", "pc": "win", "windows": "win",
	"odyssey2": "odyssey-2", "wiiware": "wii",
}

// platformName dá o nome do console pelo nome da pasta; desconhecido vira
// o próprio nome da pasta.
func platformName(slug string) string {
	s := strings.ToLower(slug)
	if n, ok := platformNames[s]; ok {
		return n
	}
	if a, ok := platformAliases[s]; ok {
		return platformNames[a]
	}
	return slug
}
