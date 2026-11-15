package library

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reBrackets  = regexp.MustCompile(`\[[^\]]*\]|\{[^}]*\}`)
	reParenYear = regexp.MustCompile(`\(((?:18|19|20)\d{2})\)`)
	reYear      = regexp.MustCompile(`(?:^|[ ._(-])((?:19|20)\d{2})(?:$|[ ._)-])`)
	reJunk      = regexp.MustCompile(`(?i)(?:^|[ ._-])(2160p|1080p|1080i|720p|576p|480p|4k|uhd|hdr10?|dv|bluray|blu-ray|bdrip|brrip|web-?dl|web-?rip|webrip|hdtv|dvdrip|dvd|x26[45]|h\.?26[45]|hevc|avc|xvid|divx|aac\d?|ac3|e-?ac3|ddp?5\.1|dts|remux|10bit|proper|repack|extended|unrated)(?:$|[ ._-])`)
	reSpaces    = regexp.MustCompile(`\s+`)
)

// parseTitle tira o nome e o ano de "Nome (Ano)" ou "Nome.Ano.1080p...",
// com o lixo de release (resolução, codec, [tags]) fora.
func parseTitle(s string) (string, int) {
	s = reBrackets.ReplaceAllString(s, " ")
	if !strings.Contains(s, " ") && (strings.Contains(s, ".") || strings.Contains(s, "_")) {
		s = strings.NewReplacer(".", " ", "_", " ").Replace(s)
	}
	year := 0
	if m := reParenYear.FindStringSubmatchIndex(s); m != nil {
		year, _ = strconv.Atoi(s[m[2]:m[3]])
		s = s[:m[0]]
	} else if m := lastYear(s); m != nil {
		year, _ = strconv.Atoi(s[m[2]:m[3]])
		s = s[:m[2]]
	}
	if m := reJunk.FindStringIndex(s); m != nil && m[0] > 0 {
		s = s[:m[0]]
	}
	return cleanName(s), year
}

// lastYear acha o último ano solto que não seja o começo do nome
// ("1917.2019.1080p" → 2019).
func lastYear(s string) []int {
	var last []int
	for _, m := range reYear.FindAllStringSubmatchIndex(s, -1) {
		if m[2] > 0 {
			last = m
		}
	}
	return last
}

// cleanName junta espaços e tira separadores das pontas.
func cleanName(s string) string {
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.Trim(s, " -._()")
}

var (
	reSxxEyy    = regexp.MustCompile(`(?i)\bs(\d{1,3})[ ._]?e(\d{1,4})(?:[ ._-]?e\d{1,4})*`)
	reNxNN      = regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{2,3})\b`)
	reLeadNum   = regexp.MustCompile(`(?i)^(?:e|ep|episode|episódio|episodio)?[ ._]?(\d{1,3})\b`)
	reSeasonDir = regexp.MustCompile(`(?i)^(?:season|temporada|series|staffel|saison|s)[ ._-]*(\d{1,3})$`)
)

// seasonDir diz se a pasta é de temporada e o número (Specials = 0).
func seasonDir(name string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "specials", "especiais", "extras especiais", "season 00", "temporada 00":
		return 0, true
	}
	if m := reSeasonDir.FindStringSubmatch(strings.TrimSpace(name)); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, true
	}
	return 0, false
}

// episodeInfo tira temporada, episódio e título do nome do arquivo.
// season = -1 quando o nome não traz a temporada.
func episodeInfo(base string) (season, ep int, title string, ok bool) {
	for _, re := range []*regexp.Regexp{reSxxEyy, reNxNN} {
		if m := re.FindStringSubmatchIndex(base); m != nil {
			season, _ = strconv.Atoi(base[m[2]:m[3]])
			ep, _ = strconv.Atoi(base[m[4]:m[5]])
			return season, ep, episodeTitle(base[m[1]:]), true
		}
	}
	if m := reLeadNum.FindStringSubmatchIndex(base); m != nil {
		ep, _ = strconv.Atoi(base[m[2]:m[3]])
		return -1, ep, episodeTitle(base[m[1]:]), true
	}
	return 0, 0, "", false
}

func episodeTitle(rest string) string {
	rest = reBrackets.ReplaceAllString(rest, " ")
	if !strings.Contains(strings.TrimSpace(rest), " ") {
		rest = strings.NewReplacer(".", " ", "_", " ").Replace(rest)
	}
	if m := reJunk.FindStringIndex(rest); m != nil {
		rest = rest[:m[0]]
	}
	return cleanName(rest)
}

// seasonName é o nome da temporada que os clientes mostram.
func seasonName(n int) string {
	if n == 0 {
		return "Especiais"
	}
	return "Temporada " + strconv.Itoa(n)
}

// lang2to3 converte ISO 639-1 pro 639-2/B (o que o ffprobe e o Jellyfin usam).
var lang2to3 = map[string]string{
	"pt": "por", "en": "eng", "es": "spa", "fr": "fre", "de": "ger", "it": "ita", "ja": "jpn",
	"ko": "kor", "zh": "chi", "ru": "rus", "nl": "dut", "sv": "swe", "no": "nor", "nb": "nob",
	"da": "dan", "fi": "fin", "pl": "pol", "tr": "tur", "ar": "ara", "he": "heb", "hi": "hin",
	"el": "gre", "cs": "cze", "hu": "hun", "ro": "rum", "uk": "ukr", "th": "tha", "vi": "vie",
	"id": "ind", "ca": "cat", "gl": "glg", "eu": "baq", "hr": "hrv", "sr": "srp", "bg": "bul",
	"pt-br": "por", "pt_br": "por", "ptbr": "por", "pob": "por", "es-419": "spa", "en-us": "eng", "en-gb": "eng",
}

var langNames = map[string]string{
	"portuguese": "por", "portugues": "por", "português": "por", "english": "eng", "ingles": "eng",
	"inglês": "eng", "spanish": "spa", "espanhol": "spa", "french": "fre", "german": "ger",
	"italian": "ita", "japanese": "jpn",
}

// subLang interpreta um pedaço do nome da legenda: idioma ou "".
func subLang(part string) string {
	p := strings.ToLower(part)
	if l, ok := lang2to3[p]; ok {
		return l
	}
	if l, ok := langNames[p]; ok {
		return l
	}
	if len(p) == 3 && isAlpha(p) {
		return p
	}
	return ""
}

func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
