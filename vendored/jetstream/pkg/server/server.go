package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bluesky-social/jetstream/pkg/consumer"
	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

type Server struct {
	Subscribers   map[int64]*Subscriber
	lk            sync.RWMutex
	nextSub       int64
	Consumer      *consumer.Consumer
	maxSubRate    float64
	subOutboxSize int64
	seq           int64
	perIPLimiters map[string]*rate.Limiter
}

var upgrader = websocket.Upgrader{
	// Allow all origins in websocket upgrade requests
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var maxConcurrentEmits = int64(100)
var cutoverThresholdUS = int64(1_000_000)
var tracer = otel.Tracer("jetstream-server")

func NewServer(maxSubRate float64, subOutboxSize int64) (*Server, error) {
	s := Server{
		Subscribers:   make(map[int64]*Subscriber),
		maxSubRate:    maxSubRate,
		subOutboxSize: subOutboxSize,
		perIPLimiters: make(map[string]*rate.Limiter),
	}

	return &s, nil
}

func (s *Server) GetSeq() int64 {
	s.lk.RLock()
	defer s.lk.RUnlock()
	return s.seq
}

func (s *Server) SetSeq(seq int64) {
	s.lk.Lock()
	defer s.lk.Unlock()
	s.seq = seq
}

func (s *Server) HandleSubscribe(c echo.Context) error {
	ctx, cancel := context.WithCancel(c.Request().Context())
	defer cancel()

	qWantedCollections := c.Request().URL.Query()["wantedCollections"]
	qWantedDids := c.Request().URL.Query()["wantedDids"]

	qMaxMessageSizeBytes := c.Request().URL.Query().Get("maxMessageSizeBytes")
	qMaxMessageSizeBytesValue := ParseMaxMessageSizeBytes(qMaxMessageSizeBytes)

	// Check if the user wants to initialize options over the websocket before subscribing
	requireHello := c.Request().URL.Query().Get("requireHello") == "true"

	// Check if the user wants zstd compression, can be set in the header or query param
	socketEncoding := c.Request().Header.Get("Socket-Encoding")
	compress := strings.Contains(socketEncoding, "zstd") || c.Request().URL.Query().Get("compress") == "true"

	// Parse the cursor
	var cursor *int64
	var err error
	qCursor := c.Request().URL.Query().Get("cursor")
	if qCursor != "" {
		cursor = new(int64)
		*cursor, err = strconv.ParseInt(qCursor, 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("invalid cursor: %s", qCursor))
			return fmt.Errorf("invalid cursor: %s", qCursor)
		}

		// If given a future cursor, just live tail
		if *cursor > time.Now().UnixMicro() {
			cursor = nil
		}
	}

	// Parse the subscriber options
	subscriberOpts, err := parseSubscriberOptions(ctx, qWantedCollections, qWantedDids, compress, qMaxMessageSizeBytesValue, cursor)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	subIP := c.RealIP()

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	writeWait := 10 * time.Second

	log := slog.With("source", "server_handle_subscribe", "socket_addr", ws.RemoteAddr().String(), "real_ip", subIP)

	sub, err := s.AddSubscriber(ws, subIP, subscriberOpts)
	if err != nil {
		log.Error("failed to add subscriber", "error", err)
		return err
	}
	defer s.RemoveSubscriber(sub.id)

	// Handle incoming messages
	go func() {
		for {
			msgType, msgBytes, err := ws.ReadMessage()
			if err != nil {
				log.Error("failed to read message from websocket", "error", err)
				cancel()
				return
			}

			if len(msgBytes) > 10_000_000 {
				log.Error("message from subscriber too large, ignoring", "size", len(msgBytes))
				continue
			}

			switch msgType {
			case websocket.PingMessage:
				if err := sub.WriteMessage(websocket.PongMessage, nil); err != nil {
					log.Error("failed to write pong to websocket", "error", err)
					cancel()
					return
				}
			case websocket.PongMessage:
				log.Debug("received pong message from client")
			case websocket.CloseMessage:
				log.Info("received close message from client")
				cancel()
				return
			case websocket.TextMessage:
				log.Info("received text message from client, parsing as subscriber options update")
				log.Debug("text message from client content", "msg", string(msgBytes))

				var subMessage SubscriberSourcedMessage
				if err := json.Unmarshal(msgBytes, &subMessage); err != nil {
					log.Error("failed to unmarshal subscriber sourced message", "error", err, "msg", string(msgBytes))
					sub.Terminate(fmt.Sprintf("failed to unmarshal subscriber sourced message: %v", err))
					cancel()
					return
				}

				switch subMessage.Type {
				case SubMessageOptionsUpdate:
					var subOptsUpdate SubscriberOptionsUpdatePayload
					if err := json.Unmarshal(subMessage.Payload, &subOptsUpdate); err != nil {
						log.Error("failed to unmarshal subscriber options update", "error", err, "msg", string(msgBytes))
						sub.Terminate(fmt.Sprintf("failed to unmarshal subscriber options update: %v", err))
						cancel()
						return
					}

					maxMessageSizeBytes := ParseMaxMessageSizeBytes(subOptsUpdate.MaxMessageSizeBytes)
					// Only WantedCollections and WantedDIDs can be updated after the initial connection
					// Cursor and Compression settings are fixed for the lifetime of the stream
					subscriberOpts, err = parseSubscriberOptions(ctx, subOptsUpdate.WantedCollections, subOptsUpdate.WantedDIDs, compress, maxMessageSizeBytes, sub.cursor)
					if err != nil {
						log.Error("failed to parse subscriber options", "error", err, "new_opts", subOptsUpdate)
						sub.Terminate(fmt.Sprintf("failed to parse subscriber options: %v", err))
						cancel()
						return
					}

					sub.UpdateOptions(subscriberOpts)

					if requireHello {
						sub.hello <- struct{}{}
						requireHello = false
					}
				default:
					log.Warn("received unexpected message type from client, ignoring", "type", subMessage.Type)
				}
			case websocket.BinaryMessage:
				log.Warn("received unexpected binary message from client, ignoring")
			}
		}
	}()

	// If requireHello is set, wait for the client to send a valid options update message before proceeding
	if requireHello {
		log.Info("waiting for hello message from client")
		select {
		case <-ctx.Done():
			log.Info("shutting down subscriber")
			return nil
		case <-sub.hello:
			log.Info("received hello message from client, proceeding")
		}
	}

	// Replay events if a cursor is provided
	if sub.cursor != nil {
		log.Info("replaying events", "cursor", *sub.cursor)
		playbackRateLimit := s.maxSubRate * 10

		go func() {
			for {
				lastSeq, err := s.Consumer.ReplayEvents(ctx, sub.compress, *sub.cursor, playbackRateLimit, func(ctx context.Context, timeUS int64, did, collection string, getEventBytes func() []byte) error {
					return emitToSubscriber(ctx, log, sub, timeUS, did, collection, true, getEventBytes)
				})
				if err != nil {
					log.Error("failed to replay events", "error", err)
					sub.Terminate("failed to replay events")
					cancel()
					return
				}
				serverLastSeq := s.GetSeq()
				log.Info("finished replaying events", "replay_last_time", time.UnixMicro(lastSeq), "server_last_time", time.UnixMicro(serverLastSeq))

				// If last event replayed is close enough to the last live event, start live tailing
				if lastSeq > serverLastSeq-(cutoverThresholdUS/2) {
					break
				}

				// Otherwise, update the cursor and replay again
				lastSeq++
				sub.SetCursor(&lastSeq)
			}
			log.Info("finished replaying events, starting live tail")
			sub.SetCursor(nil)
		}()
	}

	// Read events from the outbox and send them to the subscriber
	pingPeriod := 30 * time.Second
	t := time.NewTicker(pingPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down subscriber")
			return nil
		case msg := <-sub.outbox:
			err := sub.rl.Wait(ctx)
			if err != nil {
				log.Error("failed to wait for rate limiter", "error", err)
				return fmt.Errorf("failed to wait for rate limiter: %w", err)
			}

			ws.SetWriteDeadline(time.Now().Add(writeWait))

			// When compression is enabled, the msg is a zstd compressed message
			if compress {
				if err := sub.WriteMessage(websocket.BinaryMessage, *msg); err != nil {
					log.Error("failed to write message to websocket", "error", err)
					return nil
				}
				continue
			}

			// Otherwise, the msg is serialized JSON
			if err := sub.WriteMessage(websocket.TextMessage, *msg); err != nil {
				log.Error("failed to write message to websocket", "error", err)
				return nil
			}
		case <-t.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := sub.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error("failed to write ping to websocket", "error", err)
				return nil
			}
		}
	}
}

func (s *Server) Emit(ctx context.Context, e *models.Event, asJSON, compBytes []byte) error {
	ctx, span := tracer.Start(ctx, "Emit")
	defer span.End()

	log := slog.With("source", "server_emit")

	s.lk.RLock()
	defer func() {
		s.lk.RUnlock()
		s.SetSeq(e.TimeUS)
	}()

	eventsEmitted.Inc()
	evtSize := float64(len(asJSON))
	bytesEmitted.Add(evtSize)

	collection := ""
	if e.Kind == models.EventKindCommit && e.Commit != nil {
		collection = e.Commit.Collection
	}

	// Wrap the valuer functions for more lightweight event filtering
	getJSONEvent := func() []byte { return asJSON }
	getCompressedEvent := func() []byte { return compBytes }

	// Concurrently emit to all subscribers
	// We can't move on until all subscribers have received the event or been dropped for being too slow
	sem := semaphore.NewWeighted(maxConcurrentEmits)
	for _, sub := range s.Subscribers {
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Error("failed to acquire semaphore", "error", err)
			return fmt.Errorf("failed to acquire semaphore: %w", err)
		}
		go func(sub *Subscriber) {
			defer sem.Release(1)
			sub.lk.Lock()
			defer sub.lk.Unlock()

			// Don't emit events to subscribers that are replaying and are too far behind
			if sub.cursor != nil && sub.seq < e.TimeUS-cutoverThresholdUS || sub.tearingDown {
				return
			}

			// Pick the event valuer for the subscriber based on their compression preference
			getEventBytes := getJSONEvent
			if sub.compress {
				getEventBytes = getCompressedEvent
			}

			emitToSubscriber(ctx, log, sub, e.TimeUS, e.Did, collection, false, getEventBytes)
		}(sub)
	}

	if err := sem.Acquire(ctx, maxConcurrentEmits); err != nil {
		log.Error("failed to acquire semaphore", "error", err)
		return fmt.Errorf("failed to acquire semaphore: %w", err)
	}

	return nil
}
