// Command demoapi serves the reference JSON API. It holds wiring only; every
// behavior it assembles lives, and is tested, in an internal package.
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/craig-hunt/go-standards/internal/api"
	"github.com/craig-hunt/go-standards/internal/config"
	"github.com/craig-hunt/go-standards/internal/logging"
	"github.com/craig-hunt/go-standards/internal/postgres"
	"github.com/craig-hunt/go-standards/internal/server"
)

const exitFailure = 1

func main() {
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
}

func run(ctx context.Context, lookup config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.Load(lookup)
	if err != nil {
		return err
	}
	logger := logging.New(stdout, cfg.ServiceName, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	handler := api.NewHandler(api.Dependencies{
		Tasks:     postgres.NewTaskStore(pool),
		Signups:   postgres.NewSignupStore(pool),
		Inventory: postgres.NewInventoryStore(pool),
		Pinger:    pool,
		APIToken:  cfg.APIToken,
		Logger:    logger,
		Now:       time.Now,
	})

	listener, err := (&net.ListenConfig{}).Listen(ctx, server.Network, server.Address(cfg.Port))
	if err != nil {
		return err
	}
	return server.Run(ctx, server.New(handler), listener, cfg.ShutdownTimeout, logger)
}
