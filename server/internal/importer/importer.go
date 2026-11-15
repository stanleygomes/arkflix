// Package importer traz o que já existe nos servidores antigos (Jellyfin,
// Navidrome, RomM) pro arkflix-server: usuários, onde parou, favoritos,
// fila, marcadores, saves e listas. Roda uma vez, pela linha de comando:
//
//	arkflix-server import jellyfin  --db jellyfin.db [--map OLD=NEW] [--dry-run]
//	arkflix-server import navidrome --db navidrome.db [--key K] [--map ...] [--dry-run]
//	arkflix-server import romm --dsn 'u:p@tcp(host:3306)/romm' --assets DIR [--map ...] [--dry-run]
//
// Os bancos de origem só são lidos (SQLite com mode=ro; MariaDB numa
// transação READ ONLY). Os itens casam pelo caminho do arquivo, depois de
// trocar os prefixos de --map. Rodar de novo não duplica nada.
package importer

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/deps"
	"github.com/stanleygomes/arkflix/server/internal/lists"

	// as tabelas do RomM (saves, props) e do Subsonic (fila, marcadores)
	_ "github.com/stanleygomes/arkflix/server/internal/api/romm"
	_ "github.com/stanleygomes/arkflix/server/internal/api/subsonic"
)

// Importer junta o que as três fontes usam.
type Importer struct {
	D         *deps.Deps
	Lists     ListSink  // playlists do Navidrome e coleções do RomM
	Out       io.Writer // relatório
	Maps      Mapper    // troca de prefixo dos caminhos
	Dry       bool      // só mostra o que mudaria
	Overwrite bool      // senha da fonte troca a de quem já existe
	Show      int       // quantos caminhos sem par mostrar

	idx *index
	rep *Report
	ud  map[udKey]udIn
}

const usage = `uso: arkflix-server import jellyfin|navidrome|romm [flags]

  jellyfin  --db jellyfin.db
  navidrome --db navidrome.db [--key PasswordEncryptionKey]
  romm      --dsn 'usuario:senha@tcp(host:3306)/romm' --assets PASTA [--library /romm/library]

comuns: --map ANTIGO=NOVO (repete), --dry-run, --passwords keep|overwrite, --show N`

// Run executa `arkflix-server import <fonte> [flags]`.
func Run(ctx context.Context, d *deps.Deps, args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New(usage)
	}
	src := args[0]
	fs := flag.NewFlagSet("import "+src, flag.ContinueOnError)
	im := &Importer{D: d, Lists: storeLists{lists.New(d.Store)}, Out: os.Stdout}
	fs.Var(&im.Maps, "map", "ANTIGO=NOVO: troca o prefixo dos caminhos da fonte (repete)")
	fs.BoolVar(&im.Dry, "dry-run", false, "só mostra o que mudaria")
	pw := fs.String("passwords", "keep", "keep: não troca a senha de quem já existe; overwrite: troca")
	fs.IntVar(&im.Show, "show", 10, "quantos caminhos sem par mostrar")
	var db, key, dsn, assets, library string
	switch src {
	case "jellyfin":
		fs.StringVar(&db, "db", "", "jellyfin.db (config/data/jellyfin.db)")
	case "navidrome":
		fs.StringVar(&db, "db", "", "navidrome.db")
		fs.StringVar(&key, "key", "", "PasswordEncryptionKey do Navidrome (vazio = a padrão)")
	case "romm":
		fs.StringVar(&dsn, "dsn", "", "MariaDB do RomM: usuario:senha@tcp(host:3306)/romm")
		fs.StringVar(&assets, "assets", "", "pasta de assets do RomM (/romm/assets): saves, states, screenshots")
		fs.StringVar(&library, "library", "/romm/library", "pasta da biblioteca como o RomM vê")
	default:
		return fmt.Errorf("fonte desconhecida: %s\n%s", src, usage)
	}
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	switch *pw {
	case "keep":
	case "overwrite":
		im.Overwrite = true
	default:
		return errors.New("--passwords: keep ou overwrite")
	}
	switch src {
	case "jellyfin":
		if db == "" {
			return errors.New("falta --db")
		}
		return im.Jellyfin(ctx, db)
	case "navidrome":
		if db == "" {
			return errors.New("falta --db")
		}
		return im.Navidrome(ctx, db, key)
	default:
		if dsn == "" {
			return errors.New("falta --dsn")
		}
		srcDB, done, err := openMySQL(ctx, dsn)
		if err != nil {
			return err
		}
		defer done()
		return im.RomM(ctx, srcDB, RomMOptions{Assets: assets, Library: library})
	}
}

// begin prepara um import: lê o acervo e zera o relatório.
func (im *Importer) begin(ctx context.Context, source string) error {
	if im.Out == nil {
		im.Out = io.Discard
	}
	if im.Lists == nil {
		im.Lists = storeLists{lists.New(im.D.Store)}
	}
	idx, err := loadIndex(ctx, im.D.Store)
	if err != nil {
		return err
	}
	if idx.size == 0 {
		return errors.New("acervo vazio: rode `arkflix-server scan` antes de importar (os itens casam pelo caminho)")
	}
	im.idx = idx
	im.rep = newReport(source, im.Dry)
	im.ud = map[udKey]udIn{}
	return nil
}

// finish grava o estado acumulado e imprime o relatório.
func (im *Importer) finish(ctx context.Context) error {
	if err := im.flushUserData(ctx); err != nil {
		return err
	}
	im.rep.Print(im.Out, im.Show)
	return nil
}

// Report devolve o relatório do último import (testes).
func (im *Importer) Report() *Report { return im.rep }

// userRef é o usuário do arkflix que corresponde a um da fonte.
type userRef struct {
	ID   string
	RID  int64
	Name string
}

// user cria ou junta o usuário da fonte (por nome, sem diferenciar
// maiúsculas). No dry-run não grava: quem ainda não existe ganha um id de
// mentira, só pra contar o resto.
func (im *Importer) user(ctx context.Context, in auth.Imported) (userRef, error) {
	in.Overwrite = im.Overwrite
	r := im.rep
	if im.Dry {
		u, err := im.D.Auth.UserByName(ctx, strings.TrimSpace(in.Name))
		if err != nil {
			r.Add("usuários criados", 1)
			if in.Password == "" && in.Hash == "" {
				r.Note("%s: sem senha na fonte, vai precisar de `user passwd`", in.Name)
			}
			return userRef{ID: "dry:" + strings.ToLower(in.Name), Name: in.Name}, nil
		}
		r.Add("usuários juntados", 1)
		if has, err := im.D.Auth.HasPassword(ctx, u.ID); err == nil && has && !im.Overwrite && (in.Password != "" || in.Hash != "") {
			r.Note("%s: já tem senha, fica a dela (--passwords=overwrite troca)", u.Name)
		}
		return userRef{ID: u.ID, RID: u.RID, Name: u.Name}, nil
	}
	res, err := im.D.Auth.ImportUser(ctx, in)
	if errors.Is(err, auth.ErrUnknownHash) {
		r.Note("%s: hash de senha em formato desconhecido, importado sem senha", in.Name)
		in.Hash = ""
		res, err = im.D.Auth.ImportUser(ctx, in)
	}
	if err != nil {
		return userRef{}, fmt.Errorf("usuário %s: %w", in.Name, err)
	}
	if res.Created {
		r.Add("usuários criados", 1)
	} else {
		r.Add("usuários juntados", 1)
	}
	switch {
	case res.NoPassword:
		r.Note("%s: sem senha, defina com `arkflix-server user passwd %s`", res.User.Name, res.User.Name)
	case res.PassKept && (in.Password != "" || in.Hash != ""):
		r.Note("%s: já tinha senha, ficou a dela (--passwords=overwrite troca)", res.User.Name)
	}
	return userRef{ID: res.User.ID, RID: res.User.RID, Name: res.User.Name}, nil
}
