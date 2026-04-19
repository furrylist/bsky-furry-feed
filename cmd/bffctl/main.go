package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/exp/slices"

	"github.com/joho/godotenv"
	"github.com/strideynet/bsky-furry-feed/bfflog"
	"github.com/strideynet/bsky-furry-feed/bluesky"
	"github.com/strideynet/bsky-furry-feed/internal/env"
	"github.com/urfave/cli/v2"
)

type environment struct {
	dbURL                   string
	blacklistBlueskyHandles []string
}

var environments = map[env.Mode]environment{
	env.ModeDev: {
		dbURL: "postgres://bff:bff@localhost:5432/bff?sslmode=disable",
		blacklistBlueskyHandles: []string{
			"furryli.st",
		},
	},
}

// TODO: Have a `login` and `logout` command that persists auth state to disk.
func getBlueskyClient(ctx context.Context, e *environment) (*bluesky.PDSClient, error) {
	creds, err := bluesky.CredentialsFromEnv()
	if err != nil {
		return nil, err
	}
	if slices.Contains(e.blacklistBlueskyHandles, creds.Identifier) {
		return nil, fmt.Errorf("prohibited handle for environment used")
	}
	return bluesky.ClientFromCredentials(ctx, bluesky.DefaultPDSHost, creds)
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(log)

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Info("could not load .env file", bfflog.Err(err))
	}

	environments[env.ModeProd] = environment{
		dbURL: os.Getenv(env.EnvDB_URI),
	}

	var environ = &environment{}
	app := &cli.App{
		Name:  "bffctl",
		Usage: "The swiss army knife of any BFF operator",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "environment",
				Aliases: []string{
					"e",
				},
				Action: func(c *cli.Context, s string) error {
					if s == "" {
						s = string(env.ModeDev)
					}
					v, ok := environments[env.Mode(s)]
					if !ok {
						return fmt.Errorf("unrecognized environment: %s", s)
					}
					log.Info("configured environment", slog.String("env", s))
					*environ = v
					return nil
				},
			},
		},
		Commands: []*cli.Command{
			dbCmd(log, environ),
			bskyCmd(log, environ),
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
