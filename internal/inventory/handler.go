package inventory

import (
	"log/slog"
	"net/http"

	"github.com/craig-hunt/go-standards/internal/httpjson"
)

type Handler struct {
	store  Store
	logger *slog.Logger
}

func NewHandler(store Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc(RouteList, h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query, err := ParseQuery(r.URL.Query())
	if err != nil {
		httpjson.WriteError(w, r, h.logger, http.StatusBadRequest, CodeInvalidQuery, err.Error())
		return
	}
	items, err := h.store.Items(r.Context())
	if err != nil {
		httpjson.WriteInternal(w, r, h.logger, err)
		return
	}
	httpjson.Write(w, r, h.logger, http.StatusOK, Apply(items, query))
}
