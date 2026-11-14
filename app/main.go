// arkflix: frontend de jogos (biblioteca, saves, controle, configurações).
//
// Uso:
//
//	arkflix                      # janela desktop (build com -tags desktop)
//	arkflix --web 127.0.0.1:8484 # mesma interface no navegador
//	arkflix --smoke              # abre escondido, espera a tela e sai (teste)
//
// Os jogos abrem nos Flatpaks do RetroArch, PPSSPP e PCSX2; --bin e --cores
// trocam por executáveis e cores próprios. --data: configurações e estado. --fixture monta um ambiente falso (testes de ponta a ponta).
package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/stanleygomes/arkflix/app/internal/bt"
	"github.com/stanleygomes/arkflix/app/internal/core"
	"github.com/stanleygomes/arkflix/app/internal/fixture"
	"github.com/stanleygomes/arkflix/app/internal/gamepad"
	"github.com/stanleygomes/arkflix/app/internal/httpapi"
	"github.com/stanleygomes/arkflix/app/internal/launcher"
	"github.com/stanleygomes/arkflix/app/internal/settings"
)

var version = "dev"

//go:embed all:web/dist
var dist embed.FS

type flags struct {
	windowed bool
	web      string
	smoke    bool
	data     string
	bin      string
	cores    string
	fixture  string
	test     bool
	assets   string
	input    string
}

func parseFlags() flags {
	var f flags
	flag.StringVar(&f.web, "web", "", "serve a interface no navegador neste endereço (ex.: 127.0.0.1:8484)")
	flag.BoolVar(&f.windowed, "windowed", false, "abre em janela em vez de tela cheia (F11 alterna)")
	flag.BoolVar(&f.smoke, "smoke", false, "abre a janela escondida, espera a interface ficar pronta e sai (0 = ok)")
	flag.StringVar(&f.data, "data", "", "pasta de configurações e estado (padrão: ~/.config/arkflix)")
	flag.StringVar(&f.bin, "bin", "", "pasta com retroarch, ppsspp e pcsx2 próprios (padrão: Flatpaks)")
	flag.StringVar(&f.cores, "cores", "", "pasta dos cores do RetroArch (padrão: a do Flatpak)")
	flag.StringVar(&f.fixture, "fixture", "", "monta um ambiente falso nesta pasta (testes)")
	flag.BoolVar(&f.test, "test", false, "libera rotas de teste (/api/test/*) no modo web")
	flag.StringVar(&f.assets, "assets", "", "serve a interface desta pasta em vez da embutida (desenvolvimento)")
	flag.StringVar(&f.input, "input", "/dev/input", "pasta dos dispositivos de entrada")
	showVersion := flag.Bool("version", false, "mostra a versão")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}
	return f
}

// defaultData: ~/.config/arkflix no Linux, %AppData%\arkflix no
// Windows, ~/Library/Application Support/arkflix no macOS.
func defaultData() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return legacyDir(dir, "dados")
}

// legacyDir: <base>/arkflix, trazendo a <base>/arkflix-games de antes do nome
// novo na primeira vez (o que falhar, avisa e segue com a pasta nova).
func legacyDir(base, what string) string {
	dir := filepath.Join(base, "arkflix")
	old := filepath.Join(base, "arkflix-games")
	if moved, err := settings.MoveLegacy(dir, old); err != nil {
		log.Printf("não deu pra trazer os %s de %s: %v", what, old, err)
	} else if moved {
		log.Printf("%s de %s agora em %s", what, old, dir)
	}
	return dir
}

func assets(f flags) (fs.FS, error) {
	if f.assets != "" {
		return os.DirFS(f.assets), nil
	}
	sub, err := fs.Sub(dist, "web/dist")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, errors.New("interface não compilada: rode make app-web")
	}
	return sub, nil
}

// openStore abre as configurações e diz onde estão os emuladores.
func openStore(f flags) (*settings.Store, launcher.Spec, error) {
	home, _ := os.UserHomeDir()
	defaults := settings.Defaults(home)
	emu := launcher.Spec{Bin: f.bin, Cores: f.cores}
	data := f.data
	if f.fixture != "" {
		env := fixture.NewEnv(f.fixture)
		if os.Getenv("ARKFLIX_FIXTURE_FIRST_RUN") == "1" {
			os.Remove(filepath.Join(env.Data, "profiles.json")) // app recém-instalado
		}
		// libretro falso (banco e imagens), pra testar capas sem internet
		thumbsSrv := httptest.NewServer(&fixture.FakeThumbs{})
		env.Settings.Covers.Images = thumbsSrv.URL + "/thumbs"
		env.Settings.Covers.DB = thumbsSrv.URL + "/metadat"
		defaults, data = env.Settings, env.Data
		emu = launcher.Spec{Bin: env.Bin, Cores: env.Cores}
	}
	if data == "" {
		data = defaultData()
	}
	store, err := settings.Open(data, defaults)
	return store, emu, err
}

// No ambiente falso toda sessão conta como partida (emulador falso fecha em 1 s).
func minPlay(f flags) time.Duration {
	if f.fixture != "" {
		return -1
	}
	return 0
}

func main() {
	log.SetFlags(0)
	f := parseFlags()
	ui, err := assets(f)
	if err != nil {
		log.Fatal(err)
	}
	store, emu, err := openStore(f)
	if err != nil {
		log.Fatalf("configurações: %v", err)
	}
	pads := gamepad.New(f.input)
	if f.fixture == "" {
		pads.Start()
	}
	defer pads.Close()

	if f.web != "" {
		broker := httpapi.NewBroker()
		svc, err := core.New(core.Options{Version: version, Mode: "web", Store: store, Pads: pads, Emitter: broker, MinPlay: minPlay(f), Cache: cacheDir(f), Emulators: emu,
			BrowserPads: f.fixture != "" && os.Getenv("ARKFLIX_FIXTURE_BROWSER_PADS") == "1", Bluetooth: bluetooth(f), Update: fixtureUpdate(f)})
		if err != nil {
			log.Fatal(err)
		}
		ln, err := net.Listen("tcp", f.web)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("arkflix %s em http://%s\n", version, ln.Addr())
		handler := httpapi.Handler(svc, broker, httpapi.Options{Assets: ui, Test: f.test, Pads: pads})
		if f.fixture != "" && f.test {
			// RomM falso no mesmo endereço (testes de ponta a ponta do servidor)
			mux := http.NewServeMux()
			mux.Handle("/fake-romm/", http.StripPrefix("/fake-romm", fixture.NewFakeRomM()))
			// Jellyfin falso: stanley / senha123, filmes e uma série
			mux.Handle("/fake-jellyfin/", http.StripPrefix("/fake-jellyfin", fixture.NewFakeJellyfin()))
			// RetroAchievements falso: stanley / senha123, Super Mario World
			ra := fixture.NewFakeRA()
			ra.Prefix = "/fake-ra"
			mux.Handle("/fake-ra/", http.StripPrefix("/fake-ra", ra))
			os.Setenv("ARKFLIX_RA_BASE", "http://"+ln.Addr().String()+"/fake-ra")
			// Wikidata/Wikipedia falsos: Sonic 2 e Crash Bandicoot
			mux.Handle("/fake-wiki/", http.StripPrefix("/fake-wiki", fixture.NewFakeWiki()))
			os.Setenv("ARKFLIX_WIKI_BASE", "http://"+ln.Addr().String()+"/fake-wiki")
			mux.Handle("/", handler)
			handler = mux
		}
		log.Fatal(http.Serve(ln, handler))
	}
	desktopCache = cacheDir(f)
	os.Exit(runDesktop(store, pads, ui, f.smoke, f.windowed, emu, bluetooth(f)))
}

// bluetooth: no ambiente falso, rádio de mentira (testes nunca pareiam nada
// de verdade); fora dele, o do sistema.
func bluetooth(f flags) bt.Backend {
	if f.fixture != "" {
		return bt.NewFake()
	}
	return bt.New()
}

// cacheDir: capas baixadas e bancos do libretro (~/.cache/arkflix no
// Linux; a pasta de cache do sistema nos outros).
// No ambiente falso, fica dentro dele.
func cacheDir(f flags) string {
	if f.fixture != "" {
		return filepath.Join(f.fixture, "cache")
	}
	// teste de fumaça com pasta própria: nem lê nem migra o cache de verdade
	if f.smoke && f.data != "" {
		return filepath.Join(f.data, "cache")
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache")
	}
	return legacyDir(dir, "arquivos de cache")
}

// fixtureUpdate: no navegador só o ambiente falso olha atualização, e só com
// ARKFLIX_FIXTURE_RELEASES (URL de um GitHub falso, ou "fake" pro embutido).
// Baixa e confere, nunca instala.
func fixtureUpdate(f flags) core.UpdateOptions {
	api := os.Getenv("ARKFLIX_FIXTURE_RELEASES")
	if f.fixture == "" || api == "" {
		return core.UpdateOptions{}
	}
	if api == "fake" {
		api = httptest.NewServer(&fixture.FakeReleases{}).URL
	}
	current := version
	if current == "dev" {
		current = "v1.0.0" // "dev" nunca atualiza
	}
	return core.UpdateOptions{Check: true, API: api, DryRun: true, Current: current}
}

// O desktop nunca roda com o ambiente falso: partida mínima padrão (10 s).
const minPlayDesktop time.Duration = 0

// desktopCache: o cache que o main escolheu (cacheDir) pra janela.
var desktopCache string

func cacheDirDesktop() string { return desktopCache }
