package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/collections"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/JonasHaugen/multe_backend/internal/items"
	"github.com/JonasHaugen/multe_backend/internal/test"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	// Connect to db
	db, err := pgxpool.New(context.Background(), os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		log.Fatalln("Error connecting to db:", err)
	}

	defer db.Close()

	err = db.Ping(context.Background())

	if err != nil {
		log.Fatalln("Error pinging db:", err)
	}

	fmt.Println("Connected to db")

	auth.GoogleConfig()

	mux := http.NewServeMux()

	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(os.Getenv("CORS_ORIGIN"), ","),
		AllowCredentials: true,
	})

	// Insert the middleware
	handler := c.Handler(mux)

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Healthy", nil)
	})

	authWithDB := auth.AuthHandler{DB: db}
	collectionsWithDB := collections.CollectionsHandler{DB: db}
	itemsWithDB := items.ItemsHandler{DB: db}

	// Handlers
	mux.HandleFunc("GET /api/auth/google/login", auth.GoogleLogin())
	mux.HandleFunc("GET /api/auth/google/callback", authWithDB.GoogleCallback())
	mux.HandleFunc("GET /api/auth/session", authWithDB.GetSession())
	mux.HandleFunc("POST /api/auth/logout", authWithDB.Logout())

	mux.HandleFunc("GET /api/test", authWithDB.Protected(test.Test()))

	mux.HandleFunc("POST /api/collection", authWithDB.Protected(collectionsWithDB.CreateCollection()))
	mux.HandleFunc("GET /api/collections", authWithDB.Protected(collectionsWithDB.GetAllCollections()))
	mux.HandleFunc("GET /api/collection/{collectionID}", authWithDB.Protected(collectionsWithDB.GetCollectionByID()))

	mux.HandleFunc("POST /api/item", authWithDB.Protected(itemsWithDB.CreateItem()))
	mux.HandleFunc("GET /api/items", authWithDB.Protected(itemsWithDB.GetAllItems()))
	mux.HandleFunc("GET /api/items/{collectionID}", authWithDB.Protected(itemsWithDB.GetItemsFromCollection()))

	fmt.Println("Server started on port", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%v", os.Getenv("PORT")), handler))
}
