package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/JonasHaugen/multe_backend/internal/collections"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/JonasHaugen/multe_backend/internal/items"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/threetides/tideauth"
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

	// Create config for tideauth
	cfg := tideauth.Config{
		DB:           db,
		CookieDomain: os.Getenv("CORS_ORIGIN"),
		Google: tideauth.Google{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("REDIRECT_URL") + "/api/auth/google/callback",
		},
	}

	auth := tideauth.New(cfg)

	// Run migrations
	err = auth.Migrate(context.Background())
	if err != nil {
		log.Fatalln("Error running migrations:", err)
	}

	// Create default routes exported by tideauth
	routes, err := auth.Routes()
	if err != nil {
		log.Fatalln("Error setting up routes:", err)
	}

	collectionsHandler := collections.CollectionsHandler{DB: db, Auth: auth}
	itemsHandler := items.ItemsHandler{DB: db, Auth: auth}

	// Handlers
	mux.Handle("/api/auth/", http.StripPrefix("/api/auth", routes))

	mux.HandleFunc("POST /api/collection", auth.Protected(collectionsHandler.CreateCollection()))
	mux.HandleFunc("GET /api/collections", auth.Protected(collectionsHandler.GetAllCollections()))
	mux.HandleFunc("GET /api/collection/{collectionID}", auth.Protected(collectionsHandler.GetCollectionByID()))

	mux.HandleFunc("POST /api/item", auth.Protected(itemsHandler.CreateItem()))
	mux.HandleFunc("GET /api/items", auth.Protected(itemsHandler.GetAllItems()))
	mux.HandleFunc("GET /api/items/{collectionID}", auth.Protected(itemsHandler.GetItemsFromCollection()))

	fmt.Println("Server started on port", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%v", os.Getenv("PORT")), handler))
}
