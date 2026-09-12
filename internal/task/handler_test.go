package task_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/craig-hunt/go-standards/internal/apitest"
	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/httpjson"
	"github.com/craig-hunt/go-standards/internal/logcapture"
	"github.com/craig-hunt/go-standards/internal/task"
)

func seededStore(t *testing.T) *memoryStore {
	t.Helper()
	store := newMemoryStore()
	for _, title := range task.SeedTitles {
		_, err := store.Create(context.Background(), title)
		expect.NoError(t, err)
	}
	return store
}

func failingStore() *memoryStore {
	store := newMemoryStore()
	store.err = errors.New(storeFailure)
	return store
}

func serve(t *testing.T, store task.Store, request apitest.Request) (int, []byte) {
	t.Helper()
	logger, _ := logcapture.New()
	mux := http.NewServeMux()
	task.NewHandler(store, logger).Register(mux)
	recorder := apitest.Do(t, mux, request)
	return recorder.Code, recorder.Body.Bytes()
}

func itemPath(id task.ID) string {
	return task.PathTasks + pathSeparator + fmt.Sprint(id)
}

func rawItemPath(raw string) string {
	return task.PathTasks + pathSeparator + raw
}

func filteredPath(filter string) string {
	return task.PathTasks + querySeparator + url.Values{task.QueryFilter: {filter}}.Encode()
}

func decodeError(t *testing.T, store task.Store, request apitest.Request) (int, httpjson.ErrorBody) {
	t.Helper()
	logger, _ := logcapture.New()
	mux := http.NewServeMux()
	task.NewHandler(store, logger).Register(mux)
	recorder := apitest.Do(t, mux, request)
	return recorder.Code, apitest.Decode[httpjson.ErrorBody](t, recorder)
}

func decodeBody[T any](t *testing.T, store task.Store, request apitest.Request) (int, T) {
	t.Helper()
	logger, _ := logcapture.New()
	mux := http.NewServeMux()
	task.NewHandler(store, logger).Register(mux)
	recorder := apitest.Do(t, mux, request)
	return recorder.Code, apitest.Decode[T](t, recorder)
}

func internalError() httpjson.ErrorBody {
	return httpjson.ErrorBody{Code: httpjson.CodeInternal, Message: httpjson.MsgInternal}
}

func TestListReturnsEveryTaskWithCounts(t *testing.T) {
	store := seededStore(t)

	status, view := decodeBody[task.View](t, store, apitest.Request{Method: http.MethodGet, Target: task.PathTasks})

	expect.Equal(t, status, http.StatusOK)
	expect.Equal(t, view, task.View{Tasks: store.tasks, Remaining: len(task.SeedTitles), Total: len(task.SeedTitles)})
}

func TestListAppliesTheRequestedFilter(t *testing.T) {
	store := seededStore(t)
	completed, err := store.SetCompleted(context.Background(), firstSeededID, true)
	expect.NoError(t, err)

	status, view := decodeBody[task.View](t, store, apitest.Request{Method: http.MethodGet, Target: filteredPath(string(task.FilterCompleted))})

	expect.Equal(t, status, http.StatusOK)
	expect.Equal(t, view, task.View{Tasks: []task.Task{completed}, Remaining: remainingAfter, Total: len(task.SeedTitles)})
}

func TestListRejectsAnUnknownFilter(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{Method: http.MethodGet, Target: filteredPath(unknownFilter)})

	expect.Equal(t, status, http.StatusBadRequest)
	expect.Equal(t, body, httpjson.ErrorBody{Code: task.CodeInvalidFilter, Message: task.MsgInvalidFilter})
}

func TestListReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := decodeError(t, failingStore(), apitest.Request{Method: http.MethodGet, Target: task.PathTasks})

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, body, internalError())
}

func TestCreateStoresTheTrimmedTitleAndReturnsTheTask(t *testing.T) {
	store := seededStore(t)

	status, created := decodeBody[task.Task](t, store, apitest.Request{
		Method: http.MethodPost, Target: task.PathTasks, Body: map[string]string{task.FieldTitle: paddedTitle},
	})

	expect.Equal(t, status, http.StatusCreated)
	expect.Equal(t, created, task.Task{ID: createdID, Title: sampleTitle})
	expect.Equal(t, store.tasks[len(store.tasks)-1], created)
}

func TestCreateNamesTheTitleWhenItIsBlank(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPost, Target: task.PathTasks, Body: map[string]string{task.FieldTitle: whitespaceTitle},
	})

	expect.Equal(t, status, http.StatusUnprocessableEntity)
	expect.Equal(t, body.Fields, map[string]string{task.FieldTitle: task.MsgTitleRequired})
}

func TestCreateNamesTheTitleWhenItIsTooLong(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPost, Target: task.PathTasks, Body: map[string]string{task.FieldTitle: strings.Repeat(filler, task.MaxTitleLength+1)},
	})

	expect.Equal(t, status, http.StatusUnprocessableEntity)
	expect.Equal(t, body.Fields, map[string]string{task.FieldTitle: task.MsgTitleTooLong})
}

func TestCreateRejectsABodyWithUnknownFields(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPost, Target: task.PathTasks, Body: map[string]string{unknownField: sampleTitle},
	})

	expect.Equal(t, status, http.StatusBadRequest)
	expect.Equal(t, body, httpjson.ErrorBody{Code: httpjson.CodeInvalidBody, Message: httpjson.MsgInvalidBody})
}

func TestCreateReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := decodeError(t, failingStore(), apitest.Request{
		Method: http.MethodPost, Target: task.PathTasks, Body: map[string]string{task.FieldTitle: sampleTitle},
	})

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, body, internalError())
}

func TestUpdateMarksTheTaskCompleted(t *testing.T) {
	store := seededStore(t)

	status, updated := decodeBody[task.Task](t, store, apitest.Request{
		Method: http.MethodPatch, Target: itemPath(firstSeededID), Body: map[string]bool{task.FieldCompleted: true},
	})

	expect.Equal(t, status, http.StatusOK)
	expect.Equal(t, updated, task.Task{ID: firstSeededID, Title: task.SeedTitles[0], Completed: true})
}

func TestUpdateNamesTheCompletedFieldWhenItIsMissing(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPatch, Target: itemPath(firstSeededID), Body: map[string]bool{},
	})

	expect.Equal(t, status, http.StatusUnprocessableEntity)
	expect.Equal(t, body.Fields, map[string]string{task.FieldCompleted: task.MsgCompletedRequired})
}

func TestUpdateRejectsABodyWithUnknownFields(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPatch, Target: itemPath(firstSeededID), Body: map[string]string{unknownField: sampleTitle},
	})

	expect.Equal(t, status, http.StatusBadRequest)
	expect.Equal(t, body.Code, httpjson.CodeInvalidBody)
}

func TestUpdateRejectsAnIDThatIsNotAPositiveNumber(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPatch, Target: rawItemPath(notANumber), Body: map[string]bool{task.FieldCompleted: true},
	})

	expect.Equal(t, status, http.StatusBadRequest)
	expect.Equal(t, body, httpjson.ErrorBody{Code: task.CodeInvalidID, Message: task.MsgInvalidID})
}

func TestUpdateReportsATaskThatDoesNotExist(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{
		Method: http.MethodPatch, Target: itemPath(missingID), Body: map[string]bool{task.FieldCompleted: true},
	})

	expect.Equal(t, status, http.StatusNotFound)
	expect.Equal(t, body, httpjson.ErrorBody{Code: task.CodeNotFound, Message: task.MsgNotFound})
}

func TestUpdateReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := decodeError(t, failingStore(), apitest.Request{
		Method: http.MethodPatch, Target: itemPath(firstSeededID), Body: map[string]bool{task.FieldCompleted: true},
	})

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, body, internalError())
}

func TestDeleteRemovesTheTask(t *testing.T) {
	store := seededStore(t)

	status, _ := serve(t, store, apitest.Request{Method: http.MethodDelete, Target: itemPath(firstSeededID)})

	expect.Equal(t, status, http.StatusNoContent)
	expect.Equal(t, len(store.tasks), storeLengthAfter)
	expect.Equal(t, store.tasks[0].ID, secondSeededID)
}

func TestDeleteRejectsAnIDThatIsNotAPositiveNumber(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{Method: http.MethodDelete, Target: rawItemPath(zeroID)})

	expect.Equal(t, status, http.StatusBadRequest)
	expect.Equal(t, body.Code, task.CodeInvalidID)
}

func TestDeleteReportsATaskThatDoesNotExist(t *testing.T) {
	status, body := decodeError(t, seededStore(t), apitest.Request{Method: http.MethodDelete, Target: itemPath(missingID)})

	expect.Equal(t, status, http.StatusNotFound)
	expect.Equal(t, body.Code, task.CodeNotFound)
}

func TestDeleteReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := decodeError(t, failingStore(), apitest.Request{Method: http.MethodDelete, Target: itemPath(firstSeededID)})

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, body, internalError())
}

func TestClearCompletedRemovesFinishedTasksAndCountsThem(t *testing.T) {
	store := seededStore(t)
	_, err := store.SetCompleted(context.Background(), firstSeededID, true)
	expect.NoError(t, err)

	status, result := decodeBody[task.ClearResult](t, store, apitest.Request{Method: http.MethodDelete, Target: task.PathTasks})

	expect.Equal(t, status, http.StatusOK)
	expect.Equal(t, result, task.ClearResult{Removed: oneRemoved})
	expect.Equal(t, len(store.tasks), storeLengthAfter)
}

func TestClearCompletedReportsAStoreFailureAsAnInternalError(t *testing.T) {
	status, body := decodeError(t, failingStore(), apitest.Request{Method: http.MethodDelete, Target: task.PathTasks})

	expect.Equal(t, status, http.StatusInternalServerError)
	expect.Equal(t, body, internalError())
}
