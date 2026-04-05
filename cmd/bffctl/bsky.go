package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"slices"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/strideynet/bsky-furry-feed/bluesky"
	"github.com/strideynet/bsky-furry-feed/feed"
	"github.com/urfave/cli/v2"
	"go.yaml.in/yaml/v2"
)

func bskyCmd(log *slog.Logger, env *environment) *cli.Command {
	return &cli.Command{
		Name: "bsky",
		Subcommands: []*cli.Command{
			{
				Name:  "list-feeds",
				Usage: "Lists all feeds",
				Action: func(ctx *cli.Context) error {
					feeds := feed.ServiceWithDefaultFeeds(nil)
					metas := feeds.Metas()
					slices.SortFunc(metas, func(a, b feed.Meta) int { return strings.Compare(a.ID, b.ID) })
					for _, meta := range metas {
						o, err := yaml.Marshal(meta)
						if err != nil {
							return err
						}
						fmt.Println(string(o))
					}
					return nil
				},
			},
			{
				Name:  "purge-feeds",
				Usage: "Deletes all feeds associated with the current account",
				Action: func(cctx *cli.Context) error {
					client, err := getBlueskyClient(cctx.Context, env)
					if err != nil {
						return err
					}
					return client.PurgeFeeds(cctx.Context)
				},
			},
			{
				Name:  "publish-feeds",
				Usage: "Publishes feeds from hardcoded list.",
				Action: func(cctx *cli.Context) error {
					hostname := os.Getenv("BFF_HOSTNAME")
					if hostname == "" {
						return fmt.Errorf("BFF_HOSTNAME not set")
					}
					log.Info(hostname)

					client, err := getBlueskyClient(cctx.Context, env)
					if err != nil {
						return err
					}
					f, err := os.OpenFile("./furrylist.png", os.O_RDONLY, 0)
					if err != nil {
						return fmt.Errorf("reading avatar: %w", err)
					}
					blob, err := client.UploadBlob(cctx.Context, f)
					if err != nil {
						return fmt.Errorf("uploading avatar: %w", err)
					}

					feeds := feed.ServiceWithDefaultFeeds(nil)
					for _, meta := range feeds.Metas() {
						meta := meta

						log.Info("upserting feed", slog.String("rkey", meta.ID))
						gen := &bsky.FeedGenerator{
							Avatar:      blob,
							Did:         fmt.Sprintf("did:web:%s", hostname),
							CreatedAt:   bluesky.FormatTime(time.Now().UTC()),
							Description: &meta.Description,
							DisplayName: meta.DisplayName,
						}

						if meta.VideoOnly {
							var contentModeVideo = "app.bsky.feed.defs#contentModeVideo"
							gen.ContentMode = &contentModeVideo
						}

						err = client.PutRecord(cctx.Context, "app.bsky.feed.generator", meta.ID, gen)
						if err != nil {
							return fmt.Errorf("putting feed record: %w", err)
						}
						log.Info("upserted feed", slog.String("rkey", meta.ID))
					}

					log.Info("blob", slog.String("ref", blob.Ref.String()))
					return nil
				},
			},
			{
				Name: "resolve-handle",
				Action: func(cctx *cli.Context) error {
					client, err := getBlueskyClient(cctx.Context, env)
					if err != nil {
						return err
					}
					did, err := client.ResolveHandle(cctx.Context, cctx.Args().First())
					if err != nil {
						return err
					}
					log.Info("found did", slog.String("did", did.Did))
					return nil
				},
			},
		},
	}
}
