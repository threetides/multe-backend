package auth

import (
	"time"

	"github.com/jackc/pgx/v5"
)

type AuthHandler struct {
	DB *pgx.Conn
}

type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
}
