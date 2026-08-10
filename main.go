package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/JonasHaugen/multe_backend/internal/auth"
	"github.com/JonasHaugen/multe_backend/internal/httpx"
	"github.com/JonasHaugen/multe_backend/internal/test"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	// Connect to db
	db, err := sql.Open("postgres", os.Getenv("DB_CONNECTION_STRING"))

	if err != nil {
		log.Fatalln("Error connecting to db:", err)
	}

	defer func() {
		if err = db.Close(); err != nil {
			log.Fatalln("Error closing db connection:", err)
		}
	}()

	err = db.Ping()

	if err != nil {
		log.Fatalln("Error pinging db:", err)
	}

	fmt.Println("Connected to db")

	auth.GoogleConfig()

	mux := http.NewServeMux()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{os.Getenv("CORS_ORIGIN")},
		AllowCredentials: true,
	})

	// Insert the middleware
	handler := c.Handler(mux)

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, "Healthy", nil)
	})

	authWithDB := auth.AuthHandler{DB: db}

	// Handlers
	mux.HandleFunc("GET /api/auth/google/login", auth.GoogleLogin())
	mux.HandleFunc("GET /api/auth/google/callback", authWithDB.GoogleCallback())
	mux.HandleFunc("GET /api/auth/session", authWithDB.GetSession())
	mux.HandleFunc("POST /api/auth/logout", authWithDB.Logout())

	mux.HandleFunc("GET /api/test", authWithDB.Protected(test.Test()))

	fmt.Println("Server started on port", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%v", os.Getenv("PORT")), handler))
}
