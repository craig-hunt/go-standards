package health

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/logcapture"
)

const (
	pingFailure    = "database unreachable"
	deadlineReason = "the ping runs under a deadline no longer than the readiness timeout"
)

type fakePinger struct {
	err         error
	deadline    time.Time
	hadDeadline bool
}

func (p *fakePinger) Ping(ctx context.Context) error {
	p.deadline, p.hadDeadline = ctx.Deadline()
	return p.err
}

func probe(t *testing.T, pinger Pinger, path string) (*httptest.ResponseRecorder, *logcapture.Capture) {
	t.Helper()
	logger, capture := logcapture.New()
	mux := http.NewServeMux()
	Register(mux, pinger, logger)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return recorder, capture
}

func report(t *testing.T, recorder *httptest.ResponseRecorder) Report {
	t.Helper()
	var body Report
	expect.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestLivenessReportsOKWithoutConsultingTheDatabase(t *testing.T) {
	pinger := &fakePinger{err: errors.New(pingFailure)}

	recorder, _ := probe(t, pinger, PathLive)

	expect.Equal(t, recorder.Code, http.StatusOK)
	expect.Equal(t, report(t, recorder), Report{Status: StatusOK})
	expect.Equal(t, pinger.hadDeadline, false)
}

func TestReadinessReportsOKWhenTheDatabaseAnswers(t *testing.T) {
	recorder, capture := probe(t, &fakePinger{}, PathReady)

	expect.Equal(t, recorder.Code, http.StatusOK)
	expect.Equal(t, report(t, recorder), Report{Status: StatusOK})
	expect.Equal(t, capture.Find(t, MsgNotReady), map[string]any(nil))
}

func TestReadinessReportsUnavailableWhenTheDatabaseFails(t *testing.T) {
	recorder, capture := probe(t, &fakePinger{err: errors.New(pingFailure)}, PathReady)

	expect.Equal(t, recorder.Code, http.StatusServiceUnavailable)
	expect.Equal(t, report(t, recorder), Report{Status: StatusUnavailable})
	record := capture.Find(t, MsgNotReady)
	expect.Equal(t, record[slog.LevelKey], any(slog.LevelWarn.String()))
	expect.Equal(t, record[httpjson.LogKeyError], any(pingFailure))
}

func TestReadinessBoundsThePingWithTheReadinessTimeout(t *testing.T) {
	pinger := &fakePinger{}

	probe(t, pinger, PathReady)

	remaining := time.Until(pinger.deadline)
	expect.True(t, pinger.hadDeadline && remaining > 0 && remaining <= ReadyTimeout, deadlineReason)
}
