package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rati0ed/eneba-marketplace/backend/search"
)

// suggestLimit = how many products the header dropdown shows.
const suggestLimit = 6

// BuildSearchIndex loads every product title from the database into the in-memory search
// index. Called once at startup (main.go). The catalog never changes while the server runs
// (it's re-seeded on start), so the index never goes out of date.
func BuildSearchIndex(database *sql.DB) (*search.Index, error) {
	rows, err := database.Query(`SELECT id, title, sales_count FROM products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []search.Document
	for rows.Next() {
		var d search.Document
		if err := rows.Scan(&d.ID, &d.Title, &d.Popularity); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return search.NewIndex(docs), nil
}

// SearchSuggest answers GET /api/search/suggest?q=... for the header dropdown:
// the best 6 matches, how many matched in total, and a "did you mean" product if none did.
func (h *Handler) SearchSuggest(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 100 {
		writeError(w, http.StatusBadRequest, "search text is too long (max 100 characters)")
		return
	}

	response := map[string]any{"items": []ProductCard{}, "total": 0, "suggestion": nil}
	if query == "" {
		writeJSON(w, http.StatusOK, response)
		return
	}

	results := h.Search.Search(query, 0)
	if len(results) == 0 {
		suggestion, err := h.closestCard(r.Context(), query)
		if err != nil {
			serverError(w, err)
			return
		}
		response["suggestion"] = suggestion
		writeJSON(w, http.StatusOK, response)
		return
	}

	top := results
	if len(top) > suggestLimit {
		top = top[:suggestLimit]
	}
	whereSQL, whereArgs := idsWhere(top)
	orderSQL, orderArgs := relevanceOrder(top)
	items, err := h.queryCards(r.Context(), whereSQL, append(whereArgs, orderArgs...), orderSQL, suggestLimit, 0)
	if err != nil {
		serverError(w, err)
		return
	}

	response["items"] = items
	response["total"] = len(results)
	writeJSON(w, http.StatusOK, response)
}

// closestCard returns the product that the query most likely meant, or nil.
func (h *Handler) closestCard(ctx context.Context, query string) (*ProductCard, error) {
	id, ok := h.Search.Closest(query)
	if !ok {
		return nil, nil
	}
	cards, err := h.queryCards(ctx, "WHERE id = ?", []any{id}, "id", 1, 0)
	if err != nil || len(cards) == 0 {
		return nil, err
	}
	return &cards[0], nil
}

// idsWhere: [{ID:7}, {ID:3}] -> "WHERE id IN (?, ?)", [7, 3]
func idsWhere(results []search.Result) (string, []any) {
	args := make([]any, len(results))
	for i, r := range results {
		args[i] = r.ID
	}
	return "WHERE id IN (" + placeholders(len(results)) + ")", args
}

// relevanceOrder sorts SQL rows in the same order as the search results:
//
//	[{ID:7}, {ID:3}] -> "CASE id WHEN ? THEN 0 WHEN ? THEN 1 END", [7, 3]
//
// SQL doesn't know our search scores, so we tell it each product's position.
func relevanceOrder(results []search.Result) (string, []any) {
	var b strings.Builder
	args := make([]any, len(results))
	b.WriteString("CASE id")
	for i, r := range results {
		b.WriteString(" WHEN ? THEN ")
		b.WriteString(strconv.Itoa(i)) // the position is our own number, not user input
		args[i] = r.ID
	}
	b.WriteString(" END")
	return b.String(), args
}
