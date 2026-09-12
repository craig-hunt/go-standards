package task_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/task"
)

func TestNewTitleTrimsSurroundingWhitespace(t *testing.T) {
	title, err := task.NewTitle(paddedTitle)

	expect.NoError(t, err)
	expect.Equal(t, title, task.Title(sampleTitle))
}

func TestNewTitleRequiresVisibleText(t *testing.T) {
	for _, raw := range []string{"", whitespaceTitle} {
		_, err := task.NewTitle(raw)

		expect.ErrorIs(t, err, task.ErrTitleRequired)
	}
}

func TestNewTitleAcceptsTheLongestAllowedTitle(t *testing.T) {
	longest := strings.Repeat(filler, task.MaxTitleLength)

	title, err := task.NewTitle(longest)

	expect.NoError(t, err)
	expect.Equal(t, title, task.Title(longest))
}

func TestNewTitleRejectsATitleOverTheLimit(t *testing.T) {
	_, err := task.NewTitle(strings.Repeat(filler, task.MaxTitleLength+1))

	expect.ErrorIs(t, err, task.ErrTitleTooLong)
}

func TestNewTitleCountsCharactersRatherThanBytes(t *testing.T) {
	_, err := task.NewTitle(strings.Repeat(multibyteFiller, task.MaxTitleLength))

	expect.NoError(t, err)
}

func TestParseFilterAcceptsEveryKnownFilter(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want task.Filter
	}{
		{name: "no filter means all", raw: "", want: task.FilterAll},
		{name: "all", raw: string(task.FilterAll), want: task.FilterAll},
		{name: "active", raw: string(task.FilterActive), want: task.FilterActive},
		{name: "completed", raw: string(task.FilterCompleted), want: task.FilterCompleted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := task.ParseFilter(tc.raw)

			expect.NoError(t, err)
			expect.Equal(t, filter, tc.want)
		})
	}
}

func TestParseFilterRejectsAnUnknownFilter(t *testing.T) {
	_, err := task.ParseFilter(unknownFilter)

	expect.ErrorIs(t, err, task.ErrInvalidFilter)
}

func TestParseIDAcceptsPositiveWholeNumbers(t *testing.T) {
	id, err := task.ParseID(validID)
	expect.NoError(t, err)
	expect.Equal(t, id, validIDValue)

	smallest, err := task.ParseID(strconv.Itoa(int(firstSeededID)))
	expect.NoError(t, err)
	expect.Equal(t, smallest, firstSeededID)
}

func TestParseIDRejectsAnythingButAPositiveWholeNumber(t *testing.T) {
	for _, raw := range []string{notANumber, zeroID, negativeID, ""} {
		_, err := task.ParseID(raw)

		expect.ErrorIs(t, err, task.ErrInvalidID)
	}
}

func TestFilterIncludesOnlyMatchingTasks(t *testing.T) {
	open := task.Task{ID: firstSeededID, Title: task.SeedTitles[0]}
	done := task.Task{ID: secondSeededID, Title: task.SeedTitles[1], Completed: true}
	cases := []struct {
		name       string
		filter     task.Filter
		wantOpen   bool
		wantClosed bool
	}{
		{name: "all includes both", filter: task.FilterAll, wantOpen: true, wantClosed: true},
		{name: "active includes only open tasks", filter: task.FilterActive, wantOpen: true, wantClosed: false},
		{name: "completed includes only finished tasks", filter: task.FilterCompleted, wantOpen: false, wantClosed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expect.Equal(t, tc.filter.Includes(open), tc.wantOpen)
			expect.Equal(t, tc.filter.Includes(done), tc.wantClosed)
		})
	}
}

func TestSummarizeCountsEveryTaskWhateverTheFilterShows(t *testing.T) {
	open := task.Task{ID: firstSeededID, Title: task.SeedTitles[0]}
	done := task.Task{ID: secondSeededID, Title: task.SeedTitles[1], Completed: true}
	all := []task.Task{open, done}

	expect.Equal(t, task.Summarize(all, task.FilterAll), task.View{Tasks: all, Remaining: remainingAfter, Total: len(all)})
	expect.Equal(t, task.Summarize(all, task.FilterActive), task.View{Tasks: []task.Task{open}, Remaining: remainingAfter, Total: len(all)})
	expect.Equal(t, task.Summarize(all, task.FilterCompleted), task.View{Tasks: []task.Task{done}, Remaining: remainingAfter, Total: len(all)})
}

func TestSummarizeReturnsAnEmptyListForNoTasks(t *testing.T) {
	expect.Equal(t, task.Summarize(nil, task.FilterAll), task.View{Tasks: []task.Task{}})
}
