package collections

import (
	"time"

	"github.com/JonasHaugen/multe_backend/internal/items"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/threetides/tideauth"
)

type CollectionsHandler struct {
	DB   *pgxpool.Pool
	Auth *tideauth.Auth
}

type CollectionField struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collectionID"`
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	Type         string    `json:"type"`
	Unit         string    `json:"unit"`
	Position     int       `json:"position"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Collection struct {
	ID               string            `json:"id"`
	UserID           string            `json:"userID"`
	Name             string            `json:"name"`
	Emoji            string            `json:"emoji"`
	CollectionFields []CollectionField `json:"collectionFields" db:"-"`
	Items            []items.Item      `json:"items" db:"-"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}
