package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPRequestTool tests for HTTPRequestTool
func TestHTTPRequestTool_NewHTTPRequestTool(t *testing.T) {
	tool := NewHTTPRequestTool()
	require.NotNil(t, tool)
	assert.Equal(t, "http_request", tool.Name())
}

func TestHTTPRequestTool_Execute_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tool := NewHTTPRequestTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	require.NoError(t, err)
	assert.Contains(t, result, `"status_code":200`)
	assert.Contains(t, result, "ok")
}

func TestHTTPRequestTool_Execute_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"received": "true"})
	}))
	defer server.Close()

	tool := NewHTTPRequestTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"body":   `{"key":"value"}`,
	})

	require.NoError(t, err)
	assert.Contains(t, result, `"status_code":200`)
}

func TestHTTPRequestTool_Execute_InvalidURL(t *testing.T) {
	tool := NewHTTPRequestTool()
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": "",
	})

	assert.Error(t, err)
}

func TestHTTPRequestTool_Execute_JSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]string{"name": "John"},
		})
	}))
	defer server.Close()

	tool := NewHTTPRequestTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	require.NoError(t, err)
	assert.Contains(t, result, `"body_formatted"`)
	assert.Contains(t, result, "John")
}

func TestHTTPRequestTool_Execute_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, method, r.Method)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"method": r.Method})
			}))
			defer server.Close()

			tool := NewHTTPRequestTool()
			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"url":    server.URL,
				"method": method,
			})

			require.NoError(t, err)
			assert.Contains(t, result, `"status_code":200`)
		})
	}
}

func TestHTTPRequestTool_Execute_DefaultMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tool := NewHTTPRequestTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	require.NoError(t, err)
	assert.Contains(t, result, `"status_code":200`)
}
