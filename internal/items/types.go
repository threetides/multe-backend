package items

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemsHandler struct {
	DB *pgxpool.Pool
}

type Item struct {
	ID           string          `json:"id"`
	CollectionID string          `json:"collectionID"`
	Name         string          `json:"name"`
	Status       string          `json:"status"`
	Price        *float64        `json:"price"`
	DateBought   *time.Time      `json:"dateBought"`
	Location     string          `json:"location"`
	Custom       json.RawMessage `json:"custom"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}
