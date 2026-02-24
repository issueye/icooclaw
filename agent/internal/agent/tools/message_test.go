package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFileToolConfig tests for FileToolConfig
func TestFileToolConfig_Structure(t *testing.T) {
	config := FileToolConfig{
		AllowedRead:   true,
		AllowedWrite:  false,
		AllowedEdit:   true,
		AllowedDelete: false,
		Workspace:     "/tmp/test",
	}

	assert.True(t, config.AllowedRead)
	assert.False(t, config.AllowedWrite)
	assert.True(t, config.AllowedEdit)
	assert.False(t, config.AllowedDelete)
	assert.Equal(t, "/tmp/test", config.Workspace)
}

// TestShellExecConfig tests for ShellExecConfig
func TestShellExecConfig_Structure(t *testing.T) {
	config := ShellExecConfig{
		Allowed:   true,
		Timeout:   60,
		Workspace: "/home/user",
	}

	assert.True(t, config.Allowed)
	assert.Equal(t, 60, config.Timeout)
	assert.Equal(t, "/home/user", config.Workspace)
}

// TestMessageConfig tests for MessageConfig
func TestMessageConfig_Structure(t *testing.T) {
	config := MessageConfig{
		ChannelManager: nil,
	}

	assert.Nil(t, config.ChannelManager)
}
