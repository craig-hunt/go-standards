package task

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/craig-hunt/go-standards/internal/httpjson"
)

type Handler struct {
	store  Store
	logger *slog.Logger
}

type ClearResult struct {
	Removed int64 `json:"removed"`
}

type createRequest struct {
	Title string `json:"title"`
}

// updateRequest holds a pointer so an omitted field reads as missing rather
// than as false.
type updateRequest struct {
	Completed *bool `json:"completed"`
}

func NewHandler(store Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc(RouteList, h.list)
	mux.HandleFunc(RouteCreate, h.create)
	mux.HandleFunc(RouteUpdate, h.update)
	mux.HandleFunc(RouteDelete, h.remove)
	mux.HandleFunc(RouteClearCompleted, h.clearCompleted)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	filter, err := ParseFilter(r.URL.Query().Get(QueryFilter))
	if err != nil {
		httpjson.WriteError(w, h.logger, http.StatusBadRequest, CodeInvalidFilter, MsgInvalidFilter)
		return
	}
	all, err := h.store.List(r.Context())
	if err != nil {
		httpjson.WriteInternal(w, r, h.logger, err)
		return
	}
	httpjson.Write(w, h.logger, http.StatusOK, Summarize(all, filter))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	body, err := httpjson.Decode[createRequest](w, r)
	if err != nil {
		httpjson.WriteInvalidBody(w, h.logger)
		return
	}
	title, err := NewTitle(body.Title)
	if err != nil {
		httpjson.WriteFieldErrors(w, h.logger, map[string]string{FieldTitle: titleProblem(err)})
		return
	}
	created, err := h.store.Create(r.Context(), title)
	if err != nil {
		httpjson.WriteInternal(w, r, h.logger, err)
		return
	}
	httpjson.Write(w, h.logger, http.StatusCreated, created)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	body, err := httpjson.Decode[updateRequest](w, r)
	if err != nil {
		httpjson.WriteInvalidBody(w, h.logger)
		return
	}
	if body.Completed == nil {
		httpjson.WriteFieldErrors(w, h.logger, map[string]string{FieldCompleted: MsgCompletedRequired})
		return
	}
	updated, err := h.store.SetCompleted(r.Context(), id, *body.Completed)
	if err != nil {
		h.writeStoreError(w, r, err)
		return
	}
	httpjson.Write(w, h.logger, http.StatusOK, updated)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) clearCompleted(w http.ResponseWriter, r *http.Request) {
	removed, err := h.store.DeleteCompleted(r.Context())
	if err != nil {
		httpjson.WriteInternal(w, r, h.logger, err)
		return
	}
	httpjson.Write(w, h.logger, http.StatusOK, ClearResult{Removed: removed})
}

// titleProblem turns a validation error into the sentence a form shows. Go
// error text stays lowercase and unpunctuated; the user-facing copy lives apart.
func titleProblem(err error) string {
	if errors.Is(err, ErrTitleTooLong) {
		return MsgTitleTooLong
	}
	return MsgTitleRequired
}

func (h *Handler) pathID(w http.ResponseWriter, r *http.Request) (ID, bool) {
	id, err := ParseID(r.PathValue(PathParamID))
	if err != nil {
		httpjson.WriteError(w, h.logger, http.StatusBadRequest, CodeInvalidID, MsgInvalidID)
		return 0, false
	}
	return id, true
}

func (h *Handler) writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		httpjson.WriteError(w, h.logger, http.StatusNotFound, CodeNotFound, MsgNotFound)
		return
	}
	httpjson.WriteInternal(w, r, h.logger, err)
}
