package library

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// episode é um episódio achado, antes de gravar.
type episode struct {
	f       mediaFile
	season  int
	num     int
	title   string
	seasonP string // pasta da temporada ("" = solto na pasta da série)
}

// shows: cada pasta na raiz é uma série; episódios em "Season NN/" ou
// soltos com SxxEyy no nome.
func (s *scanner) shows() error {
	ents, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.IsDir() || skip(e.Name()) {
			continue
		}
		if err := s.series(filepath.Join(s.root, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s *scanner) series(dir string) error {
	files, err := s.videoFiles(dir)
	if err != nil {
		return err
	}
	eps := s.episodes(dir, files)
	if len(eps) == 0 {
		return nil
	}
	ser := &store.Item{Kind: store.KindSeries, Path: dir, RelPath: s.rel(dir)}
	ser.Name, ser.Year = parseTitle(filepath.Base(dir))
	if ser.Name == "" {
		ser.Name = filepath.Base(dir)
	}
	a := s.findArt(dir, "", true)
	ser.Image, ser.Backdrop, ser.Logo = a.image, a.backdrop, a.logo
	readNFO(filepath.Join(dir, "tvshow.nfo")).apply(ser)
	ser.ID = store.ID(store.KindSeries, dir)
	if err := s.put(ser); err != nil {
		return err
	}
	seasons, err := s.seasons(ser, eps)
	if err != nil {
		return err
	}
	probes := s.probeNew(files)
	for _, ep := range eps {
		if err := s.episode(ser, seasons[ep.season], ep, probes); err != nil {
			return err
		}
	}
	return nil
}

// episodes interpreta os nomes; arquivo sem número de episódio fica de fora.
func (s *scanner) episodes(dir string, files []mediaFile) []episode {
	var out []episode
	for _, f := range files {
		parent := filepath.Dir(f.path)
		dirSeason, inSeason := seasonDir(filepath.Base(parent))
		inSeason = inSeason && parent != dir
		season, num, title, ok := episodeInfo(stem(f.info.Name()))
		if !ok || (season < 0 && !inSeason) {
			continue
		}
		ep := episode{f: f, season: season, num: num, title: title}
		if inSeason {
			ep.season, ep.seasonP = dirSeason, parent
		}
		if ep.title == "" {
			ep.title = "Episódio " + strconv.Itoa(num)
		}
		out = append(out, ep)
	}
	return out
}

// seasons grava as temporadas que têm episódio. A pasta real ganha da
// virtual (episódio solto na pasta da série).
func (s *scanner) seasons(ser *store.Item, eps []episode) (map[int]*store.Item, error) {
	paths := map[int]string{}
	for _, ep := range eps {
		if p, ok := paths[ep.season]; !ok || (ep.seasonP != "" && (p == "" || ep.seasonP < p)) {
			paths[ep.season] = ep.seasonP
		}
	}
	nums := make([]int, 0, len(paths))
	for n := range paths {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	out := map[int]*store.Item{}
	for _, n := range nums {
		it, err := s.season(ser, n, paths[n])
		if err != nil {
			return nil, err
		}
		out[n] = it
	}
	return out, nil
}

func (s *scanner) season(ser *store.Item, n int, dir string) (*store.Item, error) {
	path := dir
	if path == "" {
		path = filepath.Join(ser.Path, fmt.Sprintf("Season %02d", n))
	}
	it := &store.Item{Kind: store.KindSeason, Path: path, RelPath: s.rel(path), Name: seasonName(n),
		Index: n, ParentID: ser.ID, SeriesID: ser.ID}
	if dir != "" {
		it.Image = s.findArt(dir, "", true).image
	}
	if it.Image == "" {
		tag := fmt.Sprintf("season%02d", n)
		if n == 0 {
			tag = "season-specials"
		}
		it.Image = s.find(ser.Path, []string{tag + "-poster", tag}, imageExts)
	}
	return it, s.put(it)
}

func (s *scanner) episode(ser, season *store.Item, ep episode, probes map[string]*probed) error {
	it := &store.Item{Kind: store.KindEpisode, Name: ep.title, Index: ep.num, ParentIndex: season.Index,
		ParentID: season.ID, SeriesID: ser.ID, SeasonID: season.ID}
	s.fileItem(it, ep.f, probes)
	s.addSubs(it)
	dir, base := filepath.Dir(ep.f.path), stem(ep.f.info.Name())
	it.Image = s.find(dir, []string{base, base + "-thumb"}, imageExts)
	readNFO(filepath.Join(dir, base+".nfo")).apply(it)
	return s.put(it)
}
