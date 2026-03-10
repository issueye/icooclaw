// Package channels provides channel management for icooclaw.
package channels

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/storage"
)

// Default rate limits per channel (messages per second).
var channelRateConfig = map[string]float64{
	"telegram": 20,
	"discord":  1,
	"slack":    1,
	"web":      100,
}

const (
	defaultRateLimit = 10
	maxRetries       = 3
	rateLimitDelay   = 1 * time.Second
	baseBackoff      = 500 * time.Millisecond
	maxBackoff       = 8 * time.Second
)

// Manager manages all channels.
type Manager struct {
	channels map[string]Channel
	workers  map[string]*channelWorker
	bus      *bus.MessageBus
	storage  *storage.Storage
	logger   *slog.Logger

	httpServer *http.Server
	mux        *http.ServeMux

	// State management
	placeholders  sync.Map // "channel:chatID" -> messageID
	typingStops   sync.Map // "channel:chatID" -> typingEntry
	reactionUndos sync.Map // "channel:chatID" -> func()

	running atomic.Bool
	mu      sync.RWMutex
}

type typingEntry struct {
	stop      func()
	createdAt time.Time
}

// NewManager creates a new channel manager.
func NewManager(b *bus.MessageBus, s *storage.Storage, logger *slog.Logger) *Manager {
	return &Manager{
		channels: make(map[string]Channel),
		workers:  make(map[string]*channelWorker),
		bus:      b,
		storage:  s,
		logger:   logger,
	}
}

// InitChannels initializes channels from database.
func (m *Manager) InitChannels(ctx context.Context) error {
	channels, err := m.storage.Channel().ListEnabledChannels()
	if err != nil {
		return err
	}

	for _, ch := range channels {
		factory, ok := GetFactory(ch.Type)
		if !ok {
			m.logger.Warn("unknown channel type", "type", ch.Type, "name", ch.Name)
			continue
		}

		channel, err := factory(parseConfig(ch.Config))
		if err != nil {
			m.logger.Error("failed to create channel", "name", ch.Name, "error", err)
			continue
		}

		m.channels[ch.Name] = channel
	}

	return nil
}

// StartAll starts all channels.
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, channel := range m.channels {
		if err := channel.Start(ctx); err != nil {
			m.logger.Error("failed to start channel", "name", name, "error", err)
			continue
		}

		w := newChannelWorker(name, channel)
		m.workers[name] = w

		go m.runWorker(ctx, name, w)
		go m.runMediaWorker(ctx, name, w)
	}

	// Start dispatchers
	go m.dispatchOutbound(ctx)
	go m.dispatchOutboundMedia(ctx)

	// Start TTL janitor
	go m.runTTLJanitor(ctx)

	// Start HTTP server if configured
	if m.httpServer != nil {
		go m.httpServer.ListenAndServe()
	}

	m.running.Store(true)
	return nil
}

// StopAll stops all channels.
func (m *Manager) StopAll(ctx context.Context) error {
	m.running.Store(false)

	// Stop HTTP server
	if m.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.httpServer.Shutdown(shutdownCtx)
	}

	// Close worker queues and wait
	m.mu.Lock()
	for _, w := range m.workers {
		close(w.queue)
		close(w.mediaQueue)
	}
	m.mu.Unlock()

	// Wait for workers to finish
	time.Sleep(100 * time.Millisecond)

	// Stop all channels
	m.mu.RLock()
	for name, channel := range m.channels {
		if err := channel.Stop(ctx); err != nil {
			m.logger.Error("failed to stop channel", "name", name, "error", err)
		}
	}
	m.mu.RUnlock()

	return nil
}

// dispatchOutbound dispatches outbound messages to workers.
func (m *Manager) dispatchOutbound(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-m.bus.Outbound():
			m.mu.RLock()
			w, ok := m.workers[msg.Channel]
			m.mu.RUnlock()

			if !ok {
				m.logger.Warn("unknown channel for outbound message", "channel", msg.Channel)
				continue
			}

			select {
			case w.queue <- msg:
			default:
				m.logger.Warn("worker queue full, dropping message", "channel", msg.Channel)
			}
		}
	}
}

// dispatchOutboundMedia dispatches outbound media messages to workers.
func (m *Manager) dispatchOutboundMedia(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-m.bus.OutboundMedia():
			m.mu.RLock()
			w, ok := m.workers[msg.Channel]
			m.mu.RUnlock()

			if !ok {
				m.logger.Warn("unknown channel for outbound media", "channel", msg.Channel)
				continue
			}

			select {
			case w.mediaQueue <- msg:
			default:
				m.logger.Warn("worker media queue full, dropping message", "channel", msg.Channel)
			}
		}
	}
}

// runWorker runs a channel worker.
func (m *Manager) runWorker(ctx context.Context, name string, w *channelWorker) {
	for msg := range w.queue {
		m.processOutbound(ctx, name, w, msg)
	}
	w.done <- struct{}{}
}

// runMediaWorker runs a media worker.
func (m *Manager) runMediaWorker(ctx context.Context, name string, w *channelWorker) {
	for msg := range w.mediaQueue {
		m.processOutboundMedia(ctx, name, w, msg)
	}
	w.mediaDone <- struct{}{}
}

// processOutbound processes an outbound message.
func (m *Manager) processOutbound(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	// Rate limiting
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	// Pre-send operations
	m.preSend(ctx, name, msg.ChatID)

	// Split message if needed
	maxLen := GetMaxMessageLength(name)
	chunks := SplitMessage(msg.Text, maxLen)

	for i, chunk := range chunks {
		msgCopy := msg
		msgCopy.Text = chunk

		// Only edit the first chunk if we have an edit ID
		if i > 0 {
			msgCopy.EditID = ""
		}

		m.sendWithRetry(ctx, name, w, msgCopy)
	}
}

// processOutboundMedia processes an outbound media message.
func (m *Manager) processOutboundMedia(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMediaMessage) {
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	// Check if channel supports media
	if ms, ok := w.channel.(MediaSender); ok {
		// Convert bus.OutboundMediaMessage to channels.OutboundMediaMessage
		mediaMsg := OutboundMediaMessage{
			Channel:  msg.Channel,
			ChatID:   msg.ChatID,
			Media:    msg.Media,
			Caption:  msg.Caption,
			Metadata: msg.Metadata,
		}
		if err := ms.SendMedia(ctx, mediaMsg); err != nil {
			m.logger.Error("failed to send media", "channel", name, "error", err)
		}
	}
}

// preSend performs pre-send operations (typing, reactions, placeholders).
func (m *Manager) preSend(ctx context.Context, name, chatID string) {
	key := name + ":" + chatID

	// Stop typing indicator
	if entry, ok := m.typingStops.LoadAndDelete(key); ok {
		if te, ok := entry.(typingEntry); ok {
			te.stop()
		}
	}

	// Undo reaction
	if undo, ok := m.reactionUndos.LoadAndDelete(key); ok {
		if fn, ok := undo.(func()); ok {
			fn()
		}
	}

	// Edit placeholder if exists
	if placeholderID, ok := m.placeholders.Load(key); ok {
		if editor, ok := m.channels[name].(MessageEditor); ok {
			editor.EditMessage(ctx, chatID, placeholderID.(string), "...")
		}
		m.placeholders.Delete(key)
	}
}

// sendWithRetry sends a message with retry logic.
func (m *Manager) sendWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Convert bus.OutboundMessage to channels.OutboundMessage
		chanMsg := OutboundMessage{
			Channel:  msg.Channel,
			ChatID:   msg.ChatID,
			Text:     msg.Text,
			Media:    msg.Media,
			ReplyTo:  msg.ReplyTo,
			EditID:   msg.EditID,
			Metadata: msg.Metadata,
		}
		lastErr = w.channel.Send(ctx, chanMsg)
		if lastErr == nil {
			return
		}

		// Permanent failure - don't retry
		if IsPermanent(lastErr) {
			m.logger.Error("permanent send failure", "channel", name, "error", lastErr)
			return
		}

		// Rate limit - fixed delay
		if errors.Is(lastErr, ErrRateLimit) {
			time.Sleep(rateLimitDelay)
			continue
		}

		// Temporary - exponential backoff
		backoff := baseBackoff * time.Duration(1<<uint(attempt))
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		time.Sleep(backoff)
	}

	m.logger.Error("send failed after retries", "channel", name, "error", lastErr)
}

// runTTLJanitor cleans up expired state entries.
func (m *Manager) runTTLJanitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			// Clean up old typing entries
			m.typingStops.Range(func(key, value any) bool {
				if entry, ok := value.(typingEntry); ok {
					if now.Sub(entry.createdAt) > 5*time.Minute {
						m.typingStops.LoadAndDelete(key)
						entry.stop()
					}
				}
				return true
			})
		}
	}
}

// RecordPlaceholder records a placeholder message ID.
func (m *Manager) RecordPlaceholder(channel, chatID, messageID string) {
	key := channel + ":" + chatID
	m.placeholders.Store(key, messageID)
}

// GetPlaceholder gets a placeholder message ID.
func (m *Manager) GetPlaceholder(channel, chatID string) string {
	key := channel + ":" + chatID
	if id, ok := m.placeholders.Load(key); ok {
		return id.(string)
	}
	return ""
}

// DeletePlaceholder deletes a placeholder message ID.
func (m *Manager) DeletePlaceholder(channel, chatID string) {
	key := channel + ":" + chatID
	m.placeholders.Delete(key)
}

// SetupHTTPServer sets up the shared HTTP server.
func (m *Manager) SetupHTTPServer(addr string) {
	m.mux = http.NewServeMux()
	m.httpServer = &http.Server{
		Addr:    addr,
		Handler: m.mux,
	}

	// Register webhook handlers
	for name, ch := range m.channels {
		if wh, ok := ch.(WebhookHandler); ok {
			m.mux.Handle(wh.WebhookPath(), wh)
			m.logger.Info("registered webhook", "channel", name, "path", wh.WebhookPath())
		}
	}

	// Health endpoint
	m.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// IsRunning returns true if the manager is running.
func (m *Manager) IsRunning() bool {
	return m.running.Load()
}

// channelWorker handles message processing for a channel.
type channelWorker struct {
	channel    Channel
	queue      chan bus.OutboundMessage
	mediaQueue chan bus.OutboundMediaMessage
	done       chan struct{}
	mediaDone  chan struct{}
	limiter    *rate.Limiter
}

func newChannelWorker(name string, channel Channel) *channelWorker {
	rateVal := float64(defaultRateLimit)
	if r, ok := channelRateConfig[name]; ok {
		rateVal = r
	}
	burst := int(math.Ceil(rateVal / 2))
	if burst < 1 {
		burst = 1
	}

	return &channelWorker{
		channel:    channel,
		queue:      make(chan bus.OutboundMessage, 100),
		mediaQueue: make(chan bus.OutboundMediaMessage, 50),
		done:       make(chan struct{}),
		mediaDone:  make(chan struct{}),
		limiter:    rate.NewLimiter(rate.Limit(rateVal), burst),
	}
}

func parseConfig(configStr string) map[string]any {
	// Simple JSON config parsing
	// In production, use proper JSON unmarshaling
	return make(map[string]any)
}
