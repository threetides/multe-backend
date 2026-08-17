package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"

	"golang.org/x/oauth2"
)

func GoogleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		returnOrigin := r.Header.Get("Referer")
		url := AppConfig.AuthCodeURL("randomstate|" + returnOrigin)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func (Auth *AuthHandler) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		state := queryParams.Get("state")
		nonce, returnOrigin, _ := strings.Cut(state, "|")
		if nonce != "randomstate" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		code := queryParams.Get("code")

		provider, err := oidc.NewProvider(r.Context(), "https://accounts.google.com")
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error creating oidc provider:", err)
			return
		}

		// Configure an OpenID Connect aware OAuth2 client.
		oauth2Config := oauth2.Config{
			ClientID:     AppConfig.ClientID,
			ClientSecret: AppConfig.ClientSecret,
			RedirectURL:  AppConfig.RedirectURL,

			// Discovery returns the OAuth2 endpoints.
			Endpoint: provider.Endpoint(),

			// "openid" is a required scope for OpenID Connect flows.
			Scopes: []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
		}

		// Create an ID Token verifier.
		idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: AppConfig.ClientID})

		oauth2Token, err := oauth2Config.Exchange(r.Context(), code)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error exchanging token:", err)
			return
		}

		// Extract the ID Token from OAuth2 token.
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error extracting raw id_token:", err)
			return
		}

		// Parse and verify ID Token payload.
		idToken, err := idTokenVerifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error parsing and verifying id token payload:", err)
			return
		}

		// Extract custom claims
		var claims struct {
			Sub           string `json:"sub"`
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
			Name          string `json:"name"`
			Picture       string `json:"picture"`
		}
		if err := idToken.Claims(&claims); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error extracting custom claims:", err)
			return
		}

		b := make([]byte, 32)
		rand.Read(b)
		token := base64.RawURLEncoding.EncodeToString(b)
		hash := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(hash[:])
		expiresAt := time.Now().AddDate(0, 0, 30)

		// Begin a transaction
		tx, err := Auth.DB.Begin(r.Context())
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

		// Insert into users table
		usersQuery :=
			`INSERT INTO users (name, email, email_verified) VALUES ($1, $2, $3)
			ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
			RETURNING id`

		var userID string

		err = tx.QueryRow(r.Context(), usersQuery, claims.Name, claims.Email, claims.EmailVerified).Scan(&userID)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error inserting claims into users:", err)
			return
		}

		// Insert into oauth_accounts table
		oauthAccountsQuery :=
			`INSERT INTO oauth_accounts (user_id, provider, provider_user_id) VALUES ($1, $2, $3)
			ON CONFLICT (provider, provider_user_id) DO UPDATE SET provider_user_id = EXCLUDED.provider_user_id`

		_, err = tx.Exec(r.Context(), oauthAccountsQuery, userID, "google", claims.Sub)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error inserting claims into oauth_accounts:", err)
			return
		}

		// Insert into sessions table
		sessionsQuery :=
			`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`

		_, err = tx.Exec(r.Context(), sessionsQuery, userID, tokenHash, expiresAt)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error inserting claims into sessions:", err)
			return
		}

		// Commit if nothing wrong
		err = tx.Commit(r.Context())
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error commiting transaction:", err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token, // raw token to browser
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Expires:  expiresAt,
		})

		http.Redirect(w, r, returnOrigin, http.StatusFound)
	}
}

func (Auth *AuthHandler) GetSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		hash := sha256.Sum256([]byte(cookie.Value))
		tokenHash := hex.EncodeToString(hash[:])

		userIDQuery :=
			`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`

		var userID string

		err = Auth.DB.QueryRow(r.Context(), userIDQuery, tokenHash).Scan(&userID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Println("Error quering database:", err)
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		if userID == "" {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		var user User

		userQuery :=
			`SELECT id, email, name, email_verified, created_at FROM users WHERE id = $1`

		err = Auth.DB.QueryRow(r.Context(), userQuery, userID).Scan(&user.ID, &user.Email, &user.Name, &user.EmailVerified, &user.CreatedAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Println("Error quering database:", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}

		httpx.WriteJSON(w, http.StatusOK, "Authorized", user)
	}
}

func (Auth *AuthHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			httpx.WriteJSON(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		hash := sha256.Sum256([]byte(cookie.Value))
		tokenHash := hex.EncodeToString(hash[:])

		query :=
			`DELETE from sessions WHERE token_hash = $1`

		_, err = Auth.DB.Exec(r.Context(), query, tokenHash)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, "Internal server error", nil)
			log.Println("error deleting session:", err)
			return
		}

		// Clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,              // Tells modern browsers to delete it immediately
			Expires:  time.Unix(0, 0), // Compatibility fallback for older browsers (e.g., old IE)
			HttpOnly: true,
		})

		httpx.WriteJSON(w, http.StatusOK, "Logged out", nil)
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
