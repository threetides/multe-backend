package collections

import (
	"database/sql"
	"time"
)

type CollectionsHandler struct {
	DB *sql.DB
}

type Collection struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userID"`
	Name      string    `json:"name"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"createdAt"`
}
