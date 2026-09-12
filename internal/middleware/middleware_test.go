package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/logcapture"
	"github.com/craig-hunt/go-standards/internal/requestid"
)

func steppingClock(start time.Time, step time.Duration) func() time.Time {
	current := start
	return func() time.Time {
		now := current
		current = current.Add(step)
		return now
	}
}

func statusHandler(status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
}

func serve(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func newRequest() *http.Request {
	return httptest.NewRequestWithContext(context.Background(), http.MethodGet, targetPath, nil)
}

func readError(t *testing.T, recorder *httptest.ResponseRecorder) httpjson.ErrorBody {
	t.Helper()
	var body httpjson.ErrorBody
	expect.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestChainRunsTheFirstLayerOutermost(t *testing.T) {
	var steps []string
	named := func(name string) Layer {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				steps = append(steps, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		steps = append(steps, handlerStep)
	})

	serve(Chain(handler, named(outerLayer), named(innerLayer)), newRequest())

	expect.Equal(t, steps, []string{outerLayer, innerLayer, handlerStep})
}

func TestChainWithoutLayersServesTheHandlerDirectly(t *testing.T) {
	recorder := serve(Chain(statusHandler(http.StatusAccepted)), newRequest())

	expect.Equal(t, recorder.Code, http.StatusAccepted)
}

func TestRequestIDEchoesASuppliedIdentifier(t *testing.T) {
	var seen string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestid.From(r.Context())
	})
	request := newRequest()
	request.Header.Set(requestid.Header, suppliedID)

	recorder := serve(RequestID(handler), request)

	expect.Equal(t, recorder.Header().Get(requestid.Header), suppliedID)
	expect.Equal(t, seen, suppliedID)
}

func TestRequestIDGeneratesAnIdentifierWhenNoneArrives(t *testing.T) {
	var seen string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = requestid.From(r.Context())
	})

	recorder := serve(RequestID(handler), newRequest())

	header := recorder.Header().Get(requestid.Header)
	expect.True(t, header != "", generatedReason)
	expect.True(t, seen == header, contextReason)
}

func TestLoggingRecordsTheRequestAndItsOutcome(t *testing.T) {
	logger, capture := logcapture.New()
	handler := Logging(logger, steppingClock(startInstant, stepDuration))(statusHandler(http.StatusCreated))
	request := newRequest()
	request = request.WithContext(requestid.With(request.Context(), suppliedID))

	serve(handler, request)

	record := capture.Find(t, MsgRequestHandled)
	expect.Equal(t, record[slog.LevelKey], any(slog.LevelInfo.String()))
	expect.Equal(t, record[requestid.LogKey], any(suppliedID))
	expect.Equal(t, record[LogKeyMethod], any(http.MethodGet))
	expect.Equal(t, record[LogKeyPath], any(targetPath))
	expect.Equal(t, record[LogKeyStatus], any(float64(http.StatusCreated)))
	expect.Equal(t, record[LogKeyDurationMS], any(float64(stepMilliseconds)))
}

func TestLoggingReportsOKWhenTheHandlerNeverSetsAStatus(t *testing.T) {
	logger, capture := logcapture.New()
	silent := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	serve(Logging(logger, steppingClock(startInstant, stepDuration))(silent), newRequest())

	expect.Equal(t, capture.Find(t, MsgRequestHandled)[LogKeyStatus], any(float64(http.StatusOK)))
}

func TestRecoverTurnsAPanicIntoAnInternalServerError(t *testing.T) {
	logger, capture := logcapture.New()
	exploding := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(panicMessage)
	})
	request := newRequest()
	request = request.WithContext(requestid.With(request.Context(), suppliedID))

	recorder := serve(Recover(logger)(exploding), request)

	expect.Equal(t, recorder.Code, http.StatusInternalServerError)
	expect.Equal(t, readError(t, recorder), httpjson.ErrorBody{Code: httpjson.CodeInternal, Message: httpjson.MsgInternal})
	record := capture.Find(t, MsgPanicRecovered)
	expect.Equal(t, record[slog.LevelKey], any(slog.LevelError.String()))
	expect.Equal(t, record[LogKeyPanic], any(panicMessage))
	expect.Equal(t, record[requestid.LogKey], any(suppliedID))
}

func TestRecoverLeavesAHealthyHandlerAlone(t *testing.T) {
	logger, capture := logcapture.New()

	recorder := serve(Recover(logger)(statusHandler(http.StatusNoContent)), newRequest())

	expect.Equal(t, recorder.Code, http.StatusNoContent)
	expect.Equal(t, capture.Find(t, MsgPanicRecovered), map[string]any(nil))
}

func TestRequireBearerTokenAdmitsTheConfiguredToken(t *testing.T) {
	logger, _ := logcapture.New()
	request := newRequest()
	request.Header.Set(HeaderAuthorization, BearerPrefix+validToken)

	recorder := serve(RequireBearerToken(validToken, logger)(statusHandler(http.StatusNoContent)), request)

	expect.Equal(t, recorder.Code, http.StatusNoContent)
}

func TestRequireBearerTokenRejectsAnythingElse(t *testing.T) {
	cases := []struct {
		name          string
		authorization string
	}{
		{name: "no authorization header", authorization: ""},
		{name: "wrong token", authorization: BearerPrefix + wrongToken},
		{name: "token without the scheme", authorization: validToken},
		{name: "scheme in the wrong case", authorization: lowercaseScheme + validToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger, _ := logcapture.New()
			called := false
			protected := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
			request := newRequest()
			request.Header.Set(HeaderAuthorization, tc.authorization)

			recorder := serve(RequireBearerToken(validToken, logger)(protected), request)

			expect.Equal(t, recorder.Code, http.StatusUnauthorized)
			expect.Equal(t, recorder.Header().Get(HeaderWWWAuthenticate), BearerScheme)
			expect.Equal(t, readError(t, recorder), httpjson.ErrorBody{Code: CodeUnauthorized, Message: MsgUnauthorized})
			expect.True(t, !called, notCalledReason)
		})
	}
}
