// Package middleware contains code that runs around EVERY request, before and after the handler.
//
// A middleware is a function that takes a handler and returns a new handler that
// does something extra and then calls the original one. Like a Python decorator:
//
//	@log_requests          ->   handler = Logging(CORS(mux))
//	def handler(...): ...
package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// CORS lets the browser call our API from the frontend's domain.
//
// Browsers block JavaScript on site A (our Vercel app) from reading responses from
// site B (our Render API) unless B replies with Access-Control-Allow-Origin: A.
// allowedOrigins is a comma-separated list, e.g.
// "http://localhost:5173,https://eneba-marketplace.vercel.app".
func CORS(allowedOrigins string, next http.Handler) http.Handler {
	// Build a set of allowed origins once at startup (map used as a set, like Python's set()).
	allowed := map[string]bool{}
	for _, origin := range strings.Split(allowedOrigins, ",") {
		// Trim spaces and a trailing slash, a common copy-paste mistake.
		allowed[strings.TrimRight(strings.TrimSpace(origin), "/")] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowed[origin] {
			// Echo back only the ONE origin that asked, and only if it's on our list.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600") // browser may cache this answer 10 min
		}
		// Tells caches that the response depends on the Origin header.
		w.Header().Add("Vary", "Origin")

		// "Preflight": before a POST with JSON or an Authorization header, the browser first
		// sends OPTIONS to ask "am I allowed?". We answer with the headers above and no body.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Logging prints one line per request: method, path, status code and how long it took.
// Example: "GET /api/products 200 3.2ms". These lines show up in Render's log viewer.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// The standard ResponseWriter doesn't let us read the status code afterwards,
		// so we wrap it in our own type that remembers it.
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.status, time.Since(start).Round(100*time.Microsecond))
	})
}

// statusRecorder is a ResponseWriter that also stores the status code.
// Embedding http.ResponseWriter means it gets all the original methods for free;
// we only replace WriteHeader.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
