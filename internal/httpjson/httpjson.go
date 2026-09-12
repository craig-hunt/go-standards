// Package httpjson reads and writes the JSON bodies every handler exchanges, so
// status codes, headers, and error shapes stay uniform across the service.
package httpjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/craig-hunt/go-standards/internal/requestid"
)

var ErrInvalidBody = errors.New(MsgInvalidBody)

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Write takes the request so a failure to encode logs with the request's
// identifier, like every other line the request produces.
func Write(w http.ResponseWriter, r *http.Request, logger *slog.Logger, status int, body any) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.LogAttrs(r.Context(), slog.LevelWarn, msgWriteFailed,
			requestid.Attr(r.Context()),
			slog.Any(LogKeyError, err))
	}
}

func WriteError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, status int, code, message string) {
	Write(w, r, logger, status, ErrorBody{Code: code, Message: message})
}

func WriteInvalidBody(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
	WriteError(w, r, logger, http.StatusBadRequest, CodeInvalidBody, MsgInvalidBody)
}

func WriteFieldErrors(w http.ResponseWriter, r *http.Request, logger *slog.Logger, fields map[string]string) {
	Write(w, r, logger, http.StatusUnprocessableEntity, ErrorBody{Code: CodeValidation, Message: MsgValidation, Fields: fields})
}

// WriteInternal logs the cause with the request's identifier and returns a
// generic body, so internal detail never reaches the caller.
func WriteInternal(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.LogAttrs(r.Context(), slog.LevelError, msgRequestFailed,
		requestid.Attr(r.Context()),
		slog.Any(LogKeyError, err))
	WriteError(w, r, logger, http.StatusInternalServerError, CodeInternal, MsgInternal)
}

// Decode reads the value into json.RawMessage first because decoding null into
// a struct succeeds silently. Only an object, followed by nothing but
// whitespace, reaches the strict decode into T.
func Decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var value T
	stream := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	var raw json.RawMessage
	if err := stream.Decode(&raw); err != nil {
		return value, fmt.Errorf(wrapFormat, ErrInvalidBody, err)
	}
	if raw[0] != objectStart {
		return value, ErrInvalidBody
	}
	if err := stream.Decode(&json.RawMessage{}); !errors.Is(err, io.EOF) {
		return value, ErrInvalidBody
	}

	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&value); err != nil {
		return value, fmt.Errorf(wrapFormat, ErrInvalidBody, err)
	}
	return value, nil
}
