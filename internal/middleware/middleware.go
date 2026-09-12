// Package middleware wraps handlers with the behavior every request shares: an
// identifier, an access log line, panic recovery, and authentication.
package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/requestid"
)

type Layer func(http.Handler) http.Handler

// Chain applies layers so the first one listed runs outermost.
func Chain(handler http.Handler, layers ...Layer) http.Handler {
	for index := len(layers) - 1; index >= 0; index-- {
		handler = layers[index](handler)
	}
	return handler
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestid.Accept(r.Header.Get(requestid.Header))
		w.Header().Set(requestid.Header, id)
		next.ServeHTTP(w, r.WithContext(requestid.With(r.Context(), id)))
	})
}

// statusRecorder keeps the first status the client receives. net/http honors
// only the first WriteHeader call and sends 200 on the first body write, so a
// later WriteHeader, from a second call or from Recover after a partial write,
// must not change what the access log reports.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if !s.written {
		s.status = status
		s.written = true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(body []byte) (int, error) {
	if !s.written {
		s.status = http.StatusOK
		s.written = true
	}
	return s.ResponseWriter.Write(body)
}

// Logging takes the clock as a function so tests control the reported duration.
func Logging(logger *slog.Logger, now func() time.Time) Layer {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			logger.LogAttrs(r.Context(), slog.LevelInfo, MsgRequestHandled,
				requestid.Attr(r.Context()),
				slog.String(LogKeyMethod, r.Method),
				slog.String(LogKeyPath, r.URL.Path),
				slog.Int(LogKeyStatus, recorder.status),
				slog.Int64(LogKeyDurationMS, now().Sub(start).Milliseconds()))
		})
	}
}

func Recover(logger *slog.Logger) Layer {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverPanic(w, r, logger)
			next.ServeHTTP(w, r)
		})
	}
}

// recoverPanic must run as the deferred call itself, since recover only stops
// a panic when the deferred function calls it directly.
func recoverPanic(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	recovered := recover()
	if recovered == nil {
		return
	}
	logger.LogAttrs(r.Context(), slog.LevelError, MsgPanicRecovered,
		requestid.Attr(r.Context()),
		slog.Any(LogKeyPanic, recovered))
	httpjson.WriteError(w, r, logger, http.StatusInternalServerError, httpjson.CodeInternal, httpjson.MsgInternal)
}

// RequireBearerToken compares in constant time so response timing reveals
// nothing about how much of a guessed token matched.
func RequireBearerToken(token string, logger *slog.Logger) Layer {
	expected := []byte(BearerPrefix + token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get(HeaderAuthorization)), expected) != 1 {
				w.Header().Set(HeaderWWWAuthenticate, BearerScheme)
				httpjson.WriteError(w, r, logger, http.StatusUnauthorized, CodeUnauthorized, MsgUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
