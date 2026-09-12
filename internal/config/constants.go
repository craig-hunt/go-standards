package config

import "time"

const (
	EnvServiceName     = "SERVICE_NAME"
	EnvPort            = "PORT"
	EnvDatabaseURL     = "DATABASE_URL"
	EnvAPIToken        = "API_TOKEN"
	EnvLogLevel        = "LOG_LEVEL"
	EnvShutdownTimeout = "SHUTDOWN_TIMEOUT"
)

const (
	DefaultServiceName     = "go-standards-demo"
	DefaultPort            = 8080
	DefaultShutdownTimeout = 10 * time.Second
	MinPort                = 1
	MaxPort                = 65535
)

const (
	msgMissing  = "required environment variable is not set"
	msgInvalid  = "environment variable holds an invalid value"
	errorFormat = "%w: %s"
)
