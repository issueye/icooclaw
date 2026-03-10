// Package builtin provides built-in tools for icooclaw.
package builtin

import (
	"net/http"

	"icooclaw/pkg/tools"
)

// RegisterBuiltinTools registers all built-in tools.
func RegisterBuiltinTools(registry *tools.Registry) {
	registry.Register(NewHTTPTool())
	registry.Register(NewWebSearchTool())
	registry.Register(NewDateTimeTool())
}

func flattenHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}
