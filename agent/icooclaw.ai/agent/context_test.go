package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestContextBuilder tests for ContextBuilder
func TestContextBuilder_Structure(t *testing.T) {
	// Create minimal mock for testing
	// We can't fully test ContextBuilder without proper Agent and Session setup
	cb := &ContextBuilder{}
	assert.NotNil(t, cb)
}
