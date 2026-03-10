// Package middleware provides HTTP middleware for the gateway.
package middleware

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
)

// contextKey is a type for context keys.
type contextKey string

const (
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "user_id"
	// APIKeyIDKey is the context key for API key ID.
	APIKeyIDKey contextKey = "api_key_id"
)

// APIKeyAuthConfig holds configuration for API key authentication.
type APIKeyAuthConfig struct {
	// Keys is a map of API key to user ID.
	Keys map[string]string
	// Header is the header name for the API key (default: "X-API-Key").
	Header string
	// QueryParam is the query parameter name for the API key (default: "api_key").
	QueryParam string
	// Realm is the realm for WWW-Authenticate header.
	Realm string
}

// DefaultAPIKeyAuthConfig returns the default API key auth configuration.
func DefaultAPIKeyAuthConfig() *APIKeyAuthConfig {
	return &APIKeyAuthConfig{
		Keys:       make(map[string]string),
		Header:     "X-API-Key",
		QueryParam: "api_key",
		Realm:      "API",
	}
}

// APIKeyAuth returns a middleware that authenticates requests using API keys.
func APIKeyAuth(cfg *APIKeyAuthConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultAPIKeyAuthConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from header or query parameter
			apiKey := r.Header.Get(cfg.Header)
			if apiKey == "" {
				apiKey = r.URL.Query().Get(cfg.QueryParam)
			}

			if apiKey == "" {
				logger.Debug("missing API key")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Look up user ID
			userID, ok := cfg.Keys[apiKey]
			if !ok {
				logger.Debug("invalid API key", "api_key_prefix", apiKey[:min(8, len(apiKey))])
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, APIKeyIDKey, apiKey)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyAuthFunc returns a middleware that authenticates using a custom function.
func APIKeyAuthFunc(authFunc func(apiKey string) (userID string, ok bool), logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from header or query parameter
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey == "" {
				logger.Debug("missing API key")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Authenticate using custom function
			userID, ok := authFunc(apiKey)
			if !ok {
				logger.Debug("invalid API key")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, APIKeyIDKey, apiKey)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BasicAuthConfig holds configuration for basic authentication.
type BasicAuthConfig struct {
	// Users is a map of username to password.
	Users map[string]string
	// Realm is the realm for WWW-Authenticate header.
	Realm string
}

// DefaultBasicAuthConfig returns the default basic auth configuration.
func DefaultBasicAuthConfig() *BasicAuthConfig {
	return &BasicAuthConfig{
		Users: make(map[string]string),
		Realm: "API",
	}
}

// BasicAuth returns a middleware that authenticates requests using HTTP Basic Auth.
func BasicAuth(cfg *BasicAuthConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultBasicAuthConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+cfg.Realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Check credentials
			expectedPassword, ok := cfg.Users[username]
			if !ok {
				logger.Debug("unknown user", "username", username)
				w.Header().Set("WWW-Authenticate", `Basic realm="`+cfg.Realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Use constant-time comparison
			if subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) != 1 {
				logger.Debug("invalid password", "username", username)
				w.Header().Set("WWW-Authenticate", `Basic realm="`+cfg.Realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BearerAuthConfig holds configuration for bearer token authentication.
type BearerAuthConfig struct {
	// Tokens is a map of token to user ID.
	Tokens map[string]string
	// Realm is the realm for WWW-Authenticate header.
	Realm string
}

// DefaultBearerAuthConfig returns the default bearer auth configuration.
func DefaultBearerAuthConfig() *BearerAuthConfig {
	return &BearerAuthConfig{
		Tokens: make(map[string]string),
		Realm:  "API",
	}
}

// BearerAuth returns a middleware that authenticates requests using Bearer tokens.
func BearerAuth(cfg *BearerAuthConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultBearerAuthConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+cfg.Realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+cfg.Realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Look up user ID
			userID, ok := cfg.Tokens[token]
			if !ok {
				logger.Debug("invalid bearer token")
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+cfg.Realm+`"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BearerAuthFunc returns a middleware that authenticates using a custom function.
func BearerAuthFunc(authFunc func(token string) (userID string, ok bool), logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="API"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				w.Header().Set("WWW-Authenticate", `Bearer realm="API"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Authenticate using custom function
			userID, ok := authFunc(token)
			if !ok {
				logger.Debug("invalid bearer token")
				w.Header().Set("WWW-Authenticate", `Bearer realm="API"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID returns the user ID from the context.
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetAPIKeyID returns the API key ID from the context.
func GetAPIKeyID(ctx context.Context) string {
	if apiKeyID, ok := ctx.Value(APIKeyIDKey).(string); ok {
		return apiKeyID
	}
	return ""
}

// OptionalAuth returns a middleware that optionally authenticates requests.
// If authentication fails, the request continues without a user ID.
func OptionalAuth(authFunc func(r *http.Request) (userID string, ok bool), logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := authFunc(r)
			if ok && userID != "" {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth returns a middleware that requires authentication.
func RequireAuth(authFunc func(r *http.Request) (userID string, ok bool), logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := authFunc(r)
			if !ok || userID == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Chain chains multiple middleware together.
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}