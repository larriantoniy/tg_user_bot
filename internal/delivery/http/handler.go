package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/larriantoniy/tg_user_bot/internal/useCases"
)

// Handler — сущность HTTP-обработчиков.
type Handler struct {
	prediction *useCases.PredictionService
}

// NewHandler создаёт новый Handler с внедрённой бизнес-логикой.
func NewHandler(prediction *useCases.PredictionService) *Handler {
	return &Handler{prediction: prediction}
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {

	data, err := h.prediction.GetAll()
	if err != nil {
		fmt.Printf("Get All err %s", err)
		http.Error(w, "Failed to fetch predictions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Encoding error", http.StatusInternalServerError)
	}
}
