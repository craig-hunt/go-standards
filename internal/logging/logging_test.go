package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/craig-hunt/go-standards/internal/expect"
)

func writeRecord(t *testing.T, attrs ...slog.Attr) map[string]any {
	t.Helper()
	var buffer bytes.Buffer
	logger := New(&buffer, testService, slog.LevelInfo)
	record := slog.NewRecord(sampleInstant, slog.LevelInfo, testMessage, 0)
	record.AddAttrs(attrs...)
	expect.NoError(t, logger.Handler().Handle(context.Background(), record))

	var fields map[string]any
	expect.NoError(t, json.Unmarshal(buffer.Bytes(), &fields))
	return fields
}

func parseTime(t *testing.T, value any) time.Time {
	t.Helper()
	text, _ := value.(string)
	parsed, err := time.Parse(time.RFC3339Nano, text)
	expect.NoError(t, err)
	return parsed
}

func TestNewWritesTheServiceLevelAndMessageAsJSON(t *testing.T) {
	fields := writeRecord(t)

	expect.Equal(t, fields[KeyService], any(testService))
	expect.Equal(t, fields[slog.LevelKey], any(slog.LevelInfo.String()))
	expect.Equal(t, fields[slog.MessageKey], any(testMessage))
}

func TestNewWritesTheRecordTimestampInUTC(t *testing.T) {
	stamp := parseTime(t, writeRecord(t)[slog.TimeKey])

	expect.Equal(t, stamp.Location(), time.UTC)
	expect.True(t, stamp.Equal(sampleInstant), testMessage)
}

func TestNewLeavesAttachedTimesInTheirOwnZone(t *testing.T) {
	stamp := parseTime(t, writeRecord(t, slog.Time(attachedTimeKey, sampleInstant))[attachedTimeKey])

	_, offset := stamp.Zone()
	expect.Equal(t, offset, zoneOffset)
}

func TestNewLeavesATimeKeyInsideAGroupInItsOwnZone(t *testing.T) {
	fields := writeRecord(t, slog.Group(attachedGroup, slog.Time(slog.TimeKey, sampleInstant)))

	group, _ := fields[attachedGroup].(map[string]any)
	_, offset := parseTime(t, group[slog.TimeKey]).Zone()
	expect.Equal(t, offset, zoneOffset)
}

func TestNewSuppressesRecordsBelowTheConfiguredLevel(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(&buffer, testService, slog.LevelInfo)

	logger.Debug(testMessage)

	expect.Equal(t, buffer.Len(), 0)
	expect.True(t, !logger.Enabled(context.Background(), slog.LevelDebug), suppressedReason)
}
