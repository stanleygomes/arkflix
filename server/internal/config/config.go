// Package config junta as opções do servidor: onde ficam os dados dele e
// as pastas de mídia que ele só lê (as mesmas do Jellyfin, Navidrome e RomM).
package config

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
)

// Config é lida das flags, com o ambiente (ARKFLIX_SERVER_*) como padrão.
type Config struct {
	Addr string // endereço de escuta (":8097")
	Data string // banco, chave e saves; o único lugar onde o servidor escreve

	Movies string // filmes, um por arquivo ou pasta (/var/media/Movies)
	Shows  string // séries: Série/Temporada 01/S01E02.mkv (/var/media/Shows)
	Music  string // música com tags (/var/media/musicas)
	Games  string // raiz do RomM: roms/<console>/ e bios/<console>/ (/var/media/jogos)

	FFmpeg  string // binário do ffmpeg ("" = sem transcodificação)
	FFprobe string // binário do ffprobe ("" = sem faixas de áudio/legenda)

	Name string // nome que os clientes mostram

	// Writable libera escrever nas pastas de mídia (envio de arquivo e
	// esconder item pelo painel de admin). Desligado = só leitura, o modo
	// de quando o Jellyfin/Navidrome/RomM ainda estão no ar.
	Writable bool
	// Hidden é pra onde "esconder" move um arquivo (mesmo caminho relativo,
	// fora das bibliotecas). Padrão: <pai da pasta de filmes>/escondidos.
	Hidden string

	// Metadados da internet (internal/metadata). Sem chave do TMDB, filmes e
	// séries ficam só com o que tem no disco; capas de jogos (libretro) e de
	// música (MusicBrainz/Cover Art Archive) não pedem chave.
	TMDBKey  string
	Metadata bool   // false desliga toda busca na internet
	Language string // idioma dos metadados (pt-BR)
}

func env(k, def string) string {
	if v, ok := os.LookupEnv("ARKFLIX_SERVER_" + k); ok {
		return v
	}
	return def
}

func lookPath(bin string) string {
	p, err := execLookPath(bin)
	if err != nil {
		return ""
	}
	return p
}

// Flags registra as opções no FlagSet e devolve a Config que elas preenchem.
func Flags(fs *flag.FlagSet) *Config {
	c := &Config{}
	fs.StringVar(&c.Addr, "addr", env("ADDR", ":8097"), "endereço de escuta")
	fs.StringVar(&c.Data, "data", env("DATA", "/var/lib/arkflix-server"), "pasta de dados do servidor")
	fs.StringVar(&c.Movies, "movies", env("MOVIES", ""), "pasta dos filmes")
	fs.StringVar(&c.Shows, "shows", env("SHOWS", ""), "pasta das séries")
	fs.StringVar(&c.Music, "music", env("MUSIC", ""), "pasta das músicas")
	fs.StringVar(&c.Games, "games", env("GAMES", ""), "pasta dos jogos (com roms/ e bios/)")
	fs.StringVar(&c.FFmpeg, "ffmpeg", env("FFMPEG", lookPath("ffmpeg")), "binário do ffmpeg")
	fs.StringVar(&c.FFprobe, "ffprobe", env("FFPROBE", lookPath("ffprobe")), "binário do ffprobe")
	fs.StringVar(&c.Name, "name", env("NAME", "arkflix"), "nome do servidor")
	fs.BoolVar(&c.Writable, "writable", env("WRITABLE", "") == "1" || env("WRITABLE", "") == "true", "deixa o painel de admin escrever nas pastas de mídia")
	fs.StringVar(&c.Hidden, "hidden", env("HIDDEN", ""), "pasta pra onde \"esconder\" move arquivos")
	fs.StringVar(&c.TMDBKey, "tmdb-key", env("TMDB_KEY", ""), "chave da API do TMDB (filmes e séries)")
	fs.BoolVar(&c.Metadata, "metadata", env("METADATA", "1") != "0", "busca capas e descrições na internet")
	fs.StringVar(&c.Language, "language", env("LANGUAGE", "pt-BR"), "idioma dos metadados")
	return c
}

// Check valida e deixa os caminhos absolutos.
func (c *Config) Check() error {
	if c.Data == "" {
		return errors.New("falta --data")
	}
	if c.Hidden == "" && c.Movies != "" {
		c.Hidden = filepath.Join(filepath.Dir(c.Movies), "escondidos")
	}
	for _, p := range []*string{&c.Data, &c.Movies, &c.Shows, &c.Music, &c.Games, &c.Hidden} {
		if *p == "" {
			continue
		}
		abs, err := filepath.Abs(*p)
		if err != nil {
			return err
		}
		*p = abs
	}
	return os.MkdirAll(c.Data, 0o700)
}
