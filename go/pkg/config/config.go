package config

import "time"

// Server configuration constants
const (
	DefaultGRPCPort      = ":50051"
	DefaultToolTimeout   = 30 * time.Second
	DefaultHTTPTimeout   = 30 * time.Second
	MaxMessageSize       = 32 * 1024 * 1024 // 32MB
	DefaultBSRBaseURL    = "https://api.buf.build"
	SupportedProtocolVersion = "1.0.0"
)

// Client configuration constants
const (
	DefaultGRPCTarget = "localhost:50051"
)
