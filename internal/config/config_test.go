package config

import (
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/craig-hunt/go-standards/internal/expect"
)

func environment(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func secretsOnly() map[string]string {
	return map[string]string{EnvDatabaseURL: testDatabaseURL, EnvAPIToken: testAPIToken}
}

func with(key, value string) map[string]string {
	values := secretsOnly()
	values[key] = value
	return values
}

func TestLoadAppliesDefaultsWhenOnlyRequiredSettingsExist(t *testing.T) {
	cfg, err := Load(environment(secretsOnly()))

	expect.NoError(t, err)
	expect.Equal(t, cfg, Config{
		ServiceName:     DefaultServiceName,
		Port:            DefaultPort,
		DatabaseURL:     testDatabaseURL,
		APIToken:        testAPIToken,
		LogLevel:        slog.LevelInfo,
		ShutdownTimeout: DefaultShutdownTimeout,
	})
}

func TestLoadReadsEveryOptionalSetting(t *testing.T) {
	values := secretsOnly()
	values[EnvServiceName] = testServiceName
	values[EnvPort] = testPort
	values[EnvLogLevel] = testLogLevel
	values[EnvShutdownTimeout] = testTimeout

	cfg, err := Load(environment(values))

	expect.NoError(t, err)
	expect.Equal(t, cfg, Config{
		ServiceName:     testServiceName,
		Port:            testPortValue,
		DatabaseURL:     testDatabaseURL,
		APIToken:        testAPIToken,
		LogLevel:        slog.LevelDebug,
		ShutdownTimeout: testTimeoutValue,
	})
}

func TestLoadTreatsEmptyOptionalSettingsAsUnset(t *testing.T) {
	values := secretsOnly()
	for _, key := range []string{EnvServiceName, EnvPort, EnvLogLevel, EnvShutdownTimeout} {
		values[key] = ""
	}

	cfg, err := Load(environment(values))

	expect.NoError(t, err)
	expect.Equal(t, cfg.ServiceName, DefaultServiceName)
	expect.Equal(t, cfg.Port, DefaultPort)
	expect.Equal(t, cfg.LogLevel, slog.LevelInfo)
	expect.Equal(t, cfg.ShutdownTimeout, DefaultShutdownTimeout)
}

func TestLoadRefusesToStartWithoutARequiredSetting(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]string
		key    string
	}{
		{name: "missing database URL", values: map[string]string{EnvAPIToken: testAPIToken}, key: EnvDatabaseURL},
		{name: "empty database URL", values: with(EnvDatabaseURL, ""), key: EnvDatabaseURL},
		{name: "missing API token", values: map[string]string{EnvDatabaseURL: testDatabaseURL}, key: EnvAPIToken},
		{name: "empty API token", values: with(EnvAPIToken, ""), key: EnvAPIToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(environment(tc.values))

			expect.ErrorIs(t, err, ErrMissing)
			expect.True(t, strings.Contains(err.Error(), tc.key), missingKeyMessage)
		})
	}
}

func TestLoadRejectsInvalidOptionalSettings(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "port that is not a number", key: EnvPort, value: portNotANumber},
		{name: "port below the valid range", key: EnvPort, value: strconv.Itoa(MinPort - 1)},
		{name: "port above the valid range", key: EnvPort, value: strconv.Itoa(MaxPort + 1)},
		{name: "unknown log level", key: EnvLogLevel, value: unknownLogLevel},
		{name: "unparsable shutdown timeout", key: EnvShutdownTimeout, value: unparsableDuration},
		{name: "zero shutdown timeout", key: EnvShutdownTimeout, value: zeroDuration},
		{name: "negative shutdown timeout", key: EnvShutdownTimeout, value: negativeDuration},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(environment(with(tc.key, tc.value)))

			expect.ErrorIs(t, err, ErrInvalid)
			expect.True(t, strings.Contains(err.Error(), tc.key), invalidKeyMessage)
		})
	}
}

func TestLoadAcceptsTheBoundaryValues(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "lowest valid port", key: EnvPort, value: strconv.Itoa(MinPort)},
		{name: "highest valid port", key: EnvPort, value: strconv.Itoa(MaxPort)},
		{name: "smallest positive shutdown timeout", key: EnvShutdownTimeout, value: smallestPositiveDur},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(environment(with(tc.key, tc.value)))

			expect.NoError(t, err)
		})
	}
}

func TestLoadKeepsTheBoundaryValuesItAccepts(t *testing.T) {
	lowest, err := Load(environment(with(EnvPort, strconv.Itoa(MinPort))))
	expect.NoError(t, err)
	expect.Equal(t, lowest.Port, MinPort)

	highest, err := Load(environment(with(EnvPort, strconv.Itoa(MaxPort))))
	expect.NoError(t, err)
	expect.Equal(t, highest.Port, MaxPort)

	shortest, err := Load(environment(with(EnvShutdownTimeout, smallestPositiveDur)))
	expect.NoError(t, err)
	expect.Equal(t, shortest.ShutdownTimeout, time.Nanosecond)
}

func TestRequiredReturnsThePresentValue(t *testing.T) {
	value, err := Required(environment(secretsOnly()), EnvAPIToken)

	expect.NoError(t, err)
	expect.Equal(t, value, testAPIToken)
}
