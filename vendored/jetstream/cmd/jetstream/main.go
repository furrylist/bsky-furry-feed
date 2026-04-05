package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/parallel"
	"github.com/bluesky-social/jetstream/pkg/consumer"
	"github.com/bluesky-social/jetstream/pkg/server"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:    "jetstream",
		Usage:   "atproto firehose translation service",
		Version: "0.1.0",
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "ws-url",
			Usage:   "full websocket path to the ATProto SubscribeRepos XRPC endpoint",
			Value:   "wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos",
			EnvVars: []string{"JETSTREAM_WS_URL"},
		},
		&cli.IntFlag{
			Name:    "worker-count",
			Usage:   "number of workers to process events",
			Value:   100,
			EnvVars: []string{"JETSTREAM_WORKER_COUNT"},
		},
		&cli.IntFlag{
			Name:    "max-queue-size",
			Usage:   "max number of events to queue",
			Value:   1000,
			EnvVars: []string{"JETSTREAM_MAX_QUEUE_SIZE"},
		},
		&cli.StringFlag{
			Name:    "listen-addr",
			Usage:   "addr to serve echo on",
			Value:   ":6008",
			EnvVars: []string{"JETSTREAM_LISTEN_ADDR"},
		},
		&cli.StringFlag{
			Name:    "metrics-listen-addr",
			Usage:   "addr to serve prometheus metrics on",
			Value:   ":6009",
			EnvVars: []string{"JETSTREAM_METRICS_LISTEN_ADDR"},
		},
		&cli.StringFlag{
			Name:    "data-dir",
			Usage:   "directory to store data (pebbleDB)",
			Value:   "./data",
			EnvVars: []string{"JETSTREAM_DATA_DIR"},
		},
		&cli.StringFlag{
			Name:     "zstd-dictionary-path",
			Usage:    "path to the zstd dictionary file",
			EnvVars:  []string{"JETSTREAM_ZSTD_DICTIONARY_PATH"},
			Required: false,
		},
		&cli.DurationFlag{
			Name:    "event-ttl",
			Usage:   "time to live for events",
			Value:   24 * time.Hour,
			EnvVars: []string{"JETSTREAM_EVENT_TTL"},
		},
		&cli.Float64Flag{
			Name:    "max-sub-rate",
			Usage:   "max rate of events per second we can send to a subscriber",
			Value:   5_000,
			EnvVars: []string{"JETSTREAM_MAX_SUB_RATE"},
		},
		&cli.Int64Flag{
			Name:    "sub-outbox-size",
			Usage:   "channel depth (event count) per subscriber",
			Value:   50_000,
			EnvVars: []string{"JETSTREAM_SUB_OUTBOX_SIZE"},
		},
		&cli.Int64Flag{
			Name:    "override-relay-cursor",
			Usage:   "override cursor to start from, if not set will start from the last cursor in the database, if no cursor in the database will start from live",
			Value:   -1,
			EnvVars: []string{"JETSTREAM_OVERRIDE_RELAY_CURSOR"},
		},
		&cli.DurationFlag{
			Name:    "liveness-ttl",
			Usage:   "time to restart when no event detected",
			Value:   15 * time.Second,
			EnvVars: []string{"JETSTREAM_LIVENESS_TTL"},
		},
	}

	app.Action = Jetstream

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// Jetstream is the main function for jetstream
func Jetstream(cctx *cli.Context) error {
	ctx := cctx.Context

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	log.Info("starting jetstream")

	u, err := url.Parse(cctx.String("ws-url"))
	if err != nil {
		return fmt.Errorf("failed to parse ws-url: %w", err)
	}

	s, err := server.NewServer(cctx.Float64("max-sub-rate"), cctx.Int64("sub-outbox-size"))
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	c, err := consumer.NewConsumer(
		ctx,
		log,
		u.String(),
		cctx.String("data-dir"),
		cctx.Duration("event-ttl"),
		s.Emit,
	)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	s.Consumer = c

	scheduler := parallel.NewScheduler(cctx.Int("worker-count"), cctx.Int("max-queue-size"), "prod-firehose", c.HandleStreamEvent)

	// Start a goroutine to manage the cursor, saving the current cursor every 5 seconds.
	shutdownCursorManager := make(chan struct{})
	cursorManagerShutdown := make(chan struct{})
	go func() {
		ctx := context.Background()
		ticker := time.NewTicker(5 * time.Second)
		log := log.With("source", "cursor_manager")

		for {
			select {
			case <-shutdownCursorManager:
				log.Info("shutting down cursor manager")
				err := c.WriteCursor(ctx)
				if err != nil {
					log.Error("failed to write cursor", "error", err)
				}
				log.Info("cursor manager shut down successfully")
				close(cursorManagerShutdown)
				return
			case <-ticker.C:
				err := c.WriteCursor(ctx)
				if err != nil {
					log.Error("failed to write cursor", "error", err)
				}
			}
		}
	}()

	// Create a channel that will be closed when we want to stop the application
	// Usually when a critical routine returns an error
	livenessKill := make(chan struct{})

	// Start a goroutine to manage the liveness checker, shutting down if no events are received for liveness-ttl
	shutdownLivenessChecker := make(chan struct{})
	livenessCheckerShutdown := make(chan struct{})
	go func() {
		ticker := time.NewTicker(cctx.Duration("liveness-ttl"))
		lastSeq := int64(0)
		log := log.With("source", "liveness_checker")

		for {
			select {
			case <-shutdownLivenessChecker:
				log.Info("shutting down liveness checker")
				close(livenessCheckerShutdown)
				return
			case <-ticker.C:
				seq, _ := c.Progress.Get()
				if seq == lastSeq && seq != 0 {
					log.Error("no new events in last "+cctx.Duration("liveness-ttl").String()+", shutting down for docker to restart me", "seq", seq)
					close(livenessKill)
				} else {
					// Trim the database
					err := c.TrimEvents(ctx)
					if err != nil {
						log.Error("failed to trim events", "error", err)
					}
					log.Info("successful liveness check and trim", "seq", seq)
					lastSeq = seq
				}
			}
		}
	}()

	m := echo.New()
	m.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	m.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))

	metricsServer := &http.Server{
		Addr:    cctx.String("metrics-listen-addr"),
		Handler: m,
	}

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodOptions},
	}))
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to Jetstream")
	})
	e.GET("/subscribe", s.HandleSubscribe)

	jetServer := &http.Server{
		Addr:    cctx.String("listen-addr"),
		Handler: e,
	}

	// Startup echo server
	shutdownEcho := make(chan struct{})
	echoShutdown := make(chan struct{})
	go func() {
		logger := log.With("source", "echo_server")

		logger.Info("echo server listening", "addr", cctx.String("listen-addr"))

		go func() {
			if err := jetServer.ListenAndServe(); err != http.ErrServerClosed {
				logger.Error("failed to start echo server", "error", err)
			}
		}()

		<-shutdownEcho
		if err := jetServer.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown echo server", "error", err)
		}
		logger.Info("echo server shut down")
		close(echoShutdown)
	}()

	// Startup metrics server
	shutdownMetrics := make(chan struct{})
	metricsShutdown := make(chan struct{})
	go func() {
		logger := log.With("source", "metrics_server")

		logger.Info("metrics server listening", "addr", cctx.String("metrics-listen-addr"))

		go func() {
			if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
				logger.Error("failed to start metrics server", "error", err)
			}
		}()

		<-shutdownMetrics
		if err := metricsServer.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown metrics server", "error", err)
		}
		logger.Info("metrics server shut down")
		close(metricsShutdown)
	}()

	var cursor *int64
	cursorOverride := cctx.Int64("override-relay-cursor")

	// If the last cursor in the database is set, use that as the cursor
	if c.Progress.LastSeq >= 0 {
		cursor = &c.Progress.LastSeq
	}

	// If the override cursor is set, use that instead of the last cursor in the database
	if cursorOverride >= 0 {
		log.Info("overriding cursor", "cursor", cursorOverride)
		cursor = &cursorOverride
	}

	// If the cursor is nil, we are starting from live
	if cursor != nil {
		u.RawQuery = fmt.Sprintf("cursor=%d", *cursor)
	}

	log.Info("connecting to websocket", "url", u.String())
	con, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Error("failed to connect to websocket", "error", err)
		return err
	}
	defer con.Close()

	// Create a channel that will be closed when we want to stop the application
	// Usually when a critical routine returns an error
	eventsKill := make(chan struct{})

	shutdownRepoStream := make(chan struct{})
	repoStreamShutdown := make(chan struct{})
	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			logger := log.With("source", "repo_stream")
			err = events.HandleRepoStream(ctx, con, scheduler, logger)
			if !errors.Is(err, context.Canceled) {
				logger.Info("HandleRepoStream returned unexpectedly, killing jetstream", "error", err)
				close(eventsKill)
			} else {
				logger.Info("HandleRepoStream closed on context cancel")
			}
			close(repoStreamShutdown)
		}()
		<-shutdownRepoStream
		cancel()
	}()

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signals:
		log.Info("shutting down on signal")
	case <-ctx.Done():
		log.Info("shutting down on context done")
	case <-livenessKill:
		log.Info("shutting down on liveness kill")
	case <-eventsKill:
		log.Info("shutting down on events kill")
	}

	log.Info("shutting down, waiting for workers to clean up...")

	close(shutdownRepoStream)
	close(shutdownLivenessChecker)
	close(shutdownCursorManager)
	close(shutdownEcho)
	close(shutdownMetrics)

	shutdownTimeout := time.After(10 * time.Second)

	select {
	case <-repoStreamShutdown:
		log.Info("Repo stream shutdown completed")
	case <-shutdownTimeout:
		log.Warn("Shutdown timeout reached for repo stream")
	}

	select {
	case <-livenessCheckerShutdown:
		log.Info("Liveness checker shutdown completed")
	case <-shutdownTimeout:
		log.Warn("Shutdown timeout reached for liveness checker")
	}

	select {
	case <-cursorManagerShutdown:
		log.Info("Cursor manager shutdown completed")
	case <-shutdownTimeout:
		log.Warn("Shutdown timeout reached for cursor manager")
	}

	select {
	case <-echoShutdown:
		log.Info("Echo shutdown completed")
	case <-shutdownTimeout:
		log.Warn("Shutdown timeout reached for echo server")
	}

	select {
	case <-metricsShutdown:
		log.Info("Metrics shutdown completed")
	case <-shutdownTimeout:
		log.Warn("Shutdown timeout reached for metrics server")
	}

	c.Shutdown()

	err = c.UncompressedDB.Close()
	if err != nil {
		log.Error("failed to close pebble db", "error", err)
	}

	err = c.CompressedDB.Close()
	if err != nil {
		log.Error("failed to close compressed pebble db", "error", err)
	}

	log.Info("shut down successfully")

	return nil
}
