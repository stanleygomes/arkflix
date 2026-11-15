package library

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// games: layout do RomM, roms/<slug>/<rom> (arquivo ou pasta multidisco) e
// bios/<slug>/<arquivo>.
func (s *scanner) games() error {
	romsDir := filepath.Join(s.root, "roms")
	ents, err := os.ReadDir(romsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range ents {
		if !e.IsDir() || skip(e.Name()) {
			continue
		}
		if err := s.platform(e.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (s *scanner) platform(slug string) error {
	dir := filepath.Join(s.root, "roms", slug)
	pl := &store.Item{Kind: store.KindPlatform, Path: dir, RelPath: s.rel(dir), Name: platformName(slug), Platform: slug}
	if err := s.put(pl); err != nil {
		return err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if skip(e.Name()) || (!e.IsDir() && imageExt(e.Name())) {
			continue
		}
		if err := s.rom(pl, dir, e); err != nil {
			return err
		}
	}
	return s.firmware(pl)
}

func imageExt(name string) bool {
	switch ext(name) {
	case "jpg", "jpeg", "png", "webp", "gif":
		return true
	}
	return false
}

// rom grava uma ROM: arquivo único ou pasta (arquivos em ordem, tamanho
// somado, mtime do mais novo).
func (s *scanner) rom(pl *store.Item, platDir string, e fs.DirEntry) error {
	path := filepath.Join(platDir, e.Name())
	files, size, mtime, err := s.romFiles(platDir, path, e)
	if err != nil {
		s.warn(path, err)
		return nil
	}
	if len(files) == 0 {
		return nil
	}
	name := e.Name()
	if !e.IsDir() {
		name = stem(name)
	}
	title, regions, langs, tags := parseRom(name)
	it := &store.Item{Kind: store.KindRom, Path: path, RelPath: s.rel(path), Name: title, ParentID: pl.ID,
		Platform: pl.Platform, Size: size, MTime: mtime,
		Extra: map[string]any{"fs_name": e.Name(), "multi": len(files) > 1, "files": files,
			"regions": regions, "languages": langs, "tags": tags}}
	it.Image = s.find(platDir, []string{stem(e.Name()), e.Name()}, imageExts)
	if it.Image == "" && e.IsDir() {
		it.Image = s.find(path, []string{"cover", "boxart", "folder"}, imageExts)
	}
	return s.put(it)
}

// romFiles lista os arquivos da ROM (Rel relativo à pasta do console).
func (s *scanner) romFiles(platDir, path string, e fs.DirEntry) ([]store.RomFile, int64, int64, error) {
	if !e.IsDir() {
		fi, err := e.Info()
		if err != nil {
			return nil, 0, 0, err
		}
		return []store.RomFile{{Name: e.Name(), Rel: e.Name(), Size: fi.Size()}}, fi.Size(), fi.ModTime().Unix(), nil
	}
	var files []store.RomFile
	var size, mtime int64
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != path && skip(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || (filepath.Dir(p) == path && imageExt(d.Name()) && isCoverName(d.Name())) {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(platDir, p)
		files = append(files, store.RomFile{Name: d.Name(), Rel: filepath.ToSlash(rel), Size: fi.Size()})
		size += fi.Size()
		mtime = max(mtime, fi.ModTime().Unix())
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Rel < files[j].Rel })
	return files, size, mtime, err
}

func isCoverName(name string) bool {
	switch strings.ToLower(stem(name)) {
	case "cover", "boxart", "folder":
		return true
	}
	return false
}

// firmware grava bios/<slug>/* do console.
func (s *scanner) firmware(pl *store.Item) error {
	dir := filepath.Join(s.root, "bios", pl.Platform)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil // sem bios: normal
	}
	for _, e := range ents {
		if e.IsDir() || skip(e.Name()) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		p := filepath.Join(dir, e.Name())
		it := &store.Item{Kind: store.KindFirmware, Path: p, RelPath: s.rel(p), Name: e.Name(), ParentID: pl.ID,
			Platform: pl.Platform, Size: fi.Size(), MTime: fi.ModTime().Unix()}
		if err := s.put(it); err != nil {
			return err
		}
	}
	return nil
}

var (
	reRomGroup = regexp.MustCompile(`\(([^)]*)\)|\[([^\]]*)\]`)
	reLangCode = regexp.MustCompile(`^[A-Z][a-z](?:-[A-Za-z]{2,4})?$`)
)

// regiões do No-Intro e os códigos curtos do GoodTools.
var romRegions = map[string]string{
	"usa": "USA", "u": "USA", "europe": "Europe", "e": "Europe", "japan": "Japan", "j": "Japan",
	"world": "World", "w": "World", "brazil": "Brazil", "b": "Brazil", "korea": "Korea", "k": "Korea",
	"china": "China", "c": "China", "australia": "Australia", "a": "Australia", "asia": "Asia",
	"canada": "Canada", "france": "France", "f": "France", "germany": "Germany", "g": "Germany",
	"italy": "Italy", "i": "Italy", "spain": "Spain", "s": "Spain", "netherlands": "Netherlands",
	"sweden": "Sweden", "taiwan": "Taiwan", "hong kong": "Hong Kong", "russia": "Russia", "uk": "UK",
	"scandinavia": "Scandinavia", "latin america": "Latin America", "mexico": "Mexico",
	"argentina": "Argentina", "portugal": "Portugal", "poland": "Poland", "greece": "Greece",
	"denmark": "Denmark", "finland": "Finland", "norway": "Norway", "ue": "USA, Europe", "ju": "Japan, USA",
	"jue": "Japan, USA, Europe",
}

// parseRom tira as tags "(USA)", "(En,Fr)", "[!]" do nome.
func parseRom(name string) (title string, regions, langs, tags []string) {
	regions, langs, tags = []string{}, []string{}, []string{}
	for _, m := range reRomGroup.FindAllStringSubmatch(name, -1) {
		if strings.HasPrefix(m[0], "[") {
			if t := strings.TrimSpace(m[2]); t != "" {
				tags = append(tags, t)
			}
			continue
		}
		parts := splitTrim(m[1])
		switch {
		case len(parts) == 0:
		case allRegions(parts):
			for _, p := range parts {
				regions = appendNew(regions, strings.Split(romRegions[strings.ToLower(p)], ", ")...)
			}
		case allMatch(parts, reLangCode):
			langs = appendNew(langs, parts...)
		default:
			tags = append(tags, strings.TrimSpace(m[1]))
		}
	}
	title = cleanName(reRomGroup.ReplaceAllString(name, " "))
	if title == "" {
		title = name
	}
	return title, regions, langs, tags
}

func splitTrim(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '+' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func allRegions(parts []string) bool {
	for _, p := range parts {
		if _, ok := romRegions[strings.ToLower(p)]; !ok {
			return false
		}
	}
	return true
}

func allMatch(parts []string, re *regexp.Regexp) bool {
	for _, p := range parts {
		if !re.MatchString(p) {
			return false
		}
	}
	return true
}
