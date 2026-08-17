package collectionfields

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func (cf *CollectionFieldsHandler) CreateCollectionField() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errors []httpx.FieldError

		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Println("Error getting userID from context")
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		var field CollectionField

		err := json.NewDecoder(r.Body).Decode(&field)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "Invalid json", nil)
			return
		}

		collectionID := strings.TrimSpace(field.CollectionID)
		key := strings.TrimSpace(field.Key)
		label := strings.TrimSpace(field.Label)
		fieldType := strings.TrimSpace(field.Type)
		position := field.Position

		if err := uuid.Validate(collectionID); err != nil {
			errors = append(errors, httpx.FieldError{Field: "collectionID", Message: "'collectionID' is not valid uuid"})
		}
		if key == "" {
			errors = append(errors, httpx.FieldError{Field: "key", Message: "'key' is required"})
		} else if len(key) > 255 {
			errors = append(errors, httpx.FieldError{Field: "key", Message: "'key' can't contain more than 255 characters"})
		}
		if label == "" {
			errors = append(errors, httpx.FieldError{Field: "label", Message: "'label' is required"})
		} else if len(label) > 255 {
			errors = append(errors, httpx.FieldError{Field: "label", Message: "'label' can't contain more than 255 characters"})
		}
		if fieldType == "" {
			errors = append(errors, httpx.FieldError{Field: "type", Message: "'type' is required"})
		} else if len(fieldType) > 255 {
			errors = append(errors, httpx.FieldError{Field: "type", Message: "'type' can't contain more than 255 characters"})
		}
		if position != nil && *position < 0 {
			errors = append(errors, httpx.FieldError{Field: "position", Message: "'position' must be a positive number"})
		}

		if len(errors) != 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "Bad request", errors)
			return
		}

		collectionQuery :=
			`SELECT user_id FROM collections WHERE id = ($1) AND user_id = ($2)`

		var collectionUserID string

		_ = cf.DB.QueryRow(r.Context(), collectionQuery, collectionID, userID).Scan(&collectionUserID)

		if collectionUserID == "" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		fieldQuery :=
			`INSERT into collection_fields (collection_id, key, label, type, position)
			VALUES ($1, $2, $3, $4, $5) RETURNING id, collection_id, key, label, type, position, created_at`

		var inserted CollectionField

		err = cf.DB.QueryRow(r.Context(), fieldQuery, collectionID, key, label, fieldType, position).Scan(&inserted.ID, &inserted.CollectionID, &inserted.Key, &inserted.Label, &inserted.Type, &inserted.Position, &inserted.CreatedAt)

		if err != nil {
			// 23505 = unique_violation on UNIQUE (collection_id, key)
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				httpx.WriteJSON(w, http.StatusConflict, "Conflict", []httpx.FieldError{{Field: "key", Message: "'key' already exists in this collection"}})
				return
			}
			log.Println("Error inserting into collection_fields table:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "Collection field created", inserted)
	}
}
