package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/logcapture"
	"github.com/craig-hunt/go-standards/internal/requestid"
)

type sample struct {
	Name string `json:"name"`
}

func marshal(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	expect.NoError(t, err)
	return body
}

func decodeRequest(body []byte) (sample, error) {
	request := httptest.NewRequestWithContext(context.Background(), methodPost, targetPath, bytes.NewReader(body))
	return Decode[sample](httptest.NewRecorder(), request)
}

func readError(t *testing.T, recorder *httptest.ResponseRecorder) ErrorBody {
	t.Helper()
	var body ErrorBody
	expect.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestWriteSendsStatusContentTypeAndBody(t *testing.T) {
	logger, _ := logcapture.New()
	recorder := httptest.NewRecorder()

	Write(recorder, logger, http.StatusCreated, sample{Name: sampleName})

	expect.Equal(t, recorder.Code, http.StatusCreated)
	expect.Equal(t, recorder.Header().Get(HeaderContentType), ContentTypeJSON)
	var body sample
	expect.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	expect.Equal(t, body, sample{Name: sampleName})
}

func TestWriteLogsWhenTheBodyCannotEncode(t *testing.T) {
	logger, capture := logcapture.New()

	Write(httptest.NewRecorder(), logger, http.StatusOK, map[string]any{unencodableValue: make(chan int)})

	record := capture.Find(t, msgWriteFailed)
	expect.Equal(t, record[slog.LevelKey], any(slog.LevelWarn.String()))
}

func TestWriteErrorSendsCodeAndMessage(t *testing.T) {
	logger, _ := logcapture.New()
	recorder := httptest.NewRecorder()

	WriteError(recorder, logger, http.StatusBadRequest, CodeInvalidBody, MsgInvalidBody)

	expect.Equal(t, recorder.Code, http.StatusBadRequest)
	expect.Equal(t, readError(t, recorder), ErrorBody{Code: CodeInvalidBody, Message: MsgInvalidBody})
}

func TestWriteInvalidBodySendsBadRequestWithTheSharedMessage(t *testing.T) {
	logger, _ := logcapture.New()
	recorder := httptest.NewRecorder()

	WriteInvalidBody(recorder, logger)

	expect.Equal(t, recorder.Code, http.StatusBadRequest)
	expect.Equal(t, readError(t, recorder), ErrorBody{Code: CodeInvalidBody, Message: MsgInvalidBody})
}

func TestWriteFieldErrorsSendsUnprocessableEntityWithEachField(t *testing.T) {
	logger, _ := logcapture.New()
	recorder := httptest.NewRecorder()
	fields := map[string]string{fieldKey: fieldMessage}

	WriteFieldErrors(recorder, logger, fields)

	expect.Equal(t, recorder.Code, http.StatusUnprocessableEntity)
	expect.Equal(t, readError(t, recorder), ErrorBody{Code: CodeValidation, Message: MsgValidation, Fields: fields})
}

func TestWriteInternalHidesTheCauseAndLogsItWithTheRequestID(t *testing.T) {
	logger, capture := logcapture.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), methodPost, targetPath, nil)
	request = request.WithContext(requestid.With(request.Context(), sampleRequestID))

	WriteInternal(recorder, request, logger, errors.New(causeMessage))

	expect.Equal(t, recorder.Code, http.StatusInternalServerError)
	expect.Equal(t, readError(t, recorder), ErrorBody{Code: CodeInternal, Message: MsgInternal})
	record := capture.Find(t, msgRequestFailed)
	expect.Equal(t, record[slog.LevelKey], any(slog.LevelError.String()))
	expect.Equal(t, record[requestid.LogKey], any(sampleRequestID))
	expect.Equal(t, record[LogKeyError], any(causeMessage))
}

func TestDecodeReadsOneObjectWithKnownFields(t *testing.T) {
	value, err := decodeRequest(marshal(t, sample{Name: sampleName}))

	expect.NoError(t, err)
	expect.Equal(t, value, sample{Name: sampleName})
}

func TestDecodeRejectsMalformedBodies(t *testing.T) {
	single := marshal(t, sample{Name: sampleName})
	cases := []struct {
		name string
		body []byte
	}{
		{name: "unknown field", body: marshal(t, map[string]string{unknownField: sampleName})},
		{name: "second object after the first", body: append(append([]byte{}, single...), single...)},
		{name: "body over the size limit", body: marshal(t, sample{Name: strings.Repeat(fillerCharacter, MaxBodyBytes)})},
		{name: "empty body", body: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeRequest(tc.body)

			expect.ErrorIs(t, err, ErrInvalidBody)
		})
	}
}
