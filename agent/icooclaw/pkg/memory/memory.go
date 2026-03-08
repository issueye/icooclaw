// Package memory provides memory management for icooclaw.
package memory

import (
	"context"
	"encoding/json"
	"log/slog"

	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
)

// Loader loads and saves memory for sessions.
type Loader interface {
	Load(ctx context.Context, sessionKey string) ([]providers.ChatMessage, error)
	Save(ctx context.Context, sessionKey, role, content string) error
	Clear(ctx context.Context, sessionKey string) error
}

// DefaultLoader is the default memory loader implementation.
type DefaultLoader struct {
	storage  *storage.Storage
	maxItems int
	logger   *slog.Logger
}

// NewLoader creates a new memory loader.
func NewLoader(s *storage.Storage, maxItems int, logger *slog.Logger) *DefaultLoader {
	if maxItems <= 0 {
		maxItems = 100
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultLoader{
		storage:  s,
		maxItems: maxItems,
		logger:   logger,
	}
}

// Load loads memory for a session.
func (l *DefaultLoader) Load(ctx context.Context, sessionKey string) ([]providers.ChatMessage, error) {
	memories, err := l.storage.GetMemory(sessionKey, l.maxItems)
	if err != nil {
		return nil, err
	}

	// Reverse order (oldest first)
	messages := make([]providers.ChatMessage, 0, len(memories))
	for i := len(memories) - 1; i >= 0; i-- {
		m := memories[i]
		messages = append(messages, providers.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	return messages, nil
}

// Save saves a memory entry.
func (l *DefaultLoader) Save(ctx context.Context, sessionKey, role, content string) error {
	return l.storage.SaveMemory(&storage.Memory{
		SessionID: sessionKey,
		Role:      role,
		Content:   content,
	})
}

// Clear clears memory for a session.
func (l *DefaultLoader) Clear(ctx context.Context, sessionKey string) error {
	return l.storage.DeleteMemory(sessionKey)
}

// Summarizer generates summaries of conversations.
type Summarizer interface {
	Summarize(ctx context.Context, messages []providers.ChatMessage) (string, error)
}

// DefaultSummarizer uses an LLM to generate summaries.
type DefaultSummarizer struct {
	provider providers.Provider
	model    string
	logger   *slog.Logger
}

// NewSummarizer creates a new summarizer.
func NewSummarizer(p providers.Provider, model string, logger *slog.Logger) *DefaultSummarizer {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultSummarizer{
		provider: p,
		model:    model,
		logger:   logger,
	}
}

// Summarize generates a summary of the conversation.
func (s *DefaultSummarizer) Summarize(ctx context.Context, messages []providers.ChatMessage) (string, error) {
	// Build summary prompt
	var content string
	for _, m := range messages {
		content += m.Role + ": " + m.Content + "\n"
	}

	req := providers.ChatRequest{
		Model: s.model,
		Messages: []providers.ChatMessage{
			{
				Role: "system",
				Content: "You are a helpful assistant that summarizes conversations. " +
					"Provide a concise summary of the key points discussed.",
			},
			{
				Role:    "user",
				Content: "Please summarize this conversation:\n\n" + content,
			},
		},
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// Manager manages memory with summarization.
type Manager struct {
	loader     Loader
	summarizer Summarizer
	storage    *storage.Storage
	maxItems   int
	logger     *slog.Logger
}

// NewManager creates a new memory manager.
func NewManager(loader Loader, summarizer Summarizer, s *storage.Storage, maxItems int, logger *slog.Logger) *Manager {
	if maxItems <= 0 {
		maxItems = 100
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		loader:     loader,
		summarizer: summarizer,
		storage:    s,
		maxItems:   maxItems,
		logger:     logger,
	}
}

// Load loads memory for a session.
func (m *Manager) Load(ctx context.Context, sessionKey string) ([]providers.ChatMessage, error) {
	return m.loader.Load(ctx, sessionKey)
}

// Save saves a memory entry.
func (m *Manager) Save(ctx context.Context, sessionKey, role, content string) error {
	return m.loader.Save(ctx, sessionKey, role, content)
}

// Clear clears memory for a session.
func (m *Manager) Clear(ctx context.Context, sessionKey string) error {
	return m.loader.Clear(ctx, sessionKey)
}

// SummarizeAndCompress summarizes old messages and compresses memory.
func (m *Manager) SummarizeAndCompress(ctx context.Context, sessionKey string) error {
	// Load all messages
	memories, err := m.storage.GetMemory(sessionKey, 0)
	if err != nil {
		return err
	}

	if len(memories) <= m.maxItems {
		return nil
	}

	// Get messages to summarize (keep recent ones)
	toSummarize := memories[m.maxItems:]
	messages := make([]providers.ChatMessage, 0, len(toSummarize))
	for i := len(toSummarize) - 1; i >= 0; i-- {
		mem := toSummarize[i]
		messages = append(messages, providers.ChatMessage{
			Role:    mem.Role,
			Content: mem.Content,
		})
	}

	// Generate summary
	summary, err := m.summarizer.Summarize(ctx, messages)
	if err != nil {
		return err
	}

	// Delete old messages
	for _, mem := range toSummarize {
		m.storage.DeleteMemory(mem.SessionID)
	}

	// Save summary as system message
	return m.storage.SaveMemory(&storage.Memory{
		SessionID: sessionKey,
		Role:      "system",
		Content:   "Previous conversation summary: " + summary,
		Metadata:  mustMarshalJSON(map[string]any{"type": "summary"}),
	})
}

func mustMarshalJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
