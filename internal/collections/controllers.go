package collections

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/forPelevin/gomoji"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (c *CollectionsHandler) CreateCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var fieldErrors []httpx.FieldError

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
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Message: "'name' is required"})
		} else if len(name) > 255 {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Message: "'name' can't contain more than 255 characters"})
		}
		_, err = gomoji.GetInfo(emoji)
		if err != nil {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "emoji", Message: "'emoji' is not a valid emoji"})
		}

		if collection.CollectionFields != nil {
			for _, field := range *collection.CollectionFields {
				label := strings.TrimSpace(field.Label)
				fType := strings.TrimSpace(strings.ToLower(field.Type))

				if label == "" {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "label", Message: "'label' is required"})
				} else if len(label) > 255 {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "label", Message: "'label' can't contain more than 255 characters"})
				}
				if fType == "text" && fType != "number" {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "type", Message: "'type' must be either 'text' or 'number'"})
				}
			}
		}

		if len(fieldErrors) != 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, "Bad request", fieldErrors)
			return
		}

		// Begin a transaction
		tx, err := c.DB.Begin(r.Context())
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error creating transaction:", err)
			return
		}
		defer func() {
			if err := tx.Rollback(r.Context()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				log.Println("error rolling back transaction:", err)
			}
		}()

		collectionQuery :=
			`INSERT into collections (user_id, name, emoji) VALUES ($1, $2, $3) RETURNING id`

		var collectionID string

		err = tx.QueryRow(r.Context(), collectionQuery, userID, name, emoji).Scan(&collectionID)
		if err != nil {
			log.Println("Error inserting into collections:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		if collection.CollectionFields != nil && len(*collection.CollectionFields) > 0 {
			rows := [][]any{}
			for _, c := range *collection.CollectionFields {
				key := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(c.Label)), " ", "_")
				label := strings.TrimSpace(c.Label)
				fType := strings.TrimSpace(strings.ToLower(c.Type))

				rows = append(rows, []any{collectionID, key, label, fType})
			}

			_, err = tx.CopyFrom(
				r.Context(),
				pgx.Identifier{"collection_fields"},               // Table name
				[]string{"collection_id", "key", "label", "type"}, // Target columns
				pgx.CopyFromRows(rows),                            // Data source
			)

			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					httpx.WriteJSON(w, http.StatusConflict, "Conflict", []httpx.FieldError{{Field: "key", Message: "'key' already exists in this collection"}})
					return
				}
				log.Println("Error copying into collection_fields:", err)
				httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
				return
			}
		}

		// Commit if nothing wrong
		err = tx.Commit(r.Context())
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error commiting transaction:", err)
			return
		}

		httpx.WriteJSON(w, http.StatusCreated, "Collection created", collectionID)
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
