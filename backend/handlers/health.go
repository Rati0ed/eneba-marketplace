package handlers

import "net/http"

// Health answers GET /api/health. Render calls it to check that the server is alive.
// It also pings the database, so "ok" really means "ready to serve requests".
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.PingContext(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
