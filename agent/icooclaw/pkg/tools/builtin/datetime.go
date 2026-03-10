package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/tools"
	"time"
)

// DateTimeTool provides date/time functionality.
type DateTimeTool struct{}

// NewDateTimeTool creates a new datetime tool.
func NewDateTimeTool() *DateTimeTool {
	return &DateTimeTool{}
}

// Name returns the tool name.
func (t *DateTimeTool) Name() string {
	return "datetime"
}

// Description returns the tool description.
func (t *DateTimeTool) Description() string {
	return "Get current date and time information."
}

// Parameters returns the tool parameters.
func (t *DateTimeTool) Parameters() map[string]tools.Parameter {
	return map[string]tools.Parameter{
		"timezone": {
			Type:        "string",
			Description: "Timezone (e.g., 'UTC', 'America/New_York')",
		},
		"format": {
			Type:        "string",
			Description: "Time format (e.g., '2006-01-02 15:04:05')",
		},
	}
}

// Execute executes the datetime tool.
func (t *DateTimeTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	now := time.Now()

	// Handle timezone
	if tz, ok := args["timezone"].(string); ok {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return &tools.Result{Success: false, Error: fmt.Errorf("invalid timezone: %v", err)}
		}
		now = now.In(loc)
	}

	// Handle format
	format := time.RFC3339
	if f, ok := args["format"].(string); ok {
		format = f
	}

	result := map[string]any{
		"formatted": now.Format(format),
		"timestamp": now.Unix(),
		"date":      now.Format("2006-01-02"),
		"time":      now.Format("15:04:05"),
		"weekday":   now.Weekday().String(),
		"unix_nano": now.UnixNano(),
		"year":      now.Year(),
		"month":     int(now.Month()),
		"day":       now.Day(),
		"hour":      now.Hour(),
		"minute":    now.Minute(),
		"second":    now.Second(),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &tools.Result{Success: true, Content: string(resultJSON)}
}
