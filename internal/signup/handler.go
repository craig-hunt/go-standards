package signup

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
	mux.HandleFunc(RouteCreate, h.create)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	request, err := httpjson.Decode[Request](w, r)
	if err != nil {
		httpjson.WriteInvalidBody(w, h.logger)
		return
	}
	signup, problems := Validate(request)
	if problems != nil {
		httpjson.WriteFieldErrors(w, h.logger, problems)
		return
	}
	id, err := h.store.Save(r.Context(), signup)
	if err != nil {
		httpjson.WriteInternal(w, r, h.logger, err)
		return
	}
	httpjson.Write(w, h.logger, http.StatusCreated, Confirmation{ID: id, Summary: signup.Summary()})
}
