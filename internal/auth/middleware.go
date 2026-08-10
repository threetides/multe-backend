package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/JonasHaugen/multe_backend/internal/httpx"
)

func (Auth *AuthHandler) Protected(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			log.Println("Error getting cookie:", err)
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		hash := sha256.Sum256([]byte(cookie.Value))
		tokenHash := hex.EncodeToString(hash[:])
		expiresAt := time.Now().AddDate(0, 0, 30)

		// Begin a transaction
		tx, err := Auth.DB.BeginTx(r.Context(), nil)

		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error creating transaction:", err)
			return
		}

		defer func() {
			if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
				log.Println("error rolling back transaction:", err)
			}
		}()

		userIDQuery :=
			`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`

		var userID string

		err = Auth.DB.QueryRowContext(r.Context(), userIDQuery, tokenHash).Scan(&userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Println("Error quering database:", err)
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		if userID == "" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		// Shifting expires_at
		expiresAtQuery :=
			`UPDATE sessions SET expires_at = $1 WHERE token_hash = $2;`

		_, err = Auth.DB.ExecContext(r.Context(), expiresAtQuery, expiresAt, tokenHash)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error shifting expires_at:", err)
			return
		}

		// Commit if nothing wrong
		err = tx.Commit()
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error commiting transaction:", err)
			return
		}

		next(w, r)
	}
}
