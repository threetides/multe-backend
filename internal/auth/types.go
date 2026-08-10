package auth

import (
	"database/sql"
	"time"
)

type AuthHandler struct {
	DB *sql.DB
}

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
}
