package auth

import "database/sql"

type AuthHandler struct {
	DB *sql.DB
}
