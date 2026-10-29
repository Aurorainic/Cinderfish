// Command cinderfish-admin bootstraps and maintains local accounts from
// the shell. It exists mainly for the first administrator: an instance in
// registration_mode = "admin" has no signup path, so the first account has
// to be created here.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	configPath := flag.String("config", "", "path to config.toml (optional)")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("cinderfish-admin: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("cinderfish-admin: connect: %v", err)
	}
	defer pool.Close()
	if _, err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("cinderfish-admin: migrate: %v", err)
	}

	svc := account.NewService(pool)

	switch args[0] {
	case "create":
		createAccount(ctx, svc, args[1:])
	case "promote":
		setAdmin(ctx, svc, args[1:], true)
	case "demote":
		setAdmin(ctx, svc, args[1:], false)
	default:
		fmt.Fprintf(os.Stderr, "cinderfish-admin: unknown command %q\n\n", args[0])
		usage()
		os.Exit(2)
	}
}

func createAccount(ctx context.Context, svc *account.Service, args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	username := fs.String("username", "", "login name")
	password := fs.String("password", "", "password (prefer --password-stdin)")
	passwordStdin := fs.Bool("password-stdin", false, "read the password from standard input")
	admin := fs.Bool("admin", false, "grant administrator rights")
	_ = fs.Parse(args)

	if *username == "" {
		log.Fatal("cinderfish-admin create: --username is required")
	}
	pass := *password
	if *passwordStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("cinderfish-admin: read stdin: %v", err)
		}
		pass = strings.TrimRight(string(data), "\r\n")
	}
	if pass == "" {
		log.Fatal("cinderfish-admin create: a password is required (--password or --password-stdin)")
	}

	user, err := svc.Create(ctx, *username, pass)
	if err != nil {
		log.Fatalf("cinderfish-admin: create: %v", err)
	}
	if *admin {
		if err := svc.SetAdmin(ctx, user.ID, true); err != nil {
			log.Fatalf("cinderfish-admin: grant admin: %v", err)
		}
	}
	fmt.Printf("created %s (%s), admin=%t\n", user.Username, user.ID, *admin)
}

func setAdmin(ctx context.Context, svc *account.Service, args []string, admin bool) {
	fs := flag.NewFlagSet("promote", flag.ExitOnError)
	username := fs.String("username", "", "login name")
	_ = fs.Parse(args)
	if *username == "" {
		log.Fatal("cinderfish-admin: --username is required")
	}

	user, err := svc.GetByUsername(ctx, *username)
	if err != nil {
		log.Fatalf("cinderfish-admin: %v", err)
	}
	if err := svc.SetAdmin(ctx, user.ID, admin); err != nil {
		log.Fatalf("cinderfish-admin: %v", err)
	}
	fmt.Printf("%s (%s), admin=%t\n", user.Username, user.ID, admin)
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, `cinderfish-admin — local account maintenance

Usage:
  cinderfish-admin [--config path] create --username NAME (--password PASS | --password-stdin) [--admin]
  cinderfish-admin [--config path] promote --username NAME
  cinderfish-admin [--config path] demote  --username NAME

The database URL comes from the configuration file or CINDERFISH_DATABASE_URL.
`)
}
