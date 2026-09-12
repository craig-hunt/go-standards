//go:build integration

package postgres_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/inventory"
	"github.com/craig-hunt/go-standards/internal/postgres"
	"github.com/craig-hunt/go-standards/internal/signup"
	"github.com/craig-hunt/go-standards/internal/task"
	"github.com/craig-hunt/go-standards/internal/task/storetest"
)

const (
	envTestDatabaseURL   = "TEST_DATABASE_URL"
	missingURLFormat     = "%s must name a throwaway database; run make integration"
	schemaPrefix         = "test_"
	searchPathParameter  = "search_path"
	sqlCreateSchema      = "CREATE SCHEMA %s"
	sqlDropSchema        = "DROP SCHEMA IF EXISTS %s CASCADE"
	sqlCountMigrations   = `SELECT count(*) FROM schema_migrations`
	sqlCountTasks        = `SELECT count(*) FROM tasks`
	sqlReadSignup        = `SELECT full_name, email, plan, seats, notes FROM signups WHERE id = $1`
	migrationFileCount   = 1
	existingTitle        = task.Title("Keep the existing task")
	signupName           = "Dana Whitfield"
	signupEmail          = "dana.whitfield@example.com"
	signupSeats          = 12
	signupNotes          = "Migrating next quarter."
	laterSignupReason    = "a later signup receives a larger id"
	positiveSignupReason = "a saved signup receives a positive id"
	rejectedSignupReason = "the database rejects a signup with no seats"
	nameOrderReason      = "inventory items arrive sorted by name"
)

// isolatedPool gives each test its own schema inside the throwaway database, so
// tests and parallel mutation workers never see each other's rows.
func isolatedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url, ok := os.LookupEnv(envTestDatabaseURL)
	if !ok || url == "" {
		t.Fatalf(missingURLFormat, envTestDatabaseURL)
	}
	ctx := context.Background()
	name := schemaPrefix + strings.ToLower(rand.Text())
	identifier := pgx.Identifier{name}.Sanitize()

	admin, err := pgx.Connect(ctx, url)
	expect.NoError(t, err)
	_, err = admin.Exec(ctx, fmt.Sprintf(sqlCreateSchema, identifier))
	expect.NoError(t, err)
	expect.NoError(t, admin.Close(ctx))

	config, err := pgxpool.ParseConfig(url)
	expect.NoError(t, err)
	config.ConnConfig.RuntimeParams[searchPathParameter] = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	expect.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		cleanup, err := pgx.Connect(ctx, url)
		if err != nil {
			return
		}
		_, _ = cleanup.Exec(ctx, fmt.Sprintf(sqlDropSchema, identifier))
		_ = cleanup.Close(ctx)
	})

	expect.NoError(t, postgres.Migrate(ctx, pool))
	return pool
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string) int {
	t.Helper()
	var count int
	expect.NoError(t, pool.QueryRow(context.Background(), query).Scan(&count))
	return count
}

func TestMigrateRecordsEachMigrationAndRunsAgainSafely(t *testing.T) {
	pool := isolatedPool(t)

	expect.NoError(t, postgres.Migrate(context.Background(), pool))

	expect.Equal(t, countRows(t, pool, sqlCountMigrations), migrationFileCount)
}

func TestSeedLoadsTheDemoDataAndRunsAgainSafely(t *testing.T) {
	pool := isolatedPool(t)
	ctx := context.Background()

	expect.NoError(t, postgres.Seed(ctx, pool))
	expect.NoError(t, postgres.Seed(ctx, pool))

	tasks, err := postgres.NewTaskStore(pool).List(ctx)
	expect.NoError(t, err)
	titles := make([]task.Title, 0, len(tasks))
	for _, seeded := range tasks {
		titles = append(titles, seeded.Title)
	}
	expect.Equal(t, titles, task.SeedTitles)

	items, err := postgres.NewInventoryStore(pool).Items(ctx)
	expect.NoError(t, err)
	expect.Equal(t, items, inventory.SeedItems)
}

func TestSeedLeavesATableThatAlreadyHoldsTasksAlone(t *testing.T) {
	pool := isolatedPool(t)
	ctx := context.Background()
	_, err := postgres.NewTaskStore(pool).Create(ctx, existingTitle)
	expect.NoError(t, err)

	expect.NoError(t, postgres.Seed(ctx, pool))

	expect.Equal(t, countRows(t, pool, sqlCountTasks), 1)
}

func TestTaskStoreHonorsTheStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) task.Store {
		return postgres.NewTaskStore(isolatedPool(t))
	})
}

func TestSignupStoreSavesEachSignupUnderANewID(t *testing.T) {
	pool := isolatedPool(t)
	ctx := context.Background()
	store := postgres.NewSignupStore(pool)
	record := signup.Signup{FullName: signupName, Email: signupEmail, Plan: signup.PlanGrowth, Seats: signupSeats, Notes: signupNotes}

	first, err := store.Save(ctx, record)
	expect.NoError(t, err)
	second, err := store.Save(ctx, record)
	expect.NoError(t, err)

	expect.True(t, first >= 1, positiveSignupReason)
	expect.True(t, second > first, laterSignupReason)
	var stored signup.Signup
	expect.NoError(t, pool.QueryRow(ctx, sqlReadSignup, first).Scan(&stored.FullName, &stored.Email, &stored.Plan, &stored.Seats, &stored.Notes))
	expect.Equal(t, stored, record)
}

func TestSignupStoreReportsARowTheDatabaseRejects(t *testing.T) {
	pool := isolatedPool(t)

	_, err := postgres.NewSignupStore(pool).Save(context.Background(), signup.Signup{FullName: signupName, Email: signupEmail, Plan: signup.PlanGrowth})

	expect.True(t, err != nil, rejectedSignupReason)
}

func TestInventoryStoreReturnsItemsInNameOrder(t *testing.T) {
	pool := isolatedPool(t)
	ctx := context.Background()
	expect.NoError(t, postgres.Seed(ctx, pool))

	items, err := postgres.NewInventoryStore(pool).Items(ctx)

	expect.NoError(t, err)
	expect.True(t, slices.IsSortedFunc(items, func(a, b inventory.Item) int { return strings.Compare(a.Name, b.Name) }), nameOrderReason)
}
