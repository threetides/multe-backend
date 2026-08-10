package test

import (
	"net/http"

	"github.com/JonasHaugen/multe_backend/internal/httpx"
)

func Test() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Test funker", nil)
	}
}
