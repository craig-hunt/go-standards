package health

import (
	"net/http"
	"time"
)

const (
	PathLive          = "/health"
	PathReady         = "/health/ready"
	RouteLive         = http.MethodGet + " " + PathLive
	RouteReady        = http.MethodGet + " " + PathReady
	StatusOK          = "ok"
	StatusUnavailable = "unavailable"
	ReadyTimeout      = 2 * time.Second
	MsgNotReady       = "readiness check failed"
)
