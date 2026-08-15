package items

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/google/uuid"
)

func (i *ItemsHandler) CreateItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errors []httpx.FieldError

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
		name := strings.TrimSpace(item.Name)
		status := "owned"
		price := item.Price
		dateBought := item.DateBought
		location := strings.TrimSpace(item.Location)
		custom := item.Custom

		if err := uuid.Validate(collectionID); err != nil {
			errors = append(errors, httpx.FieldError{Field: "collectionID", Message: "'collectionID' is not valid uuid"})
		}
		if name == "" {
			errors = append(errors, httpx.FieldError{Field: "name", Message: "'name' is required"})
		} else if len(name) > 255 {
			errors = append(errors, httpx.FieldError{Field: "name", Message: "'name' can't contain more than 255 characters"})
		}
		if strings.TrimSpace(item.Status) != "" && (strings.TrimSpace(item.Status) != "owned" && strings.TrimSpace(item.Status) != "wish") {
			errors = append(errors, httpx.FieldError{Field: "status", Message: "'status' must be either 'owned' or 'wish'"})
		}
		if price != nil && *price < 0 {
			errors = append(errors, httpx.FieldError{Field: "price", Message: "'price' must be a positive number"})
		}
		if item.DateBought != nil {
			if _, err = time.Parse("2006-01-02", *dateBought); err != nil {
				errors = append(errors, httpx.FieldError{Field: "dateBought", Message: "'dateBought' must be follow this structure: 2006-01-02"})
			}
		}
		if len(location) > 255 {
			errors = append(errors, httpx.FieldError{Field: "location", Message: "'location' can't contain more than 255 characters"})
		}

		if strings.TrimSpace(item.Status) == "wish" {
			status = "wish"
		}

		if len(custom) == 0 {
			custom = []byte(`{}`)
		}

		if len(errors) != 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "Bad request", errors)
			return
		}

		collectionQuery :=
			`SELECT user_id FROM collections WHERE id = ($1) AND user_id = ($2)`

		var collectionUserID string

		_ = i.DB.QueryRowContext(r.Context(), collectionQuery, collectionID, userID).Scan(&collectionUserID)

		if collectionUserID == "" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		itemQuery :=
			`INSERT into items (collection_id, name, status, price, date_bought, location, custom)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, collection_id, name, status, price, date_bought, location, custom, created_at`

		var inserted Item

		err = i.DB.QueryRowContext(r.Context(), itemQuery, collectionID, name, status, price, dateBought, location, custom).Scan(&inserted.ID, &inserted.CollectionID, &inserted.Name, &inserted.Status, &inserted.Price, &inserted.DateBought, &inserted.Location, &inserted.Custom, &inserted.CreatedAt)

		if err != nil {
			log.Println("Error inserting into items table:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "Item created", inserted)
	}
}
