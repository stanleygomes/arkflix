package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/stanleygomes/arkflix/app/internal/subsonic"
)

// Cópias no PC: "Baixar pro PC" num álbum ou playlist baixa os arquivos
// originais (download do Subsonic) pra ~/Music/arkflix/<Artista>/<Álbum>/
// (playlist: as faixas nas pastas dos álbuns + um .m3u8 em Playlists/). A
// tarefa é cancelável; o que ficou pela metade fica em .part e continua de
// onde parou no próximo "Baixar". O índice (<cache>/music/local.json) guarda
// o que foi baixado de cada servidor, com o nome das faixas: sem internet,
// o álbum baixado abre e toca do disco.

type localTrackEntry struct {
	Path  string     `json:"path"` // relativo à pasta de músicas
	Track MusicTrack `json:"track"`
}

type localSetEntry struct {
	Kind     string   `json:"kind"` // album ou playlist
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Artist   string   `json:"artist"`
	ArtistID string   `json:"artistId"`
	Year     int      `json:"year"`
	Tracks   []string `json:"tracks"`
	M3U      string   `json:"m3u,omitempty"`
	At       int64    `json:"at"`
}

func (e *localSetEntry) album(local bool) MusicAlbum {
	return MusicAlbum{ID: e.ID, Name: e.Name, Artist: e.Artist, ArtistID: e.ArtistID, Year: e.Year, Songs: len(e.Tracks), Local: local}
}

type localIndex struct {
	Tracks map[string]localTrackEntry `json:"tracks"` // servidor|id
	Sets   map[string]*localSetEntry  `json:"sets"`   // servidor|kind|id
}

func localKey(parts ...string) string { return strings.Join(parts, "|") }

func (s *Service) musicDir() string {
	if s.opt.MusicDir != "" {
		return s.opt.MusicDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Music", "arkflix")
}

func (s *Service) localIndexFile() string { return filepath.Join(s.opt.Cache, "music", "local.json") }

// index devolve o índice (carrega na primeira vez); chame com idxMu preso.
func (s *Service) index() *localIndex {
	if s.music.idx != nil {
		return s.music.idx
	}
	idx := &localIndex{Tracks: map[string]localTrackEntry{}, Sets: map[string]*localSetEntry{}}
	if raw, err := os.ReadFile(s.localIndexFile()); err == nil {
		json.Unmarshal(raw, idx)
		if idx.Tracks == nil {
			idx.Tracks = map[string]localTrackEntry{}
		}
		if idx.Sets == nil {
			idx.Sets = map[string]*localSetEntry{}
		}
	}
	s.music.idx = idx
	return idx
}

func (s *Service) saveIndex() error {
	return writePrivate(s.localIndexFile(), s.music.idx)
}

// localTrack acha a cópia baixada da faixa (o arquivo precisa existir).
func (s *Service) localTrack(server, id string) (string, bool) {
	s.music.idxMu.Lock()
	e, ok := s.index().Tracks[localKey(server, id)]
	s.music.idxMu.Unlock()
	if !ok {
		return "", false
	}
	p := filepath.Join(s.musicDir(), e.Path)
	if info, err := os.Stat(p); err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return p, true
}

func (s *Service) localSet(server, kind, id string) *localSetEntry {
	s.music.idxMu.Lock()
	defer s.music.idxMu.Unlock()
	if e, ok := s.index().Sets[localKey(server, kind, id)]; ok {
		cp := *e
		return &cp
	}
	return nil
}

func (s *Service) localTracks(server string, set *localSetEntry) []MusicTrack {
	s.music.idxMu.Lock()
	idx := s.index()
	out := make([]MusicTrack, 0, len(set.Tracks))
	for _, id := range set.Tracks {
		if e, ok := idx.Tracks[localKey(server, id)]; ok {
			out = append(out, e.Track)
		}
	}
	s.music.idxMu.Unlock()
	for i := range out {
		_, out[i].Local = s.localTrack(server, out[i].ID)
		out[i].Cover = ""
	}
	return out
}

// MusicLocal são os álbuns e playlists baixados neste PC (do servidor do
// perfil), do mais novo pro mais velho.
type MusicLocalSet struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Year   int    `json:"year"`
	Songs  int    `json:"songs"`
	Cover  string `json:"cover"`
}

// MusicLocal lista o que está no PC (funciona sem internet).
func (s *Service) MusicLocal() ([]MusicLocalSet, error) {
	l := s.musicLinkOf(s.cur().id)
	if l == nil {
		return nil, errNoMusic
	}
	base := s.musicProxy()
	s.music.idxMu.Lock()
	var sets []*localSetEntry
	for k, e := range s.index().Sets {
		if strings.HasPrefix(k, l.Base+"|") {
			sets = append(sets, e)
		}
	}
	s.music.idxMu.Unlock()
	slices.SortFunc(sets, func(a, b *localSetEntry) int { return int(b.At - a.At) })
	out := make([]MusicLocalSet, 0, len(sets))
	for _, e := range sets {
		c := ""
		if e.Kind == "album" {
			c = cover(base, e.ID, 360)
		}
		out = append(out, MusicLocalSet{Kind: e.Kind, ID: e.ID, Name: e.Name, Artist: e.Artist, Year: e.Year, Songs: len(e.Tracks), Cover: c})
	}
	return out, nil
}

// safeName limpa um nome pra virar pasta ou arquivo (em todo sistema).
func safeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(strings.TrimSpace(name), ".")
	if r := []rune(name); len(r) > 120 {
		name = strings.TrimSpace(string(r[:120]))
	}
	if name == "" {
		return "Desconhecido"
	}
	return name
}

// trackPath: <Artista>/<Álbum>/<NN - Título>.<ext> (relativo).
func trackPath(t subsonic.Song) string {
	ext := strings.ToLower(strings.Trim(t.Suffix, "."))
	if ext == "" || strings.ContainsAny(ext, `/\`) {
		ext = "mp3"
	}
	file := safeName(t.Title)
	if t.Track > 0 {
		file = fmt.Sprintf("%02d - %s", t.Track, file)
		if t.DiscNumber > 1 {
			file = fmt.Sprintf("%d-%s", t.DiscNumber, file)
		}
	}
	return filepath.Join(safeName(t.Artist), safeName(t.Album), file+"."+ext)
}

func musicTaskID(kind, id string) string { return "musicdl:" + kind + ":" + id }

// MusicDownload baixa um álbum ou uma playlist pro PC (tarefa
// "musicdl:<kind>:<id>", cancelável na Central).
func (s *Service) MusicDownload(kind, id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	c, err := s.musicAPI()
	if err != nil {
		return err
	}
	ctx, cancel := musicCtx()
	defer cancel()
	set := &localSetEntry{Kind: kind, ID: id}
	var songs []subsonic.Song
	switch kind {
	case "album":
		al, list, err := c.Album(ctx, id)
		if err != nil {
			return err
		}
		set.Name, set.Artist, set.ArtistID, set.Year, songs = al.Name, al.Artist, al.ArtistID, al.Year, list
	case "playlist":
		p, err := c.Playlist(ctx, id)
		if err != nil {
			return err
		}
		set.Name, songs = p.Name, p.Entry
	default:
		return errors.New("escolha um álbum ou uma playlist")
	}
	if len(songs) == 0 {
		return errors.New("nada pra baixar: está vazio")
	}
	dir := s.musicDir()
	label := "Baixando " + set.Name
	if !s.runTaskCtx(musicTaskID(kind, id), label, func(ctx context.Context, progress func(float64)) (string, error) {
		return s.musicFetch(ctx, c, dir, set, songs, progress)
	}) {
		return errors.New("já está baixando")
	}
	return nil
}

func (s *Service) musicFetch(ctx context.Context, c *subsonic.API, dir string, set *localSetEntry, songs []subsonic.Song, progress func(float64)) (string, error) {
	var total, done int64
	for _, t := range songs {
		total += max(t.Size, 1)
	}
	var last time.Time
	tick := func(n int64) {
		done += n
		if time.Since(last) > 200*time.Millisecond {
			last = time.Now()
			progress(min(0.99, float64(done)/float64(max(total, 1))))
		}
	}
	base := s.musicProxy()
	for _, t := range songs {
		rel := trackPath(t)
		if err := s.fetchTrack(ctx, c, filepath.Join(dir, rel), t, tick); err != nil {
			return "", fmt.Errorf("%s: %w", t.Title, err)
		}
		view := s.trackView(base, c.Base, t)
		view.Cover, view.Local = "", true
		s.music.idxMu.Lock()
		s.index().Tracks[localKey(c.Base, t.ID)] = localTrackEntry{Path: rel, Track: view}
		s.music.idxMu.Unlock()
		set.Tracks = append(set.Tracks, t.ID)
	}
	if set.Kind == "playlist" {
		rel, err := writeM3U(dir, set.Name, songs)
		if err != nil {
			return "", err
		}
		set.M3U = rel
	}
	set.At = time.Now().UnixNano()
	s.music.idxMu.Lock()
	s.index().Sets[localKey(c.Base, set.Kind, set.ID)] = set
	err := s.saveIndex()
	s.music.idxMu.Unlock()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s no PC (%d faixa%s)", set.Name, len(set.Tracks), plural(len(set.Tracks))), nil
}

// fetchTrack baixa uma faixa (pula a que já está inteira; continua o .part).
func (s *Service) fetchTrack(ctx context.Context, c *subsonic.API, dest string, t subsonic.Song, tick func(int64)) error {
	if info, err := os.Stat(dest); err == nil && (t.Size <= 0 || info.Size() == t.Size) {
		tick(max(t.Size, 1))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	part := dest + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	offset, _ := f.Seek(0, io.SeekEnd)
	tick(offset)
	_, err = c.Download(ctx, t.ID, offset, f, tick)
	if subsonic.ErrResume(err) { // servidor sem Range: do zero
		tick(-offset)
		f.Truncate(0)
		f.Seek(0, io.SeekStart)
		_, err = c.Download(ctx, t.ID, 0, f, tick)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(part, dest)
}

// writeM3U escreve Playlists/<nome>.m3u8 com caminhos relativos.
func writeM3U(dir, name string, songs []subsonic.Song) (string, error) {
	rel := filepath.Join("Playlists", safeName(name)+".m3u8")
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, t := range songs {
		fmt.Fprintf(&b, "#EXTINF:%d,%s - %s\n", t.Duration, t.Artist, t.Title)
		b.WriteString(filepath.ToSlash(filepath.Join("..", trackPath(t))) + "\n")
	}
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	return rel, os.WriteFile(p, []byte(b.String()), 0o644)
}

// MusicRemoveLocal apaga a cópia do PC de um álbum ou playlist (faixa que
// outro álbum/playlist baixado também usa fica).
func (s *Service) MusicRemoveLocal(kind, id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	l := s.musicLinkOf(s.cur().id)
	if l == nil {
		return errNoMusic
	}
	dir := s.musicDir()
	s.music.idxMu.Lock()
	defer s.music.idxMu.Unlock()
	idx := s.index()
	key := localKey(l.Base, kind, id)
	set, ok := idx.Sets[key]
	if !ok {
		return errors.New("não tem cópia no PC")
	}
	delete(idx.Sets, key)
	used := map[string]bool{}
	for k, other := range idx.Sets {
		if strings.HasPrefix(k, l.Base+"|") {
			for _, t := range other.Tracks {
				used[t] = true
			}
		}
	}
	for _, t := range set.Tracks {
		if used[t] {
			continue
		}
		tk := localKey(l.Base, t)
		if e, ok := idx.Tracks[tk]; ok {
			p := filepath.Join(dir, e.Path)
			if inside(dir, p) {
				os.Remove(p)
				pruneDirs(dir, filepath.Dir(p))
			}
			delete(idx.Tracks, tk)
		}
	}
	if set.M3U != "" {
		if p := filepath.Join(dir, set.M3U); inside(dir, p) {
			os.Remove(p)
			pruneDirs(dir, filepath.Dir(p))
		}
	}
	return s.saveIndex()
}

func inside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != "." && filepath.IsLocal(rel)
}

// pruneDirs apaga as pastas que ficaram vazias até a raiz (sem ela).
func pruneDirs(root, dir string) {
	for inside(root, dir) {
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
