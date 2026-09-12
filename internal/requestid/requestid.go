// Package requestid carries one identifier per request through its context, so
// every log line the request produces joins back to it.
package requestid

import (
	"context"
	"crypto/rand"
	"log/slog"
)

type contextKey struct{}

func New() string {
	return rand.Text()
}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

func From(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// Attr names the log field once, so every correlated log line spells it the
// same way.
func Attr(ctx context.Context) slog.Attr {
	return slog.String(LogKey, From(ctx))
}

// Accept keeps a caller-supplied identifier only when it fits, so a client
// cannot bloat every log line with an oversized header.
func Accept(candidate string) string {
	if candidate == "" || len(candidate) > MaxLength {
		return New()
	}
	return candidate
}
