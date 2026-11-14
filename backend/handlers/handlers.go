// Package handlers contains the functions that answer HTTP requests.
// One file per area: health.go, products.go, covers.go, and later auth.go, orders.go, home.go.
package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/Rati0ed/eneba-marketplace/backend/search"
)

// Handler holds what every handler needs: the database and the search index.
// Handlers are methods on it, so they can use h.DB and h.Search without global variables.
// Python comparison: like a class with self.db, where each endpoint is a method.
type Handler struct {
	DB     *sql.DB
	Search *search.Index // in-memory typo-tolerant search, built at startup
}

// writeJSON sends any Go value as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json: %v", err)
	}
}

// writeError sends our standard error shape: {"error": "message"}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// serverError logs the real error for us, but sends the user only a generic message.
// Internal details (SQL, file paths) must never leak to the client.
func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "something went wrong")
}
