package items

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
)

func (i *ItemsHandler) CreateItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Println("Error getting userID from context")
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		var item Item

		err := json.NewDecoder(r.Body).Decode(&item)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "Invalid json", nil)
			return
		}

		collectionID := strings.TrimSpace(item.CollectionID)

		collectionQuery :=
			`SELECT user_id FROM collections WHERE id = ($1) AND user_id = ($2)`

		var collectionUserID string

		_ = i.DB.QueryRowContext(r.Context(), collectionQuery, collectionID, userID).Scan(&collectionUserID)

		if collectionUserID == "" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "Item created", nil)
	}
}
