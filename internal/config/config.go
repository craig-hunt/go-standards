// Package config reads settings from the environment, the only configuration
// source a twelve-factor process consults.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

var (
	ErrMissing = errors.New(msgMissing)
	ErrInvalid = errors.New(msgInvalid)
)

// LookupFunc matches os.LookupEnv, so tests supply an environment without
// mutating the process's own.
type LookupFunc func(key string) (string, bool)

type Config struct {
	ServiceName     string
	Port            int
	DatabaseURL     string
	APIToken        string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

func Load(lookup LookupFunc) (Config, error) {
	databaseURL, err := Required(lookup, EnvDatabaseURL)
	if err != nil {
		return Config{}, err
	}
	apiToken, err := Required(lookup, EnvAPIToken)
	if err != nil {
		return Config{}, err
	}
	port, err := parsePort(lookup)
	if err != nil {
		return Config{}, err
	}
	level, err := parseLogLevel(lookup)
	if err != nil {
		return Config{}, err
	}
	timeout, err := parseShutdownTimeout(lookup)
	if err != nil {
		return Config{}, err
	}

	serviceName, ok := nonEmpty(lookup, EnvServiceName)
	if !ok {
		serviceName = DefaultServiceName
	}

	return Config{
		ServiceName:     serviceName,
		Port:            port,
		DatabaseURL:     databaseURL,
		APIToken:        apiToken,
		LogLevel:        level,
		ShutdownTimeout: timeout,
	}, nil
}

// Required returns a setting that has no safe default. Secrets and backing
// service URLs fall here, so a missing value stops startup instead of
// connecting somewhere unintended.
func Required(lookup LookupFunc, key string) (string, error) {
	value, ok := nonEmpty(lookup, key)
	if !ok {
		return "", fmt.Errorf(errorFormat, ErrMissing, key)
	}
	return value, nil
}

func nonEmpty(lookup LookupFunc, key string) (string, bool) {
	value, ok := lookup(key)
	return value, ok && value != ""
}

func invalid(key string) error {
	return fmt.Errorf(errorFormat, ErrInvalid, key)
}

func parsePort(lookup LookupFunc) (int, error) {
	raw, ok := nonEmpty(lookup, EnvPort)
	if !ok {
		return DefaultPort, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < MinPort || value > MaxPort {
		return 0, invalid(EnvPort)
	}
	return value, nil
}

func parseLogLevel(lookup LookupFunc) (slog.Level, error) {
	raw, ok := nonEmpty(lookup, EnvLogLevel)
	if !ok {
		return slog.LevelInfo, nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return 0, invalid(EnvLogLevel)
	}
	return level, nil
}

func parseShutdownTimeout(lookup LookupFunc) (time.Duration, error) {
	raw, ok := nonEmpty(lookup, EnvShutdownTimeout)
	if !ok {
		return DefaultShutdownTimeout, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, invalid(EnvShutdownTimeout)
	}
	return value, nil
}
