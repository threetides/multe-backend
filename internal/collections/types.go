package collections

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CollectionsHandler struct {
	DB *pgxpool.Pool
}

type CollectionField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Unit     string `json:"unit"`
	Position int    `json:"position"`
}

type Collection struct {
	ID               string             `json:"id"`
	UserID           string             `json:"userID"`
	Name             string             `json:"name"`
	Emoji            string             `json:"emoji"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	CollectionFields *[]CollectionField `json:"collectionFields" db:"-"`
}
