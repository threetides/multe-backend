package collections

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/forPelevin/gomoji"
	"github.com/jackc/pgx/v5"
)

func (c *CollectionsHandler) CreateCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errors []httpx.FieldError

		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Println("Error getting userID from context")
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		var collection Collection

		err := json.NewDecoder(r.Body).Decode(&collection)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, "Invalid json", nil)
			return
		}

		name := strings.TrimSpace(collection.Name)
		emoji := strings.TrimSpace(collection.Emoji)

		if name == "" {
			errors = append(errors, httpx.FieldError{Field: "name", Message: "'name' is required"})
		} else if len(name) > 255 {
			errors = append(errors, httpx.FieldError{Field: "name", Message: "'name' can't contain more than 255 characters"})
		}
		_, err = gomoji.GetInfo(emoji)
		if err != nil {
			errors = append(errors, httpx.FieldError{Field: "emoji", Message: "'emoji' is not a valid emoji"})
		}

		if len(errors) != 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "Bad request", errors)
			return
		}

		query :=
			`INSERT into collections (user_id, name, emoji) VALUES ($1, $2, $3)`

		_, err = c.DB.Exec(r.Context(), query, userID, name, emoji)
		if err != nil {
			log.Println("Error inserting into collections:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "Collection created", nil)
	}
}

func (c *CollectionsHandler) GetAllCollections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Println("Error getting userID from context")
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		query :=
			`SELECT id, user_id, name, emoji, created_at, updated_at FROM collections WHERE user_id = $1`

		rows, err := c.DB.Query(r.Context(), query, userID)
		if err != nil {
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collections, err := pgx.CollectRows(rows, pgx.RowToStructByName[Collection])
		if err != nil {
			log.Println("Error scanning into collection:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, "Collections found", collections)
	}
}
