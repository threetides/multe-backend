package items

import (
	"database/sql"
	"encoding/json"
	"time"
)

type ItemsHandler struct {
	DB *sql.DB
}

type Item struct {
	ID           string          `json:"id"`
	CollectionID string          `json:"collectionID"`
	Name         string          `json:"name"`
	Status       string          `json:"status"`
	Price        *float64        `json:"price"`
	DateBought   *string         `json:"dateBought"`
	Location     string          `json:"location"`
	Custom       json.RawMessage `json:"custom"`
	CreatedAt    time.Time       `json:"createdAt"`
}
