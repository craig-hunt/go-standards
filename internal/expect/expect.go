// Package expect holds the assertion helpers every test in this module shares,
// so failures read the same everywhere without a third-party assertion library.
package expect

import (
	"errors"
	"reflect"
)

// Reporter is the slice of testing.TB these helpers use. Accepting the
// interface rather than *testing.T lets the helpers' own tests observe failures.
type Reporter interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

func Equal[T any](r Reporter, got, want T) {
	r.Helper()
	if !reflect.DeepEqual(got, want) {
		r.Errorf(formatMismatch, got, want)
	}
}

func NoError(r Reporter, err error) {
	r.Helper()
	if err != nil {
		r.Fatalf(formatUnexpectedError, err)
	}
}

func ErrorIs(r Reporter, err, target error) {
	r.Helper()
	if !errors.Is(err, target) {
		r.Errorf(formatWrongError, err, target)
	}
}

func True(r Reporter, condition bool, description string) {
	r.Helper()
	if !condition {
		r.Errorf(formatFalse, description)
	}
}
