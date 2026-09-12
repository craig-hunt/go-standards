package expect

import (
	"errors"
	"fmt"
	"testing"
)

type recorder struct {
	helpers    int
	errors     int
	fatals     int
	lastFormat string
}

func (r *recorder) Helper() { r.helpers++ }

func (r *recorder) Errorf(format string, _ ...any) {
	r.errors++
	r.lastFormat = format
}

func (r *recorder) Fatalf(format string, _ ...any) {
	r.fatals++
	r.lastFormat = format
}

func assertCalls(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf(countMismatch, name, got, want)
	}
}

func assertFormat(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf(formatMismatchIn, got, want)
	}
}

func TestEqualStaysSilentForEqualValues(t *testing.T) {
	r := &recorder{}
	Equal(r, []int{sampleValue}, []int{sampleValue})
	assertCalls(t, errorsCalls, r.errors, 0)
	assertCalls(t, helperCalls, r.helpers, 1)
}

func TestEqualReportsDifferentValues(t *testing.T) {
	r := &recorder{}
	Equal(r, sampleValue, otherValue)
	assertCalls(t, errorsCalls, r.errors, 1)
	assertFormat(t, r.lastFormat, formatMismatch)
}

func TestNoErrorStaysSilentForNil(t *testing.T) {
	r := &recorder{}
	NoError(r, nil)
	assertCalls(t, fatalCalls, r.fatals, 0)
	assertCalls(t, helperCalls, r.helpers, 1)
}

func TestNoErrorStopsTheTestOnAnError(t *testing.T) {
	r := &recorder{}
	NoError(r, errors.New(sampleMessage))
	assertCalls(t, fatalCalls, r.fatals, 1)
	assertFormat(t, r.lastFormat, formatUnexpectedError)
}

func TestErrorIsAcceptsAWrappedTarget(t *testing.T) {
	r := &recorder{}
	target := errors.New(sampleMessage)
	ErrorIs(r, fmt.Errorf(wrapFormat, target), target)
	assertCalls(t, errorsCalls, r.errors, 0)
	assertCalls(t, helperCalls, r.helpers, 1)
}

func TestErrorIsReportsAnUnrelatedError(t *testing.T) {
	r := &recorder{}
	ErrorIs(r, errors.New(sampleMessage), errors.New(sampleMessage))
	assertCalls(t, errorsCalls, r.errors, 1)
	assertFormat(t, r.lastFormat, formatWrongError)
}

func TestTrueStaysSilentWhenTheConditionHolds(t *testing.T) {
	r := &recorder{}
	True(r, true, sampleCondition)
	assertCalls(t, errorsCalls, r.errors, 0)
	assertCalls(t, helperCalls, r.helpers, 1)
}

func TestTrueReportsAFailedCondition(t *testing.T) {
	r := &recorder{}
	True(r, false, sampleCondition)
	assertCalls(t, errorsCalls, r.errors, 1)
	assertFormat(t, r.lastFormat, formatFalse)
}
