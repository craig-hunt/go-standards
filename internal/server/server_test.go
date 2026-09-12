package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/craig-hunt/go-standards/internal/expect"
	"github.com/craig-hunt/go-standards/internal/logcapture"
)

const (
	loopbackAddress  = "127.0.0.1:0"
	samplePort       = 8080
	sampleAddress    = ":8080"
	urlFormat        = "http://%s/"
	generousShutdown = 5 * time.Second
	briefShutdown    = 50 * time.Millisecond
	listenerReason   = "the run reports the closed listener"
	shutdownReason   = "the run reports the shutdown deadline"
)

func listen(t *testing.T) net.Listener {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), Network, loopbackAddress)
	expect.NoError(t, err)
	return listener
}

func get(ctx context.Context, listener net.Listener) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(urlFormat, listener.Addr()), nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(request)
}

func TestAddressListensOnEveryInterfaceAtThePort(t *testing.T) {
	expect.Equal(t, Address(samplePort), sampleAddress)
}

func TestNewAppliesTheServerTimeouts(t *testing.T) {
	srv := New(http.NotFoundHandler())

	expect.Equal(t, srv.ReadHeaderTimeout, ReadHeaderTimeout)
	expect.Equal(t, srv.ReadTimeout, ReadTimeout)
	expect.Equal(t, srv.WriteTimeout, WriteTimeout)
	expect.Equal(t, srv.IdleTimeout, IdleTimeout)
}

func TestRunServesRequestsUntilTheContextEndsThenStopsCleanly(t *testing.T) {
	logger, capture := logcapture.New()
	listener := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, New(handler), listener, generousShutdown, logger)
	}()

	response, err := get(context.Background(), listener)
	expect.NoError(t, err)
	expect.NoError(t, response.Body.Close())
	cancel()

	expect.NoError(t, <-done)
	expect.Equal(t, response.StatusCode, http.StatusNoContent)
	expect.Equal(t, capture.Find(t, MsgListening)[LogKeyAddress], any(listener.Addr().String()))
	expect.Equal(t, capture.Find(t, MsgShuttingDown)[slog.LevelKey], any(slog.LevelInfo.String()))
}

func TestRunReturnsTheServeErrorWhenTheListenerFails(t *testing.T) {
	logger, _ := logcapture.New()
	listener := listen(t)
	expect.NoError(t, listener.Close())

	err := Run(context.Background(), New(http.NotFoundHandler()), listener, generousShutdown, logger)

	expect.True(t, errors.Is(err, net.ErrClosed), listenerReason)
}

func TestRunReportsAShutdownThatOutlastsTheTimeout(t *testing.T) {
	logger, _ := logcapture.New()
	listener := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, New(slow), listener, briefShutdown, logger)
	}()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		if response, err := get(context.Background(), listener); err == nil {
			_ = response.Body.Close()
		}
	}()

	<-started
	cancel()
	err := <-done
	close(release)
	<-requestDone

	expect.True(t, errors.Is(err, context.DeadlineExceeded), shutdownReason)
}
