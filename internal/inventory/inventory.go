// Package inventory serves the stock table: a case-insensitive name search and
// a sort on any column, with counts of what matched.
package inventory

import (
	"cmp"
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
)

var (
	ErrInvalidSort      = errors.New(MsgInvalidSort)
	ErrInvalidDirection = errors.New(MsgInvalidDirection)
)

type Column string

type Direction string

type Status string

type Item struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Status   Status `json:"status"`
}

type Query struct {
	Search    string
	Sort      Column
	Direction Direction
}

type Result struct {
	Items []Item `json:"items"`
	Shown int    `json:"shown"`
	Total int    `json:"total"`
}

type Store interface {
	Items(ctx context.Context) ([]Item, error)
}

func (c Column) Known() bool {
	switch c {
	case ColumnName, ColumnQuantity, ColumnStatus:
		return true
	default:
		return false
	}
}

func (d Direction) Known() bool {
	return d == DirectionAscending || d == DirectionDescending
}

func ParseQuery(values url.Values) (Query, error) {
	query := Query{
		Search:    strings.TrimSpace(values.Get(QuerySearch)),
		Sort:      ColumnName,
		Direction: DirectionAscending,
	}
	if raw := values.Get(QuerySort); raw != "" {
		query.Sort = Column(raw)
		if !query.Sort.Known() {
			return Query{}, ErrInvalidSort
		}
	}
	if raw := values.Get(QueryDirection); raw != "" {
		query.Direction = Direction(raw)
		if !query.Direction.Known() {
			return Query{}, ErrInvalidDirection
		}
	}
	return query, nil
}

// Apply leaves the caller's slice untouched and sorts stably, so rows that tie
// on the sort column keep the order the store returned.
func Apply(items []Item, query Query) Result {
	term := strings.ToLower(query.Search)
	shown := make([]Item, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), term) {
			shown = append(shown, item)
		}
	}
	slices.SortStableFunc(shown, func(a, b Item) int {
		order := compare(a, b, query.Sort)
		if query.Direction == DirectionDescending {
			return -order
		}
		return order
	})
	return Result{Items: shown, Shown: len(shown), Total: len(items)}
}

func compare(a, b Item, column Column) int {
	switch column {
	case ColumnQuantity:
		return cmp.Compare(a.Quantity, b.Quantity)
	case ColumnStatus:
		return cmp.Compare(a.Status, b.Status)
	default:
		return cmp.Compare(a.Name, b.Name)
	}
}
