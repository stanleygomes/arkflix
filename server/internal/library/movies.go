package library

import (
	"path/filepath"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// movies: um filme por arquivo, solto na raiz ou na própria pasta
// ("Nome (Ano)/Nome (Ano).mkv").
func (s *scanner) movies() error {
	files, err := s.videoFiles(s.root)
	if err != nil {
		return err
	}
	perDir := map[string]int{}
	for _, f := range files {
		perDir[filepath.Dir(f.path)]++
	}
	probes := s.probeNew(files)
	for _, f := range files {
		if err := s.movie(f, perDir[filepath.Dir(f.path)] == 1, probes); err != nil {
			return err
		}
	}
	return nil
}

// movie grava um filme. own = está sozinho numa pasta que é dele.
func (s *scanner) movie(f mediaFile, own bool, probes map[string]*probed) error {
	dir, base := filepath.Dir(f.path), stem(f.info.Name())
	own = own && dir != s.root
	it := &store.Item{Kind: store.KindMovie}
	it.Name, it.Year = parseTitle(base)
	if own {
		name, year := parseTitle(filepath.Base(dir))
		if name != "" {
			it.Name = name
		}
		if year != 0 {
			it.Year = year
		}
	}
	if it.Name == "" {
		it.Name = base
	}
	s.fileItem(it, f, probes)
	s.addSubs(it)
	a := s.findArt(dir, base, own)
	it.Image, it.Backdrop, it.Logo = a.image, a.backdrop, a.logo
	nfos := []string{filepath.Join(dir, base+".nfo")}
	if own {
		nfos = append(nfos, filepath.Join(dir, "movie.nfo"))
	}
	readNFO(nfos...).apply(it)
	return s.put(it)
}
