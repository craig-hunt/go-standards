package api_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/craig-hunt/go-standards/internal/api"
	"github.com/craig-hunt/go-standards/internal/apitest"
	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/health"
	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/inventory"
	"github.com/craig-hunt/go-standards/internal/logcapture"
	"github.com/craig-hunt/go-standards/internal/middleware"
	"github.com/craig-hunt/go-standards/internal/requestid"
	"github.com/craig-hunt/go-standards/internal/signup"
	"github.com/craig-hunt/go-standards/internal/task"
)

const (
	apiToken       = "api-test-token"
	pingFailure    = "database unreachable"
	panicMessage   = "store exploded"
	unknownAPIPath = api.PathPrefix + "missing"
	requestReason  = "every response carries a request identifier"
)

type taskStub struct {
	panics bool
}

func (s taskStub) List(context.Context) ([]task.Task, error) {
	if s.panics {
		panic(panicMessage)
	}
	return []task.Task{}, nil
}

func (taskStub) Create(context.Context, task.Title) (task.Task, error) { return task.Task{}, nil }

func (taskStub) SetCompleted(context.Context, task.ID, bool) (task.Task, error) {
	return task.Task{}, nil
}

func (taskStub) Delete(context.Context, task.ID) error { return nil }

func (taskStub) DeleteCompleted(context.Context) (int64, error) { return 0, nil }

type signupStub struct{}

func (signupStub) Save(context.Context, signup.Signup) (signup.ID, error) { return 0, nil }

type inventoryStub struct{}

func (inventoryStub) Items(context.Context) ([]inventory.Item, error) {
	return inventory.SeedItems, nil
}

type pingerStub struct {
	err error
}

func (p pingerStub) Ping(context.Context) error { return p.err }

func newAPI(tasks task.Store, pinger health.Pinger) (http.Handler, *logcapture.Capture) {
	logger, capture := logcapture.New()
	return api.NewHandler(api.Dependencies{
		Tasks:     tasks,
		Signups:   signupStub{},
		Inventory: inventoryStub{},
		Pinger:    pinger,
		APIToken:  apiToken,
		Logger:    logger,
		Now:       time.Now,
	}), capture
}

func authorized(method, target string) apitest.Request {
	return apitest.Request{
		Method:  method,
		Target:  target,
		Headers: map[string]string{middleware.HeaderAuthorization: middleware.BearerPrefix + apiToken},
	}
}

func TestHealthProbesAnswerWithoutAToken(t *testing.T) {
	handler, _ := newAPI(taskStub{}, pingerStub{})

	live := apitest.Do(t, handler, apitest.Request{Method: http.MethodGet, Target: health.PathLive})
	ready := apitest.Do(t, handler, apitest.Request{Method: http.MethodGet, Target: health.PathReady})

	expect.Equal(t, live.Code, http.StatusOK)
	expect.Equal(t, ready.Code, http.StatusOK)
}

func TestReadinessReflectsTheDatabase(t *testing.T) {
	handler, _ := newAPI(taskStub{}, pingerStub{err: errors.New(pingFailure)})

	ready := apitest.Do(t, handler, apitest.Request{Method: http.MethodGet, Target: health.PathReady})

	expect.Equal(t, ready.Code, http.StatusServiceUnavailable)
}

func TestAPIRoutesRequireTheBearerToken(t *testing.T) {
	handler, _ := newAPI(taskStub{}, pingerStub{})

	for _, target := range []string{task.PathTasks, signup.PathSignups, inventory.PathInventory, unknownAPIPath} {
		response := apitest.Do(t, handler, apitest.Request{Method: http.MethodGet, Target: target})

		expect.Equal(t, response.Code, http.StatusUnauthorized)
	}
}

func TestEveryFeatureAnswersWithTheToken(t *testing.T) {
	handler, _ := newAPI(taskStub{}, pingerStub{})

	tasks := apitest.Do(t, handler, authorized(http.MethodGet, task.PathTasks))
	items := apitest.Do(t, handler, authorized(http.MethodGet, inventory.PathInventory))
	signups := apitest.Do(t, handler, authorized(http.MethodPost, signup.PathSignups))

	expect.Equal(t, tasks.Code, http.StatusOK)
	expect.Equal(t, items.Code, http.StatusOK)
	expect.Equal(t, signups.Code, http.StatusBadRequest)
}

func TestResponsesCarryARequestIdentifierAndAnAccessLogLine(t *testing.T) {
	handler, capture := newAPI(taskStub{}, pingerStub{})

	response := apitest.Do(t, handler, authorized(http.MethodGet, task.PathTasks))

	id := response.Header().Get(requestid.Header)
	expect.True(t, id != "", requestReason)
	record := capture.Find(t, middleware.MsgRequestHandled)
	expect.Equal(t, record[requestid.LogKey], any(id))
	expect.Equal(t, record[middleware.LogKeyStatus], any(float64(http.StatusOK)))
}

func TestAPanicBecomesALoggedInternalError(t *testing.T) {
	handler, capture := newAPI(taskStub{panics: true}, pingerStub{})

	response := apitest.Do(t, handler, authorized(http.MethodGet, task.PathTasks))

	expect.Equal(t, response.Code, http.StatusInternalServerError)
	expect.Equal(t, apitest.Decode[httpjson.ErrorBody](t, response).Code, httpjson.CodeInternal)
	expect.Equal(t, capture.Find(t, middleware.MsgPanicRecovered)[slog.LevelKey], any(slog.LevelError.String()))
	expect.Equal(t, capture.Find(t, middleware.MsgRequestHandled)[middleware.LogKeyStatus], any(float64(http.StatusInternalServerError)))
}
