// Package health answers liveness and readiness probes. Liveness reports that
// the process serves requests; readiness also confirms its backing database, so
// a platform stops routing traffic to an instance that cannot do its work.
package health

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/craig-hunt/go-standards/internal/httpjson"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Report struct {
	Status string `json:"status"`
}

func Register(mux *http.ServeMux, pinger Pinger, logger *slog.Logger) {
	mux.HandleFunc(RouteLive, func(w http.ResponseWriter, r *http.Request) {
		httpjson.Write(w, r, logger, http.StatusOK, Report{Status: StatusOK})
	})
	mux.HandleFunc(RouteReady, func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), ReadyTimeout)
		defer cancel()
		if err := pinger.Ping(ctx); err != nil {
			logger.LogAttrs(ctx, slog.LevelWarn, MsgNotReady, slog.Any(httpjson.LogKeyError, err))
			httpjson.Write(w, r, logger, http.StatusServiceUnavailable, Report{Status: StatusUnavailable})
			return
		}
		httpjson.Write(w, r, logger, http.StatusOK, Report{Status: StatusOK})
	})
}
