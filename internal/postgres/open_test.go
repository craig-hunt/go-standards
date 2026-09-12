package postgres_test

import (
	"context"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/postgres"
)

const (
	unparsableURL   = "postgres://%zz"
	unreachableURL  = "postgres://demo@127.0.0.1:1/demo?connect_timeout=1&sslmode=disable"
	openErrorReason = "an unusable database URL fails to open"
	pingErrorReason = "an unreachable database fails to open"
)

func TestOpenRejectsAURLItCannotParse(t *testing.T) {
	pool, err := postgres.Open(context.Background(), unparsableURL)

	expect.True(t, err != nil, openErrorReason)
	expect.True(t, pool == nil, openErrorReason)
}

func TestOpenReportsADatabaseItCannotReach(t *testing.T) {
	pool, err := postgres.Open(context.Background(), unreachableURL)

	expect.True(t, err != nil, pingErrorReason)
	expect.True(t, pool == nil, pingErrorReason)
}
