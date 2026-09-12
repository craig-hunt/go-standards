package middleware

import "time"

const (
	targetPath       = "/api/tasks"
	suppliedID       = "trace-7f3a"
	validToken       = "correct-token"
	wrongToken       = "wrong-token"
	lowercaseScheme  = "bearer "
	panicMessage     = "handler exploded"
	partialBody      = "partial response"
	stepDuration     = 250 * time.Millisecond
	stepMilliseconds = 250
	outerLayer       = "outer"
	innerLayer       = "inner"
	handlerStep      = "handler"
	generatedReason  = "a generated identifier fills the response header"
	contextReason    = "the handler sees the identifier in its context"
	notCalledReason  = "the protected handler never runs"
)

var startInstant = time.Date(2026, time.September, 12, 9, 0, 0, 0, time.UTC)
