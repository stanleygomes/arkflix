package store

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Tipos de item. Um acervo só, visto de jeitos diferentes por cada API.
const (
	KindMovie    = "movie"    // library movies
	KindSeries   = "series"   // library shows; path = pasta da série
	KindSeason   = "season"   // parent = série; idx = número
	KindEpisode  = "episode"  // parent = temporada; idx = episódio, parent_idx = temporada
	KindArtist   = "artist"   // library music; path = "artist:<nome normalizado>"
	KindAlbum    = "album"    // parent = artista do álbum; path = "album:<artista>/<álbum>"
	KindTrack    = "track"    // parent = álbum; idx = faixa, parent_idx = disco
	KindPlatform = "platform" // library games; path = roms/<slug>; platform = slug
	KindRom      = "rom"      // parent = console; arquivo ou pasta (multidisco)
	KindFirmware = "firmware" // parent = console; bios/<slug>/<arquivo>
)

// ErrNotFound vem quando o item/usuário não existe.
var ErrNotFound = errors.New("não encontrado")

// Stream é uma faixa do arquivo (do ffprobe).
type Stream struct {
	Index    int     `json:"index"`
	Type     string  `json:"type"` // video | audio | subtitle
	Codec    string  `json:"codec"`
	Language string  `json:"language,omitempty"`
	Title    string  `json:"title,omitempty"`
	Default  bool    `json:"default,omitempty"`
	Forced   bool    `json:"forced,omitempty"`
	Channels int     `json:"channels,omitempty"`
	Layout   string  `json:"layout,omitempty"`
	Rate     int     `json:"rate,omitempty"` // sample rate
	Bitrate  int     `json:"bitrate,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	FPS      float64 `json:"fps,omitempty"`
	Profile  string  `json:"profile,omitempty"`
	Level    int     `json:"level,omitempty"`
	PixFmt   string  `json:"pixFmt,omitempty"`
	Range    string  `json:"range,omitempty"`    // SDR | HDR10 | HLG | DOVI
	External string  `json:"external,omitempty"` // legenda de arquivo ao lado (.srt/.vtt/.ass): caminho absoluto
}

// RomFile é um arquivo dentro da pasta de uma ROM multidisco (ou o próprio
// arquivo da ROM simples).
type RomFile struct {
	Name string `json:"name"` // nome do arquivo
	Rel  string `json:"rel"`  // relativo à pasta do console (roms/<slug>/)
	Size int64  `json:"size"`
	CRC  string `json:"crc,omitempty"`
	MD5  string `json:"md5,omitempty"`
	SHA1 string `json:"sha1,omitempty"`
}

// Item é uma linha de items. Genres/Streams/Extra já decodificados.
type Item struct {
	RID         int64
	ID          string
	Kind        string
	Library     string
	ParentID    string
	SeriesID    string
	SeasonID    string
	ArtistID    string
	AlbumID     string
	Path        string
	RelPath     string
	Name        string
	SortName    string
	Overview    string
	Year        int
	Premiere    string
	Genres      []string
	Index       int
	ParentIndex int
	Artist      string
	AlbumArtist string
	Album       string
	Platform    string
	Duration    float64
	Size        int64
	MTime       int64
	Container   string
	Bitrate     int
	Width       int
	Height      int
	Streams     []Stream
	Image       string
	Backdrop    string
	Logo        string
	Extra       map[string]any
	AddedAt     time.Time
	UpdatedAt   time.Time
}

// Files lê Extra["files"] de uma ROM.
func (it *Item) Files() []RomFile {
	raw, err := json.Marshal(it.Extra["files"])
	if err != nil {
		return nil
	}
	var out []RomFile
	json.Unmarshal(raw, &out)
	return out
}

// ID gera o id estável (32 hex) de um item a partir do tipo e da chave.
func ID(kind, key string) string {
	sum := md5.Sum([]byte(kind + "\x00" + key))
	return hex.EncodeToString(sum[:])
}

// NormalizeID aceita GUID com ou sem hífens, maiúsculo ou não.
func NormalizeID(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "-", ""))
}

// ItemColumns é a lista de colunas na ordem que ScanItem espera, pra
// SELECTs próprios: "SELECT " + ItemColumns + " FROM items i ...".
const ItemColumns = `i.rid, i.id, i.kind, i.library, COALESCE(i.parent_id,''), COALESCE(i.series_id,''), COALESCE(i.season_id,''),
COALESCE(i.artist_id,''), COALESCE(i.album_id,''), i.path, i.rel_path, i.name, i.sort_name, i.overview, i.year, i.premiere,
i.genres, i.idx, i.parent_idx, i.artist, i.album_artist, i.album, i.platform, i.duration, i.size, i.mtime, i.container,
i.bitrate, i.width, i.height, i.streams, i.image, i.backdrop, i.logo, i.extra, i.added_at, i.updated_at`

type scanner interface{ Scan(...any) error }

// ScanItem lê uma linha selecionada com ItemColumns.
func ScanItem(r scanner) (*Item, error) {
	var it Item
	var genres, streams, extra, added, updated string
	err := r.Scan(&it.RID, &it.ID, &it.Kind, &it.Library, &it.ParentID, &it.SeriesID, &it.SeasonID,
		&it.ArtistID, &it.AlbumID, &it.Path, &it.RelPath, &it.Name, &it.SortName, &it.Overview, &it.Year, &it.Premiere,
		&genres, &it.Index, &it.ParentIndex, &it.Artist, &it.AlbumArtist, &it.Album, &it.Platform, &it.Duration, &it.Size, &it.MTime, &it.Container,
		&it.Bitrate, &it.Width, &it.Height, &streams, &it.Image, &it.Backdrop, &it.Logo, &extra, &added, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(genres), &it.Genres)
	json.Unmarshal([]byte(streams), &it.Streams)
	json.Unmarshal([]byte(extra), &it.Extra)
	if it.Genres == nil {
		it.Genres = []string{}
	}
	if it.Extra == nil {
		it.Extra = map[string]any{}
	}
	it.AddedAt = parseTime(added)
	it.UpdatedAt = parseTime(updated)
	return &it, nil
}

func parseTime(s string) time.Time {
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// Now é o formato de data gravado no banco.
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// Item busca pelo id (aceita GUID com hífen).
func (s *Store) Item(ctx context.Context, id string) (*Item, error) {
	return ScanItem(s.DB.QueryRowContext(ctx, `SELECT `+ItemColumns+` FROM items i WHERE i.id = ?`, NormalizeID(id)))
}

// ItemByRID busca pelo id numérico.
func (s *Store) ItemByRID(ctx context.Context, rid int64) (*Item, error) {
	return ScanItem(s.DB.QueryRowContext(ctx, `SELECT `+ItemColumns+` FROM items i WHERE i.rid = ?`, rid))
}

// Items roda um SELECT com ItemColumns e devolve a lista.
func (s *Store) Items(ctx context.Context, query string, args ...any) ([]*Item, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Item
	for rows.Next() {
		it, err := ScanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// UpsertItem grava o item pelo caminho (Path é a chave). Mantém rid e
// added_at de quem já existia; ID vazio vira ID(Kind, Path).
func (s *Store) UpsertItem(ctx context.Context, it *Item) error {
	if it.ID == "" {
		it.ID = ID(it.Kind, it.Path)
	}
	if it.SortName == "" {
		it.SortName = SortName(it.Name)
	}
	genres, _ := json.Marshal(nonNil(it.Genres))
	streams, _ := json.Marshal(nonNilS(it.Streams))
	extra, _ := json.Marshal(nonNilM(it.Extra))
	now := Now()
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO items (id, kind, library, parent_id, series_id, season_id, artist_id, album_id, path, rel_path, name, sort_name,
  overview, year, premiere, genres, idx, parent_idx, artist, album_artist, album, platform, duration, size, mtime, container,
  bitrate, width, height, streams, image, backdrop, logo, extra, added_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(path) DO UPDATE SET id=excluded.id, kind=excluded.kind, library=excluded.library, parent_id=excluded.parent_id,
  series_id=excluded.series_id, season_id=excluded.season_id, artist_id=excluded.artist_id, album_id=excluded.album_id,
  rel_path=excluded.rel_path, name=excluded.name, sort_name=excluded.sort_name, overview=excluded.overview, year=excluded.year,
  premiere=excluded.premiere, genres=excluded.genres, idx=excluded.idx, parent_idx=excluded.parent_idx, artist=excluded.artist,
  album_artist=excluded.album_artist, album=excluded.album, platform=excluded.platform, duration=excluded.duration,
  size=excluded.size, mtime=excluded.mtime, container=excluded.container, bitrate=excluded.bitrate, width=excluded.width,
  height=excluded.height, streams=excluded.streams, image=excluded.image, backdrop=excluded.backdrop, logo=excluded.logo,
  extra=excluded.extra, updated_at=excluded.updated_at`,
		it.ID, it.Kind, it.Library, null(it.ParentID), null(it.SeriesID), null(it.SeasonID), null(it.ArtistID), null(it.AlbumID),
		it.Path, it.RelPath, it.Name, it.SortName, it.Overview, it.Year, it.Premiere, string(genres), it.Index, it.ParentIndex,
		it.Artist, it.AlbumArtist, it.Album, it.Platform, it.Duration, it.Size, it.MTime, it.Container, it.Bitrate, it.Width,
		it.Height, string(streams), it.Image, it.Backdrop, it.Logo, string(extra), addedAt(it), now)
	if err != nil {
		return err
	}
	return s.DB.QueryRowContext(ctx, `SELECT rid, added_at FROM items WHERE path = ?`, it.Path).Scan(&it.RID, new(string))
}

func addedAt(it *Item) string {
	if !it.AddedAt.IsZero() {
		return it.AddedAt.UTC().Format(time.RFC3339Nano)
	}
	return Now()
}

// DeleteItems apaga pelos ids.
func (s *Store) DeleteItems(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if _, err := s.DB.ExecContext(ctx, `DELETE FROM items WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

// SortName tira artigos do começo pra ordenar ("The Matrix" → "matrix").
func SortName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, a := range []string{"the ", "a ", "an ", "o ", "os ", "as "} {
		if strings.HasPrefix(n, a) && len(n) > len(a) {
			return n[len(a):]
		}
	}
	return n
}

func null(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilS(v []Stream) []Stream {
	if v == nil {
		return []Stream{}
	}
	return v
}

func nonNilM(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}
