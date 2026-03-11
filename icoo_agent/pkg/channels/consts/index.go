package consts

import "time"

const (
	DINGTALK  = "dingtalk"
	FEISHU    = "feishu"
	TELEGRAM  = "telegram"
	DISCORD   = "discord"
	SLACK     = "slack"
	WEB       = "web"
	WEBSOCKET = "websocket"
)

// Default rate limits per channel (messages per second).
var ChannelRateConfig = map[string]float64{
	DINGTALK: 10,
	FEISHU:   10,
	TELEGRAM: 20,
	DISCORD:  1,
	SLACK:    100,
}

const (
	DefaultRateLimit      = 10
	DefaultRetries        = 3
	DefaultRateLimitDelay = 1 * time.Second
	DefaultBaseBackoff    = 500 * time.Millisecond
	DefaultMaxBackoff     = 8 * time.Second
)
