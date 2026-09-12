// Package logging builds the structured logger every process in the module
// writes through: JSON to one stream, UTC timestamps, the service name on
// every line.
package logging

import (
	"io"
	"log/slog"
)

const KeyService = "service"

func New(w io.Writer, service string, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level, ReplaceAttr: utcTimestamp})
	return slog.New(handler).With(slog.String(KeyService, service))
}

// utcTimestamp converts only the record's own timestamp. Time values a caller
// attaches keep their zone, since the caller chose it deliberately.
func utcTimestamp(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey && attr.Value.Kind() == slog.KindTime {
		return slog.Time(slog.TimeKey, attr.Value.Time().UTC())
	}
	return attr
}
