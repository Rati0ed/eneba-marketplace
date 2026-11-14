// main.go starts the API server: read config, open the database, connect routes, listen.
package main

import (
	"crypto/rand"
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
	jwtSecret := []byte(getEnv("JWT_SECRET", ""))
	if len(jwtSecret) == 0 {
		// No secret configured (local development): make a random one for this run.
		// Tokens then stop working after a restart, which is fine: the database is re-created
		// on restart anyway, so the accounts they belong to are gone too.
		jwtSecret = make([]byte, 32)
		if _, err := rand.Read(jwtSecret); err != nil {
			log.Fatalf("generate JWT secret: %v", err)
		}
		log.Println("WARNING: JWT_SECRET is not set; using a random secret for this run")
	} else if len(jwtSecret) < 32 {
		// HS256 keys should be at least 256 bits (32 bytes); a short secret can be brute-forced.
		log.Println("WARNING: JWT_SECRET is shorter than 32 characters; use a long random value")
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err) // Fatalf = print and exit, the server can't run without a DB
	}
	defer database.Close()
	log.Printf("database ready at %s (fresh copy, seeded)", dbPath)

	searchIndex, err := handlers.BuildSearchIndex(database)
	if err != nil {
		log.Fatalf("search index: %v", err)
	}

	h := &handlers.Handler{DB: database, Search: searchIndex, JWTSecret: jwtSecret}

	// All URL -> handler mappings are in handlers/routes.go.
	mux := h.Routes()

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
