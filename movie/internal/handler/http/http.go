package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"movieexample.com/movie/internal/controller/movie"
	"movieexample.com/movie/internal/gateway"
)

// Handler defines a movie handler.
type Handler struct {
	ctrl *movie.Controller
}

// New creates a new movie HTTP handler.
func New(ctrl *movie.Controller) *Handler {
	return &Handler{ctrl}
}

// GetMovieDetails handles GET /movie requests.
func (h *Handler) GetMovieDetails(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	movieDetails, err := h.ctrl.Get(r.Context(), id)
	if err != nil && errors.Is(err, gateway.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(movieDetails); err != nil {
		log.Printf("Response encode error: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
