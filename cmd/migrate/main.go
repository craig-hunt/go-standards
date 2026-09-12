// Command migrate is the service's admin process: it applies schema migrations
// and loads demo data as a one-off run, separate from the long-running server.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/craig-hunt/go-standards/internal/config"
	"github.com/craig-hunt/go-standards/internal/logging"
	"github.com/craig-hunt/go-standards/internal/postgres"
)

const (
	exitFailure = 1
	serviceName = "go-standards-migrate"
	msgMigrated = "migrations applied"
	msgSeeded   = "demo data loaded"
)

func main() {
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
}

func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	databaseURL, err := config.Required(lookup, config.EnvDatabaseURL)
	if err != nil {
		return err
	}
	logger := logging.New(stdout, serviceName, slog.LevelInfo)

	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		return err
	}
	logger.InfoContext(ctx, msgMigrated)

	if err := postgres.Seed(ctx, pool); err != nil {
		return err
	}
	logger.InfoContext(ctx, msgSeeded)
	return nil
}
