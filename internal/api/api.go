// Package api assembles the service's HTTP surface: health probes open to the
// platform, every /api route behind the bearer token, and the shared middleware
// around both.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/craig-hunt/go-standards/internal/health"
	"github.com/craig-hunt/go-standards/internal/inventory"
	"github.com/craig-hunt/go-standards/internal/middleware"
	"github.com/craig-hunt/go-standards/internal/signup"
	"github.com/craig-hunt/go-standards/internal/task"
)

const PathPrefix = "/api/"

type Dependencies struct {
	Tasks     task.Store
	Signups   signup.Store
	Inventory inventory.Store
	Pinger    health.Pinger
	APIToken  string
	Logger    *slog.Logger
	Now       func() time.Time
}

func NewHandler(d Dependencies) http.Handler {
	routes := http.NewServeMux()
	task.NewHandler(d.Tasks, d.Logger).Register(routes)
	signup.NewHandler(d.Signups, d.Logger).Register(routes)
	inventory.NewHandler(d.Inventory, d.Logger).Register(routes)

	root := http.NewServeMux()
	health.Register(root, d.Pinger, d.Logger)
	root.Handle(PathPrefix, middleware.RequireBearerToken(d.APIToken, d.Logger)(routes))

	return middleware.Chain(root,
		middleware.RequestID,
		middleware.Logging(d.Logger, d.Now),
		middleware.Recover(d.Logger),
	)
}
