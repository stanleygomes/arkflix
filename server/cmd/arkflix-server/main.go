// arkflix-server: um servidor só no lugar do Jellyfin, do Navidrome e do
// RomM, falando a API dos três pros clientes que já existem.
//
//	arkflix-server [flags]                    sobe o servidor (escaneia ao subir)
//	arkflix-server [flags] scan               escaneia e sai
//	arkflix-server [flags] user add NOME [--admin]   senha pela entrada padrão
//	arkflix-server [flags] user passwd NOME
//	arkflix-server [flags] user del NOME
//	arkflix-server [flags] user list
//	arkflix-server [flags] import jellyfin|navidrome|romm [flags]   traz dados dos servidores antigos
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/config"
	"github.com/stanleygomes/arkflix/server/internal/deps"
	"github.com/stanleygomes/arkflix/server/internal/importer"
	"github.com/stanleygomes/arkflix/server/internal/library"
	"github.com/stanleygomes/arkflix/server/internal/media"
	"github.com/stanleygomes/arkflix/server/internal/metadata"
	"github.com/stanleygomes/arkflix/server/internal/store"
	"github.com/stanleygomes/arkflix/server/internal/web"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("arkflix-server", flag.ContinueOnError)
	cfg := config.Flags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := cfg.Check(); err != nil {
		return err
	}
	st, err := store.Open(cfg.Data)
	if err != nil {
		return err
	}
	defer st.Close()
	au, err := auth.New(st, cfg.Data)
	if err != nil {
		return err
	}
	d := &deps.Deps{Config: cfg, Store: st, Auth: au, Library: library.New(cfg, st), Media: media.New(cfg), Meta: metadata.New(cfg, st), Version: version}
	// depois de cada scan, completa pela internet o que faltar
	d.Meta.Version = version
	d.Library.OnScanned = func() { go d.Meta.Run(context.Background()) }

	rest := fs.Args()
	if len(rest) == 0 {
		return serve(d)
	}
	ctx := context.Background()
	switch rest[0] {
	case "scan":
		if err := d.Library.Scan(ctx); err != nil {
			return err
		}
		fmt.Printf("%d itens\n", d.Library.Status().Count)
		return nil
	case "user":
		return user(ctx, au, rest[1:])
	case "import":
		return importer.Run(ctx, d, rest[1:])
	}
	return fmt.Errorf("comando desconhecido: %s", rest[0])
}

func serve(d *deps.Deps) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := firstUser(ctx, d.Auth); err != nil {
		return err
	}
	srv := &http.Server{Addr: d.Config.Addr, Handler: web.New(d), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(sh)
	}()
	go func() {
		if err := d.Library.Scan(ctx); err != nil {
			slog.Error("scan", "err", err)
		}
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				d.Auth.PurgeExpired(ctx)
				d.Library.Scan(ctx)
			}
		}
	}()
	slog.Info("arkflix-server no ar", "addr", d.Config.Addr, "version", d.Version)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// firstUser cria o admin na primeira subida a partir de
// ARKFLIX_SERVER_ADMIN_USER/ARKFLIX_SERVER_ADMIN_PASSWORD.
func firstUser(ctx context.Context, au *auth.Auth) error {
	users, err := au.Users(ctx)
	if err != nil || len(users) > 0 {
		return err
	}
	name, pass := os.Getenv("ARKFLIX_SERVER_ADMIN_USER"), os.Getenv("ARKFLIX_SERVER_ADMIN_PASSWORD")
	if name == "" || pass == "" {
		slog.Warn("nenhum usuário: rode `arkflix-server user add NOME --admin`")
		return nil
	}
	_, err = au.CreateUser(ctx, name, pass, true)
	return err
}

func readPassword() (string, error) {
	fmt.Fprint(os.Stderr, "senha: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func user(ctx context.Context, au *auth.Auth, args []string) error {
	if len(args) == 0 {
		return errors.New("user add|passwd|del|list")
	}
	need := func() (string, error) {
		if len(args) < 2 {
			return "", errors.New("falta o nome")
		}
		return args[1], nil
	}
	switch args[0] {
	case "list":
		users, err := au.Users(ctx)
		for _, u := range users {
			admin := ""
			if u.Admin {
				admin = " (admin)"
			}
			fmt.Println(u.Name + admin)
		}
		return err
	case "add":
		name, err := need()
		if err != nil {
			return err
		}
		pass, err := readPassword()
		if err != nil {
			return err
		}
		admin := len(args) > 2 && args[2] == "--admin"
		_, err = au.CreateUser(ctx, name, pass, admin)
		return err
	case "passwd":
		name, err := need()
		if err != nil {
			return err
		}
		u, err := au.UserByName(ctx, name)
		if err != nil {
			return err
		}
		pass, err := readPassword()
		if err != nil {
			return err
		}
		return au.SetPassword(ctx, u.ID, pass)
	case "del":
		name, err := need()
		if err != nil {
			return err
		}
		u, err := au.UserByName(ctx, name)
		if err != nil {
			return err
		}
		return au.DeleteUser(ctx, u.ID)
	}
	return fmt.Errorf("user %s: desconhecido", args[0])
}
