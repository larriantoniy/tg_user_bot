package http

import (
	"net/http"
)

// NewRouter настраивает маршруты HTTP-службы.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/prediction/getAll", h.GetAll)
	return mux
}
