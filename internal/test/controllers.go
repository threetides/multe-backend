package test

import (
	"net/http"

	"github.com/JonasHaugen/multe_backend/internal/httpx"
)

func Test() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		httpx.WriteJSON(w, http.StatusOK, "Test funker", nil)
	}
}
