package server

import "time"

const (
	Network           = "tcp"
	ReadHeaderTimeout = 5 * time.Second
	ReadTimeout       = 15 * time.Second
	WriteTimeout      = 15 * time.Second
	IdleTimeout       = 60 * time.Second
	LogKeyAddress     = "address"
	MsgListening      = "listening"
	MsgShuttingDown   = "shutting down"
)

const (
	addressFormat       = ":%d"
	shutdownErrorFormat = "graceful shutdown: %w"
)
