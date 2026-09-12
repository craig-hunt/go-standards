// Package httpjson reads and writes the JSON bodies every handler exchanges, so
// status codes, headers, and error shapes stay uniform across the service.
package httpjson

import (
	"encoding/json"
	"errors"
	"fmt"
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

func Write(w http.ResponseWriter, logger *slog.Logger, status int, body any) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.Warn(msgWriteFailed, slog.Any(LogKeyError, err))
	}
}

func WriteError(w http.ResponseWriter, logger *slog.Logger, status int, code, message string) {
	Write(w, logger, status, ErrorBody{Code: code, Message: message})
}

func WriteInvalidBody(w http.ResponseWriter, logger *slog.Logger) {
	WriteError(w, logger, http.StatusBadRequest, CodeInvalidBody, MsgInvalidBody)
}

func WriteFieldErrors(w http.ResponseWriter, logger *slog.Logger, fields map[string]string) {
	Write(w, logger, http.StatusUnprocessableEntity, ErrorBody{Code: CodeValidation, Message: MsgValidation, Fields: fields})
}

// WriteInternal logs the cause with the request's identifier and returns a
// generic body, so internal detail never reaches the caller.
func WriteInternal(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.LogAttrs(r.Context(), slog.LevelError, msgRequestFailed,
		slog.String(requestid.LogKey, requestid.From(r.Context())),
		slog.Any(LogKeyError, err))
	WriteError(w, logger, http.StatusInternalServerError, CodeInternal, MsgInternal)
}

func Decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var value T
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf(wrapFormat, ErrInvalidBody, err)
	}
	if decoder.More() {
		return value, ErrInvalidBody
	}
	return value, nil
}
