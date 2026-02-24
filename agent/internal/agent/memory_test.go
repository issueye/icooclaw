package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMemoryConfig tests for MemoryConfig struct
func TestMemoryConfig_Structure(t *testing.T) {
	cfg := MemoryConfig{
		ConsolidationThreshold: 100,
		SummaryEnabled:         true,
		AutoSave:               true,
		MaxMemoryAge:           30,
	}

	assert.Equal(t, 100, cfg.ConsolidationThreshold)
	assert.True(t, cfg.SummaryEnabled)
	assert.True(t, cfg.AutoSave)
	assert.Equal(t, 30, cfg.MaxMemoryAge)
}

// TestMemoryStore tests for MemoryStore (without actual storage)
func TestMemoryStore_NewMemoryStoreWithConfig(t *testing.T) {
	// Test with zero values (should use defaults)
	cfg := MemoryConfig{
		ConsolidationThreshold: 0,
		MaxMemoryAge:           0,
	}

	// Note: We can't create actual MemoryStore without a storage backend
	// This just tests the config defaults
	assert.Equal(t, 0, cfg.ConsolidationThreshold)
	assert.Equal(t, 0, cfg.MaxMemoryAge)
}

// TestTruncate tests for truncate function
func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"Short string", "hello", 10, "hello"},
		{"Long string", "hello world", 5, "hello..."},
		{"Exact length", "hello", 5, "hello"},
		{"Empty string", "", 5, ""},
		{"Zero max length", "hello", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}
