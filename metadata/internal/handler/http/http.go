package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"movieexample.com/metadata/internal/controller/metadata"
	"movieexample.com/metadata/pkg/model"
)

// Handler defines a movie metadata HTTP handler.
type Handler struct {
	ctrl *metadata.Controller
}

// New creates a new movie metadata HTTP Handler.
func New(ctrl *metadata.Controller) *Handler {
	return &Handler{ctrl: ctrl}
}

// GetMetadata handles GET /metadata requests.
func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	m, err := h.ctrl.Get(ctx, id)
	if err != nil && errors.Is(err, metadata.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Repository get error for movie %s: %v\n", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(m); err != nil {
		log.Printf("Response encode error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// PutMetadata handles PUT /metadata requests.
func (h *Handler) PutMetadata(w http.ResponseWriter, r *http.Request) {
	var metadata *model.Metadata
	err := json.NewDecoder(r.Body).Decode(&metadata)
	if err != nil || metadata == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = h.ctrl.Put(r.Context(), metadata)
	if err != nil {
		log.Printf("Put metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
