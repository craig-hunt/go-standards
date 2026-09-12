// Package apitest drives HTTP handlers in tests: it encodes a request body,
// serves the request, and decodes the response, so handler tests state only
// what they send and what they expect back.
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/craig-hunt/go-standards/internal/expect"
)

const (
	formatUnencodable = "request body does not encode: %v"
	formatUndecodable = "response body %q does not decode: %v"
)

type Request struct {
	Method  string
	Target  string
	Body    any
	Headers map[string]string
}

func Do(r expect.Reporter, handler http.Handler, request Request) *httptest.ResponseRecorder {
	r.Helper()
	var body io.Reader
	if request.Body != nil {
		encoded, err := json.Marshal(request.Body)
		if err != nil {
			r.Fatalf(formatUnencodable, err)
			return nil
		}
		body = bytes.NewReader(encoded)
	}
	built := httptest.NewRequestWithContext(context.Background(), request.Method, request.Target, body)
	for name, value := range request.Headers {
		built.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, built)
	return recorder
}

func Decode[T any](r expect.Reporter, recorder *httptest.ResponseRecorder) T {
	r.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		r.Fatalf(formatUndecodable, recorder.Body.String(), err)
	}
	return value
}
