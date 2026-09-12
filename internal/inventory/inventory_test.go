package inventory_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/craig-hunt/go-standards/internal/apitest"
	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/inventory"
	"github.com/craig-hunt/go-standards/internal/logcapture"
)

func named(t *testing.T, names ...string) []inventory.Item {
	t.Helper()
	items := make([]inventory.Item, 0, len(names))
	for _, name := range names {
		index := slices.IndexFunc(inventory.SeedItems, func(item inventory.Item) bool { return item.Name == name })
		expect.True(t, index >= 0, name)
		items = append(items, inventory.SeedItems[index])
	}
	return items
}

func query(values url.Values) inventory.Query {
	parsed, _ := inventory.ParseQuery(values)
	return parsed
}

type fixedStore struct {
	items []inventory.Item
	err   error
}

func (s fixedStore) Items(context.Context) ([]inventory.Item, error) {
	return s.items, s.err
}

func list(t *testing.T, store inventory.Store, values url.Values) (int, []byte) {
	t.Helper()
	logger, _ := logcapture.New()
	mux := http.NewServeMux()
	inventory.NewHandler(store, logger).Register(mux)
	recorder := apitest.Do(t, mux, apitest.Request{Method: http.MethodGet, Target: inventory.PathInventory + querySeparator + values.Encode()})
	return recorder.Code, recorder.Body.Bytes()
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var value T
	expect.NoError(t, json.Unmarshal(raw, &value))
	return value
}

func TestParseQueryDefaultsToNameAscendingWithNoSearch(t *testing.T) {
	parsed, err := inventory.ParseQuery(url.Values{})

	expect.NoError(t, err)
	expect.Equal(t, parsed, inventory.Query{Sort: inventory.ColumnName, Direction: inventory.DirectionAscending})
}

func TestParseQueryReadsEverySetting(t *testing.T) {
	parsed, err := inventory.ParseQuery(url.Values{
		inventory.QuerySearch:    {mixedCaseSearch},
		inventory.QuerySort:      {string(inventory.ColumnQuantity)},
		inventory.QueryDirection: {string(inventory.DirectionDescending)},
	})

	expect.NoError(t, err)
	expect.Equal(t, parsed, inventory.Query{Search: trimmedSearch, Sort: inventory.ColumnQuantity, Direction: inventory.DirectionDescending})
}

func TestParseQueryRejectsAnUnknownSortColumn(t *testing.T) {
	_, err := inventory.ParseQuery(url.Values{inventory.QuerySort: {unknownColumn}})

	expect.ErrorIs(t, err, inventory.ErrInvalidSort)
}

func TestParseQueryRejectsAnUnknownDirection(t *testing.T) {
	_, err := inventory.ParseQuery(url.Values{inventory.QueryDirection: {unknownOrder}})

	expect.ErrorIs(t, err, inventory.ErrInvalidDirection)
}

func TestApplySortsByNameAscendingByDefault(t *testing.T) {
	result := inventory.Apply(inventory.SeedItems, query(url.Values{}))

	expect.Equal(t, result.Items, named(t, accessBadge, dockingStation, laptopSleeve, monitorArm, headset, webcam))
	expect.Equal(t, result.Shown, len(inventory.SeedItems))
	expect.Equal(t, result.Total, len(inventory.SeedItems))
}

func TestApplySortsOnEachColumnInEitherDirection(t *testing.T) {
	cases := []struct {
		name      string
		column    inventory.Column
		direction inventory.Direction
		want      []string
	}{
		{name: "name descending", column: inventory.ColumnName, direction: inventory.DirectionDescending,
			want: []string{webcam, headset, monitorArm, laptopSleeve, dockingStation, accessBadge}},
		{name: "quantity ascending", column: inventory.ColumnQuantity, direction: inventory.DirectionAscending,
			want: []string{laptopSleeve, headset, dockingStation, webcam, monitorArm, accessBadge}},
		{name: "quantity descending", column: inventory.ColumnQuantity, direction: inventory.DirectionDescending,
			want: []string{accessBadge, monitorArm, webcam, dockingStation, headset, laptopSleeve}},
		{name: "status ascending keeps ties in store order", column: inventory.ColumnStatus, direction: inventory.DirectionAscending,
			want: []string{accessBadge, monitorArm, webcam, dockingStation, headset, laptopSleeve}},
		{name: "status descending keeps ties in store order", column: inventory.ColumnStatus, direction: inventory.DirectionDescending,
			want: []string{laptopSleeve, dockingStation, headset, accessBadge, monitorArm, webcam}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := inventory.Apply(inventory.SeedItems, inventory.Query{Sort: tc.column, Direction: tc.direction})

			expect.Equal(t, result.Items, named(t, tc.want...))
		})
	}
}

func TestApplyMatchesNamesWithoutRegardToCase(t *testing.T) {
	result := inventory.Apply(inventory.SeedItems, query(url.Values{inventory.QuerySearch: {mixedCaseSearch}}))

	expect.Equal(t, result.Items, named(t, webcam))
	expect.Equal(t, result.Shown, len(named(t, webcam)))
	expect.Equal(t, result.Total, len(inventory.SeedItems))
}

func TestApplyReturnsAnEmptyListWhenNothingMatches(t *testing.T) {
	result := inventory.Apply(inventory.SeedItems, query(url.Values{inventory.QuerySearch: {noMatchSearch}}))

	expect.Equal(t, result, inventory.Result{Items: []inventory.Item{}, Total: len(inventory.SeedItems)})
}

func TestApplyLeavesTheCallersSliceInItsOriginalOrder(t *testing.T) {
	original := slices.Clone(inventory.SeedItems)

	inventory.Apply(inventory.SeedItems, inventory.Query{Sort: inventory.ColumnQuantity, Direction: inventory.DirectionDescending})

	expect.True(t, slices.Equal(inventory.SeedItems, original), unchangedReason)
}

func TestColumnAndDirectionKnowOnlyTheirOwnValues(t *testing.T) {
	for _, column := range []inventory.Column{inventory.ColumnName, inventory.ColumnQuantity, inventory.ColumnStatus} {
		expect.Equal(t, column.Known(), true)
	}
	expect.Equal(t, inventory.Column(unknownColumn).Known(), false)
	expect.Equal(t, inventory.DirectionAscending.Known(), true)
	expect.Equal(t, inventory.DirectionDescending.Known(), true)
	expect.Equal(t, inventory.Direction(unknownOrder).Known(), false)
}

func TestListReturnsTheFilteredSortedItems(t *testing.T) {
	status, body := list(t, fixedStore{items: inventory.SeedItems}, url.Values{
		inventory.QuerySort:      {string(inventory.ColumnQuantity)},
		inventory.QueryDirection: {string(inventory.DirectionDescending)},
	})

	expect.Equal(t, status, http.StatusOK)
	result := decode[inventory.Result](t, body)
	expect.Equal(t, result.Items, named(t, accessBadge, monitorArm, webcam, dockingStation, headset, laptopSleeve))
}

func TestListRejectsAnInvalidQuery(t *testing.T) {
	cases := []struct {
		name   string
		values url.Values
		want   string
	}{
		{name: "unknown sort column", values: url.Values{inventory.QuerySort: {unknownColumn}}, want: inventory.MsgInvalidSort},
		{name: "unknown direction", values: url.Values{inventory.QueryDirection: {unknownOrder}}, want: inventory.MsgInvalidDirection},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := list(t, fixedStore{items: inventory.SeedItems}, tc.values)

			expect.Equal(t, status, http.StatusBadRequest)
			expect.Equal(t, decode[httpjson.ErrorBody](t, body), httpjson.ErrorBody{Code: inventory.CodeInvalidQuery, Message: tc.want})
		})
	}
}

func TestListReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := list(t, fixedStore{err: errors.New(storeFailure)}, url.Values{})

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, decode[httpjson.ErrorBody](t, body).Code, httpjson.CodeInternal)
}
