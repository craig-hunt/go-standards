package apitest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/craig-hunt/go-standards/internal/expect"
)

const (
	samplePath   = "/samples"
	sampleName   = "Dana Whitfield"
	headerName   = "X-Sample"
	headerValue  = "present"
	notJSON      = "plain text"
	noBodyReason = "a request without a body arrives empty"
)

type sample struct {
	Name string `json:"name"`
}

type echo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Header string `json:"header"`
	Body   string `json:"body"`
}

type recorder struct {
	fatals int
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(string, ...any) {}

func (r *recorder) Fatalf(string, ...any) { r.fatals++ }

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(echo{Method: r.Method, Path: r.URL.Path, Header: r.Header.Get(headerName), Body: string(body)})
	})
}

func TestDoSendsTheMethodTargetHeadersAndEncodedBody(t *testing.T) {
	response := Do(t, echoHandler(), Request{
		Method:  http.MethodPost,
		Target:  samplePath,
		Body:    sample{Name: sampleName},
		Headers: map[string]string{headerName: headerValue},
	})

	got := Decode[echo](t, response)
	var sent sample
	expect.NoError(t, json.Unmarshal([]byte(got.Body), &sent))
	expect.Equal(t, got.Method, http.MethodPost)
	expect.Equal(t, got.Path, samplePath)
	expect.Equal(t, got.Header, headerValue)
	expect.Equal(t, sent, sample{Name: sampleName})
}

func TestDoSendsNoBodyWhenNoneIsGiven(t *testing.T) {
	got := Decode[echo](t, Do(t, echoHandler(), Request{Method: http.MethodGet, Target: samplePath}))

	expect.True(t, got.Body == "", noBodyReason)
}

func TestDoStopsTheTestWhenTheBodyCannotEncode(t *testing.T) {
	reporter := &recorder{}

	response := Do(reporter, echoHandler(), Request{Method: http.MethodPost, Target: samplePath, Body: make(chan int)})

	expect.Equal(t, reporter.fatals, 1)
	expect.Equal(t, response, (*httptest.ResponseRecorder)(nil))
}

func TestDecodeStopsTheTestWhenTheResponseIsNotJSON(t *testing.T) {
	reporter := &recorder{}
	response := httptest.NewRecorder()
	_, err := response.WriteString(notJSON)
	expect.NoError(t, err)

	Decode[sample](reporter, response)

	expect.Equal(t, reporter.fatals, 1)
}
