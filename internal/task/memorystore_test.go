package task_test

import (
	"context"
	"slices"
	"testing"

	"github.com/craig-hunt/go-standards/internal/task"
	"github.com/craig-hunt/go-standards/internal/task/storetest"
)

// memoryStore stands in for Postgres in handler tests. It runs the same
// contract as the real store, so a handler test cannot pass against behavior
// the database would not reproduce.
type memoryStore struct {
	tasks  []task.Task
	nextID task.ID
	err    error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{tasks: []task.Task{}, nextID: 1}
}

func (s *memoryStore) List(context.Context) ([]task.Task, error) {
	if s.err != nil {
		return nil, s.err
	}
	return slices.Clone(s.tasks), nil
}

func (s *memoryStore) Create(_ context.Context, title task.Title) (task.Task, error) {
	if s.err != nil {
		return task.Task{}, s.err
	}
	created := task.Task{ID: s.nextID, Title: title}
	s.nextID++
	s.tasks = append(s.tasks, created)
	return created, nil
}

func (s *memoryStore) SetCompleted(_ context.Context, id task.ID, completed bool) (task.Task, error) {
	if s.err != nil {
		return task.Task{}, s.err
	}
	index := s.indexOf(id)
	if index < 0 {
		return task.Task{}, task.ErrNotFound
	}
	s.tasks[index].Completed = completed
	return s.tasks[index], nil
}

func (s *memoryStore) Delete(_ context.Context, id task.ID) error {
	if s.err != nil {
		return s.err
	}
	index := s.indexOf(id)
	if index < 0 {
		return task.ErrNotFound
	}
	s.tasks = slices.Delete(s.tasks, index, index+1)
	return nil
}

func (s *memoryStore) DeleteCompleted(context.Context) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	before := len(s.tasks)
	s.tasks = slices.DeleteFunc(s.tasks, func(t task.Task) bool { return t.Completed })
	return int64(before - len(s.tasks)), nil
}

func (s *memoryStore) indexOf(id task.ID) int {
	return slices.IndexFunc(s.tasks, func(t task.Task) bool { return t.ID == id })
}

func TestMemoryStoreHonorsTheStoreContract(t *testing.T) {
	storetest.Run(t, func(*testing.T) task.Store { return newMemoryStore() })
}
