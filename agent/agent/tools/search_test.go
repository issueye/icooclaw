package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSearchTool tests for WebSearchTool
func TestWebSearchTool_NewWebSearchTool(t *testing.T) {
	tool := NewWebSearchTool()
	require.NotNil(t, tool)
	assert.Equal(t, "web_search", tool.Name())
}

func TestWebSearchTool_Execute(t *testing.T) {
	tool := NewWebSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test query",
	})

	require.NoError(t, err)
	assert.Contains(t, result, "test query")
	assert.Contains(t, result, `"num_results"`)
}

func TestWebSearchTool_Execute_InvalidQuery(t *testing.T) {
	tool := NewWebSearchTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "",
	})

	assert.Error(t, err)
}

// TestExtractTextFromHTML tests the extractTextFromHTML function
func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"Simple text", "Hello World", "Hello World"},
		{"With script", "Text<script>alert('x')</script>more", "Textmore"},
		{"BR tags", "Line1<br>Line2", "Line1\nLine2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextFromHTML(tt.html)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestRemoveTag tests the removeTag function
func TestRemoveTag(t *testing.T) {
	tests := []struct {
		name string
		html string
		tag  string
		want string
	}{
		{"Simple script", `before<script>alert('x')</script>after`, "script", "beforeafter"},
		{"No tag", "plain text", "script", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTag(tt.html, tt.tag)
			assert.Equal(t, tt.want, got)
		})
	}
}
