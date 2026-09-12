// Package task manages the to-do list the sibling standards projects drive in a
// browser: titles, completion, and filtered views with counts.
package task

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	ErrTitleRequired = errors.New(errTitleRequired)
	ErrTitleTooLong  = errors.New(errTitleTooLong)
	ErrInvalidFilter = errors.New(MsgInvalidFilter)
	ErrInvalidID     = errors.New(MsgInvalidID)
	ErrNotFound      = errors.New(MsgNotFound)
)

type ID int64

// Title exists only through NewTitle, so every Title a store receives has
// already been trimmed and length-checked.
type Title string

type Filter string

type Task struct {
	ID        ID    `json:"id"`
	Title     Title `json:"title"`
	Completed bool  `json:"completed"`
}

type View struct {
	Tasks     []Task `json:"tasks"`
	Remaining int    `json:"remaining"`
	Total     int    `json:"total"`
}

// Store names only what this package calls. A persistence package satisfies it
// without this package importing that one.
type Store interface {
	List(ctx context.Context) ([]Task, error)
	Create(ctx context.Context, title Title) (Task, error)
	SetCompleted(ctx context.Context, id ID, completed bool) (Task, error)
	Delete(ctx context.Context, id ID) error
	DeleteCompleted(ctx context.Context) (int64, error)
}

func NewTitle(raw string) (Title, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrTitleRequired
	}
	if utf8.RuneCountInString(trimmed) > MaxTitleLength {
		return "", ErrTitleTooLong
	}
	return Title(trimmed), nil
}

func ParseFilter(raw string) (Filter, error) {
	switch Filter(raw) {
	case "", FilterAll:
		return FilterAll, nil
	case FilterActive, FilterCompleted:
		return Filter(raw), nil
	default:
		return "", ErrInvalidFilter
	}
}

func ParseID(raw string) (ID, error) {
	value, err := strconv.ParseInt(raw, decimalBase, idBits)
	if err != nil || value < firstValidID {
		return 0, ErrInvalidID
	}
	return ID(value), nil
}

func (f Filter) Includes(t Task) bool {
	switch f {
	case FilterActive:
		return !t.Completed
	case FilterCompleted:
		return t.Completed
	default:
		return true
	}
}

func Summarize(all []Task, filter Filter) View {
	shown := make([]Task, 0, len(all))
	remaining := 0
	for _, t := range all {
		if !t.Completed {
			remaining++
		}
		if filter.Includes(t) {
			shown = append(shown, t)
		}
	}
	return View{Tasks: shown, Remaining: remaining, Total: len(all)}
}
