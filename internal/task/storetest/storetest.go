// Package storetest holds the behavior every task.Store must share. The
// in-memory fake used by handler tests and the Postgres store both run it, so
// the fake cannot drift from the real thing.
package storetest

import (
	"context"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/task"
)

const (
	firstTitle     task.Title = "Draft the incident postmortem"
	secondTitle    task.Title = "Schedule the security review"
	missingID      task.ID    = 999999
	increaseReason            = "a later task receives a larger id"
	positiveReason            = "a created task receives a positive id"
)

// Open returns a store holding no tasks.
type Open func(t *testing.T) task.Store

func Run(t *testing.T, open Open) {
	t.Run("lists tasks in the order they were created", func(t *testing.T) {
		store := open(t)
		ctx := context.Background()
		first, err := store.Create(ctx, firstTitle)
		expect.NoError(t, err)
		second, err := store.Create(ctx, secondTitle)
		expect.NoError(t, err)

		all, err := store.List(ctx)

		expect.NoError(t, err)
		expect.Equal(t, all, []task.Task{first, second})
		expect.True(t, first.ID >= 1, positiveReason)
		expect.True(t, second.ID > first.ID, increaseReason)
	})

	t.Run("creates tasks incomplete with the given title", func(t *testing.T) {
		created, err := open(t).Create(context.Background(), firstTitle)

		expect.NoError(t, err)
		expect.Equal(t, created.Title, firstTitle)
		expect.Equal(t, created.Completed, false)
	})

	t.Run("marks a task completed and back again", func(t *testing.T) {
		store := open(t)
		ctx := context.Background()
		created, err := store.Create(ctx, firstTitle)
		expect.NoError(t, err)

		completed, err := store.SetCompleted(ctx, created.ID, true)
		expect.NoError(t, err)
		reopened, err := store.SetCompleted(ctx, created.ID, false)
		expect.NoError(t, err)

		expect.Equal(t, completed, task.Task{ID: created.ID, Title: firstTitle, Completed: true})
		expect.Equal(t, reopened, created)
	})

	t.Run("reports a missing task when completing an unknown id", func(t *testing.T) {
		_, err := open(t).SetCompleted(context.Background(), missingID, true)

		expect.ErrorIs(t, err, task.ErrNotFound)
	})

	t.Run("deletes one task and leaves the rest", func(t *testing.T) {
		store := open(t)
		ctx := context.Background()
		first, err := store.Create(ctx, firstTitle)
		expect.NoError(t, err)
		second, err := store.Create(ctx, secondTitle)
		expect.NoError(t, err)

		expect.NoError(t, store.Delete(ctx, first.ID))

		all, err := store.List(ctx)
		expect.NoError(t, err)
		expect.Equal(t, all, []task.Task{second})
	})

	t.Run("reports a missing task when deleting an unknown id", func(t *testing.T) {
		expect.ErrorIs(t, open(t).Delete(context.Background(), missingID), task.ErrNotFound)
	})

	t.Run("deletes only completed tasks and counts them", func(t *testing.T) {
		store := open(t)
		ctx := context.Background()
		done, err := store.Create(ctx, firstTitle)
		expect.NoError(t, err)
		pending, err := store.Create(ctx, secondTitle)
		expect.NoError(t, err)
		_, err = store.SetCompleted(ctx, done.ID, true)
		expect.NoError(t, err)

		removed, err := store.DeleteCompleted(ctx)

		expect.NoError(t, err)
		expect.Equal(t, removed, int64(1))
		all, err := store.List(ctx)
		expect.NoError(t, err)
		expect.Equal(t, all, []task.Task{pending})
	})

	t.Run("returns an empty list rather than nil when no tasks exist", func(t *testing.T) {
		all, err := open(t).List(context.Background())

		expect.NoError(t, err)
		expect.Equal(t, all, []task.Task{})
	})
}
