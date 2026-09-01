package collections

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/JonasHaugen/multe_backend/internal/items"
	"github.com/forPelevin/gomoji"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (c *CollectionsHandler) CreateCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var fieldErrors []httpx.FieldError

		userID, ok := c.Auth.UserIDFromContext(r.Context())
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
			for _, field := range collection.CollectionFields {
				label := strings.TrimSpace(field.Label)
				fType := strings.TrimSpace(strings.ToLower(field.Type))
				unit := strings.TrimSpace(field.Unit)

				if label == "" {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "label", Message: "'label' is required"})
				} else if len(label) > 255 {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "label", Message: "'label' can't contain more than 255 characters"})
				}
				if fType != "text" && fType != "number" {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "type", Message: "'type' must be either 'text' or 'number'"})
				}
				if len(unit) > 255 {
					fieldErrors = append(fieldErrors, httpx.FieldError{Field: "unit", Message: "'unit' can't contain more than 255 characters"})
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

		if len(collection.CollectionFields) > 0 {
			rows := [][]any{}
			for _, c := range collection.CollectionFields {
				key := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(c.Label)), " ", "_")
				label := strings.TrimSpace(c.Label)
				fType := strings.TrimSpace(strings.ToLower(c.Type))
				unit := strings.TrimSpace(c.Unit)
				position := c.Position

				rows = append(rows, []any{collectionID, key, label, fType, unit, position})
			}

			_, err = tx.CopyFrom(
				r.Context(),
				pgx.Identifier{"collection_fields"}, // Table name
				[]string{"collection_id", "key", "label", "type", "unit", "position"}, // Target columns
				pgx.CopyFromRows(rows), // Data source
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
		userID, ok := c.Auth.UserIDFromContext(r.Context())
		if !ok {
			log.Println("Error getting userID from context")
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
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
			`SELECT id, user_id, name, emoji, created_at, updated_at FROM collections WHERE user_id = $1`

		collectionRows, err := tx.Query(r.Context(), collectionQuery, userID)
		if err != nil {
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collections, err := pgx.CollectRows(collectionRows, pgx.RowToStructByName[Collection])
		if err != nil {
			log.Println("Error scanning into collection:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		var ids []string

		for _, c := range collections {
			ids = append(ids, c.ID)
		}

		collectionFieldQuery :=
			`SELECT id, collection_id, key, label, type, unit, position, created_at, updated_at FROM collection_fields WHERE collection_id = ANY($1) ORDER BY position`

		collectionFieldRows, err := tx.Query(r.Context(), collectionFieldQuery, ids)
		if err != nil {
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collectionFields, err := pgx.CollectRows(collectionFieldRows, pgx.RowToStructByName[CollectionField])
		if err != nil {
			log.Println("Error scanning into collection:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		itemsQuery :=
			`SELECT id, collection_id, name, status, price, date_bought, location, custom, created_at, updated_at FROM items`

		itemRows, err := tx.Query(r.Context(), itemsQuery)
		if err != nil {
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		items, err := pgx.CollectRows(itemRows, pgx.RowToStructByName[items.Item])
		if err != nil {
			log.Println("Error scanning into items:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		for i := range collections {
			for _, cf := range collectionFields {
				if cf.CollectionID == collections[i].ID {
					collections[i].CollectionFields = append(collections[i].CollectionFields, cf)
				}
			}
			for _, item := range items {
				if item.CollectionID == collections[i].ID {
					collections[i].Items = append(collections[i].Items, item)
				}
			}
		}

		// Commit if nothing wrong
		err = tx.Commit(r.Context())
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error commiting transaction:", err)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, "Collections found", collections)
	}
}

func (c *CollectionsHandler) GetCollectionByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := c.Auth.UserIDFromContext(r.Context())
		if !ok {
			log.Println("Error getting userID from context")
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collectionID := r.PathValue("collectionID")

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
			`SELECT id, user_id, name, emoji, created_at, updated_at FROM collections WHERE user_id = $1 AND id = $2`

		var collection Collection

		err = tx.QueryRow(r.Context(), collectionQuery, userID, collectionID).Scan(&collection.ID, &collection.UserID, &collection.Name, &collection.Emoji, &collection.CreatedAt, &collection.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				log.Println("Error doing query:", err)
				httpx.WriteJSON(w, http.StatusNotFound, "Collection not found", nil)
				return
			}
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collectionFieldQuery :=
			`SELECT id, collection_id, key, label, type, unit, position, created_at, updated_at FROM collection_fields WHERE collection_id = $1 ORDER BY position`

		collectionFieldRows, err := tx.Query(r.Context(), collectionFieldQuery, collectionID)
		if err != nil {
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collectionFields, err := pgx.CollectRows(collectionFieldRows, pgx.RowToStructByName[CollectionField])
		if err != nil {
			log.Println("Error scanning into collectionFields:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collection.CollectionFields = collectionFields

		itemsQuery :=
			`SELECT id, collection_id, name, status, price, date_bought, location, custom, created_at, updated_at FROM items WHERE collection_id = $1`

		itemRows, err := tx.Query(r.Context(), itemsQuery, collectionID)
		if err != nil {
			log.Println("Error doing query:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		items, err := pgx.CollectRows(itemRows, pgx.RowToStructByName[items.Item])
		if err != nil {
			log.Println("Error scanning into items:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		collection.Items = items

		// Commit if nothing wrong
		err = tx.Commit(r.Context())
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error commiting transaction:", err)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, "Collections found", collection)
	}
}
