// Package sse provides Server-Sent Events support for the gateway.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Event represents a Server-Sent Event.
type Event struct {
	ID    string      `json:"id,omitempty"`
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// Client represents an SSE client connection.
type Client struct {
	ID       string
	channel  chan Event
	done     chan struct{}
	lastSent time.Time
	logger   *slog.Logger
	mu       sync.Mutex
}

// NewClient creates a new SSE client.
func NewClient(id string, logger *slog.Logger) *Client {
	return &Client{
		ID:      id,
		channel: make(chan Event, 64),
		done:    make(chan struct{}),
		logger:  logger,
	}
}

// Send sends an event to the client.
func (c *Client) Send(event Event) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.channel <- event:
		c.lastSent = time.Now()
		return true
	default:
		c.logger.Warn("client buffer full, dropping event", "client_id", c.ID)
		return false
	}
}

// Close closes the client connection.
func (c *Client) Close() {
	close(c.done)
}

// Done returns the done channel.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// Events returns the event channel.
func (c *Client) Events() <-chan Event {
	return c.channel
}

// WriteTo writes SSE events to an http.ResponseWriter.
func (c *Client) WriteTo(w io.Writer, flusher http.Flusher) error {
	for {
		select {
		case <-c.done:
			return nil
		case event, ok := <-c.channel:
			if !ok {
				return nil
			}

			if err := WriteEvent(w, event); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

// WriteEvent writes a single SSE event to a writer.
func WriteEvent(w io.Writer, event Event) error {
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	if event.Event != "" {
		fmt.Fprintf(w, "event: %s\n", event.Event)
	}

	data, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)

	return nil
}

// Broker manages SSE clients and broadcasts events.
type Broker struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event

	logger *slog.Logger
	mu     sync.RWMutex
}

// NewBroker creates a new SSE broker.
func NewBroker(logger *slog.Logger) *Broker {
	if logger == nil {
		logger = slog.Default()
	}

	return &Broker{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		broadcast:  make(chan Event, 256),
		logger:     logger,
	}
}

// Run starts the broker.
func (b *Broker) Run(ctx context.Context) {
	b.logger.Debug("SSE broker started")

	for {
		select {
		case <-ctx.Done():
			b.logger.Debug("SSE broker stopped")
			return

		case client := <-b.register:
			b.mu.Lock()
			b.clients[client.ID] = client
			b.mu.Unlock()
			b.logger.Debug("SSE client registered", "client_id", client.ID, "total", len(b.clients))

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client.ID]; ok {
				delete(b.clients, client.ID)
			}
			b.mu.Unlock()
			b.logger.Debug("SSE client unregistered", "client_id", client.ID, "total", len(b.clients))

		case event := <-b.broadcast:
			b.mu.RLock()
			for _, client := range b.clients {
				select {
				case client.channel <- event:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()
		}
	}
}

// Register registers a client with the broker.
func (b *Broker) Register(client *Client) {
	b.register <- client
}

// Unregister unregisters a client from the broker.
func (b *Broker) Unregister(client *Client) {
	b.unregister <- client
}

// Broadcast sends an event to all connected clients.
func (b *Broker) Broadcast(event Event) {
	select {
	case b.broadcast <- event:
	default:
		b.logger.Warn("broadcast channel full, dropping event")
	}
}

// BroadcastTo sends an event to a specific client.
func (b *Broker) BroadcastTo(clientID string, event Event) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	client, ok := b.clients[clientID]
	if !ok {
		return false
	}

	return client.Send(event)
}

// GetClientCount returns the number of connected clients.
func (b *Broker) GetClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Handler returns an http.Handler for SSE connections.
func (b *Broker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Create client
		client := NewClient(generateClientID(), b.logger)
		b.Register(client)
		defer func() {
			b.Unregister(client)
			client.Close()
		}()

		// Send initial connection event
		client.Send(Event{
			Event: "connected",
			Data:  map[string]string{"client_id": client.ID},
		})

		// Handle client disconnect
		go func() {
			<-r.Context().Done()
			client.Close()
		}()

		// Write events to client
		client.WriteTo(w, flusher)
	}
}

// StreamHandler creates a handler that streams events from a source function.
func StreamHandler(source func(ctx context.Context) (<-chan Event, error), logger *slog.Logger) http.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Get event channel from source
		events, err := source(r.Context())
		if err != nil {
			logger.Error("failed to create event source", "error", err)
			http.Error(w, "failed to create event source", http.StatusInternalServerError)
			return
		}

		// Stream events
		for {
			select {
			case <-r.Context().Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}

				if err := WriteEvent(w, event); err != nil {
					logger.Error("failed to write event", "error", err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

func generateClientID() string {
	return fmt.Sprintf("sse-%d", time.Now().UnixNano())
}