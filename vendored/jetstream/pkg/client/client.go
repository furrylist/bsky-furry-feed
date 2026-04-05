package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/atomic"
)

type ClientConfig struct {
	Compress          bool
	WebsocketURL      string
	WantedDids        []string
	WantedCollections []string
	MaxSize           uint32
	ExtraHeaders      map[string]string
}

type Scheduler interface {
	AddWork(ctx context.Context, repo string, evt *models.Event) error
	Shutdown()
}

type Client struct {
	Scheduler  Scheduler
	con        *websocket.Conn
	config     *ClientConfig
	logger     *slog.Logger
	decoder    *zstd.Decoder
	BytesRead  atomic.Int64
	EventsRead atomic.Int64
	shutdown   chan chan struct{}
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Compress:          true,
		WebsocketURL:      "ws://localhost:6008/subscribe",
		WantedDids:        []string{},
		WantedCollections: []string{},
		MaxSize:           0,
		ExtraHeaders: map[string]string{
			"User-Agent": "jetstream-client/v0.0.1",
		},
	}
}

func NewClient(config *ClientConfig, logger *slog.Logger, scheduler Scheduler) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	logger = logger.With("component", "jetstream-client")
	c := Client{
		config:    config,
		shutdown:  make(chan chan struct{}),
		logger:    logger,
		Scheduler: scheduler,
	}

	if config.Compress {
		c.config.ExtraHeaders["Socket-Encoding"] = "zstd"
		dec, err := zstd.NewReader(nil, zstd.WithDecoderDicts(models.ZSTDDictionary))
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
		}
		c.decoder = dec
	}

	return &c, nil
}

func (c *Client) ConnectAndRead(ctx context.Context, cursor *int64) error {
	header := http.Header{}
	for k, v := range c.config.ExtraHeaders {
		header.Add(k, v)
	}

	fullURL := c.config.WebsocketURL
	params := []string{}
	if cursor != nil {
		params = append(params, fmt.Sprintf("cursor=%d", *cursor))
	}

	for _, did := range c.config.WantedDids {
		params = append(params, fmt.Sprintf("wantedDids=%s", did))
	}

	for _, collection := range c.config.WantedCollections {
		params = append(params, fmt.Sprintf("wantedCollections=%s", collection))
	}

	if c.config.MaxSize > 0 {
		params = append(params, fmt.Sprintf("maxSize=%d", c.config.MaxSize))
	}

	if len(params) > 0 {
		fullURL += "?" + params[0]
		for _, p := range params[1:] {
			fullURL += "&" + p
		}
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return fmt.Errorf("failed to parse connection url %q: %w", c.config.WebsocketURL, err)
	}

	c.logger.Info("connecting to websocket", "url", u.String())
	con, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return err
	}

	c.con = con

	var readErr error = nil
	if err := c.readLoop(ctx); err != nil {
		readErr = fmt.Errorf("read loop failed: %w", err)
	}

	var closeErr error = nil
	if err := c.con.Close(); err != nil {
		closeErr = fmt.Errorf("failed to close connection: %w", err)
	}

	return errors.Join(readErr, closeErr)
}

func (c *Client) readLoop(ctx context.Context) error {
	c.logger.Info("starting websocket read loop")

	bytesRead := bytesRead.WithLabelValues(c.config.WebsocketURL)
	eventsRead := eventsRead.WithLabelValues(c.config.WebsocketURL)

	go func() {
		select {
		case <-ctx.Done():
			_ = c.con.SetReadDeadline(time.Now())
		case s := <-c.shutdown:
			c.logger.Info("shutting down read loop on shutdown signal")
			_ = c.con.SetReadDeadline(time.Now())
			s <- struct{}{}
		}
	}()

	for {
		_, msg, err := c.con.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("shutting down read loop on context completion")
				return nil
			}
			c.logger.Error("failed to read message from websocket", "error", err)
			return fmt.Errorf("failed to read message from websocket: %w", err)
		}

		bytesRead.Add(float64(len(msg)))
		eventsRead.Inc()
		c.BytesRead.Add(int64(len(msg)))
		c.EventsRead.Inc()

		if c.decoder != nil && c.config.Compress {
			m, err := c.decoder.DecodeAll(msg, nil)
			if err != nil {
				c.logger.Error("failed to decompress message", "error", err)
				return fmt.Errorf("failed to decompress message: %w", err)
			}
			msg = m
		}

		var event models.Event
		if err := json.Unmarshal(msg, &event); err != nil {
			c.logger.Error("failed to unmarshal event", "error", err)
			return fmt.Errorf("failed to unmarshal event: %w", err)
		}

		if err := c.Scheduler.AddWork(ctx, event.Did, &event); err != nil {
			c.logger.Error("failed to add work to scheduler", "error", err)
			return fmt.Errorf("failed to add work to scheduler: %w", err)
		}
	}
}
