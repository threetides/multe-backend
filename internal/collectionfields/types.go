package collectionfields

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CollectionFieldsHandler struct {
	DB *pgxpool.Pool
}

type CollectionField struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collectionID"`
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	Type         string    `json:"type"`
	Position     *int      `json:"position"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
