package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/craig-hunt/go-standards/internal/inventory"
	"github.com/craig-hunt/go-standards/internal/signup"
	"github.com/craig-hunt/go-standards/internal/task"
)

type TaskStore struct {
	pool *pgxpool.Pool
}

func NewTaskStore(pool *pgxpool.Pool) *TaskStore {
	return &TaskStore{pool: pool}
}

func (s *TaskStore) List(ctx context.Context) ([]task.Task, error) {
	rows, err := s.pool.Query(ctx, sqlListTasks)
	if err != nil {
		return nil, wrap(opListTasks, err)
	}
	tasks, err := pgx.CollectRows(rows, pgx.RowToStructByPos[task.Task])
	if err != nil {
		return nil, wrap(opListTasks, err)
	}
	return tasks, nil
}

func (s *TaskStore) Create(ctx context.Context, title task.Title) (task.Task, error) {
	created, err := scanTask(s.pool.QueryRow(ctx, sqlCreateTask, title))
	if err != nil {
		return task.Task{}, wrap(opCreateTask, err)
	}
	return created, nil
}

func (s *TaskStore) SetCompleted(ctx context.Context, id task.ID, completed bool) (task.Task, error) {
	updated, err := scanTask(s.pool.QueryRow(ctx, sqlSetCompleted, id, completed))
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	if err != nil {
		return task.Task{}, wrap(opSetCompleted, err)
	}
	return updated, nil
}

func (s *TaskStore) Delete(ctx context.Context, id task.ID) error {
	tag, err := s.pool.Exec(ctx, sqlDeleteTask, id)
	if err != nil {
		return wrap(opDeleteTask, err)
	}
	if tag.RowsAffected() == 0 {
		return task.ErrNotFound
	}
	return nil
}

func (s *TaskStore) DeleteCompleted(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, sqlDeleteCompleted)
	if err != nil {
		return 0, wrap(opDeleteCompleted, err)
	}
	return tag.RowsAffected(), nil
}

func scanTask(row pgx.Row) (task.Task, error) {
	var scanned task.Task
	err := row.Scan(&scanned.ID, &scanned.Title, &scanned.Completed)
	return scanned, err
}

type SignupStore struct {
	pool *pgxpool.Pool
}

func NewSignupStore(pool *pgxpool.Pool) *SignupStore {
	return &SignupStore{pool: pool}
}

func (s *SignupStore) Save(ctx context.Context, record signup.Signup) (signup.ID, error) {
	var id signup.ID
	err := s.pool.QueryRow(ctx, sqlSaveSignup, record.FullName, record.Email, record.Plan, record.Seats, record.Notes).Scan(&id)
	if err != nil {
		return 0, wrap(opSaveSignup, err)
	}
	return id, nil
}

type InventoryStore struct {
	pool *pgxpool.Pool
}

func NewInventoryStore(pool *pgxpool.Pool) *InventoryStore {
	return &InventoryStore{pool: pool}
}

func (s *InventoryStore) Items(ctx context.Context) ([]inventory.Item, error) {
	rows, err := s.pool.Query(ctx, sqlListItems)
	if err != nil {
		return nil, wrap(opListItems, err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[inventory.Item])
	if err != nil {
		return nil, wrap(opListItems, err)
	}
	return items, nil
}
