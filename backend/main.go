// main.go starts the API server: read config, open the database, connect routes, listen.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Rati0ed/eneba-marketplace/backend/db"
	"github.com/Rati0ed/eneba-marketplace/backend/handlers"
	"github.com/Rati0ed/eneba-marketplace/backend/middleware"
)

func main() {
	// Configuration comes from environment variables, with defaults that work locally.
	// On Render we set them in the dashboard. Render sets PORT by itself.
	port := getEnv("PORT", "8080")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "http://localhost:5173") // Vite's dev server
	dbPath := getEnv("DB_PATH", "data.db")
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		// Not used until Day 5 (login). A missing secret in production would be a security hole,
		// so we warn loudly now and will refuse to start once auth exists.
		log.Println("WARNING: JWT_SECRET is not set")
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err) // Fatalf = print and exit, the server can't run without a DB
	}
	defer database.Close()
	log.Printf("database ready at %s (fresh copy, seeded)", dbPath)

	h := &handlers.Handler{DB: database}

	// The router ("mux") maps "METHOD /path" to a handler function.
	// Since Go 1.22 the standard library supports methods and {wildcards} in patterns.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/products", h.ListProducts)
	mux.HandleFunc("GET /api/covers/{file}", h.Cover)

	// Wrap the router in middleware: Logging runs first, then CORS, then the route's handler.
	handler := middleware.Logging(middleware.CORS(allowedOrigin, mux))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
		// Timeouts protect us from slow or stuck clients holding connections forever.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on :%s, allowing frontend origin(s): %s", port, allowedOrigin)
	log.Fatal(server.ListenAndServe()) // blocks forever, unless the server crashes
}

// getEnv returns the environment variable `key`, or `fallback` if it is empty.
// Python equivalent: os.environ.get(key) or fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
