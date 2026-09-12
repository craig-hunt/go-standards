// Package postgres satisfies the domain packages' Store interfaces with
// PostgreSQL through pgx. The domain packages define those interfaces and never
// import this one.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craig-hunt/go-standards/internal/inventory"
	"github.com/craig-hunt/go-standards/internal/task"
)

//go:embed migrations/*.sql
var migrations embed.FS

func wrap(operation string, err error) error {
	return fmt.Errorf(errorFormat, operation, err)
}

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, wrap(opOpen, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, wrap(opPing, err)
	}
	return pool, nil
}

// Migrate applies each embedded migration once, in file-name order. Each file
// and its tracking row commit together, so a failed migration leaves no record
// claiming it ran.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, sqlCreateMigrationsTable); err != nil {
		return wrap(opPrepareMigrate, err)
	}
	names, err := fs.Glob(migrations, migrationsPattern)
	if err != nil {
		return wrap(opListMigrations, err)
	}
	for _, name := range names {
		if err := applyMigration(ctx, pool, name); err != nil {
			return wrap(opApplyMigration, err)
		}
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, name string) error {
	script, err := migrations.ReadFile(name)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var applied bool
		if err := tx.QueryRow(ctx, sqlMigrationApplied, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			return nil
		}
		if _, err := tx.Exec(ctx, string(script)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, sqlRecordMigration, name)
		return err
	})
}

// Seed loads the demo data an admin process installs. Tasks load only into an
// empty table and inventory rows upsert, so running it twice changes nothing.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	titles := make([]string, 0, len(task.SeedTitles))
	for _, title := range task.SeedTitles {
		titles = append(titles, string(title))
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, sqlSeedTasks, titles); err != nil {
			return err
		}
		for _, item := range inventory.SeedItems {
			if _, err := tx.Exec(ctx, sqlUpsertItem, item.Name, item.Quantity, item.Status); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return wrap(opSeed, err)
	}
	return nil
}
