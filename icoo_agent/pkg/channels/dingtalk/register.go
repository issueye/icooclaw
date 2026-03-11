package dingtalk

import (
	"encoding/json"
	"log/slog"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
	"icooclaw/pkg/channels/consts"
)

func init() {
	channels.RegisterFactory(consts.DINGTALK, func(config map[string]any, bus *bus.MessageBus, logger *slog.Logger) (channels.Channel, error) {
		cfg, err := parseConfig(config)
		if err != nil {
			return nil, err
		}

		return New(cfg, bus, logger)
	})
}

// parseConfig parses the configuration map into Config struct.
func parseConfig(config map[string]any) (Config, error) {
	cfg := Config{}

	if v, ok := config["enabled"]; ok {
		if b, ok := v.(bool); ok {
			cfg.Enabled = b
		}
	}

	if v, ok := config["client_id"]; ok {
		if s, ok := v.(string); ok {
			cfg.ClientID = s
		}
	}

	if v, ok := config["client_secret"]; ok {
		if s, ok := v.(string); ok {
			cfg.ClientSecret = s
		}
	}

	if v, ok := config["agent_id"]; ok {
		switch val := v.(type) {
		case int64:
			cfg.AgentID = val
		case int:
			cfg.AgentID = int64(val)
		case float64:
			cfg.AgentID = int64(val)
		}
	}

	if v, ok := config["allow_from"]; ok {
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if s, ok := item.(string); ok {
					cfg.AllowFrom = append(cfg.AllowFrom, s)
				}
			}
		}
	}

	if v, ok := config["reasoning_chat_id"]; ok {
		if s, ok := v.(string); ok {
			cfg.ReasoningChatID = s
		}
	}

	return cfg, nil
}

// ParseConfigFromJSON parses configuration from JSON bytes.
func ParseConfigFromJSON(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
