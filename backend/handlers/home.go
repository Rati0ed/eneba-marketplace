package handlers

import (
	"net/http"
	"strconv"
)

// The homepage is DATA-DRIVEN: the frontend first asks which blocks to show (GET /api/home/blocks),
// then each block calls its own endpoint for its content. Changing the homepage_blocks table
// (order, titles, turning a block off) changes the homepage without touching frontend code.

// HomeBlock is one section of the homepage.
type HomeBlock struct {
	ID       int64   `json:"id"`
	Type     string  `json:"type"`     // "banner" or "product_row": tells the frontend which component to draw
	Title    string  `json:"title"`    // e.g. "Top deals"
	Endpoint string  `json:"endpoint"` // where this block gets its content, e.g. "/api/home/deals"
	LinkURL  *string `json:"link_url"` // "Show all" link, or null
}

// Banner is one slide of the homepage carousel.
type Banner struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	ImageURL string `json:"image_url"`
	LinkURL  string `json:"link_url"`
}

// HomeBlocks answers GET /api/home/blocks: the active blocks, in display order.
func (h *Handler) HomeBlocks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, type, title, endpoint, link_url
		FROM homepage_blocks
		WHERE is_active = 1
		ORDER BY position, id`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()

	blocks := make([]HomeBlock, 0)
	for rows.Next() {
		var b HomeBlock
		// link_url can be NULL; scanning into a *string gives nil for NULL.
		if err := rows.Scan(&b.ID, &b.Type, &b.Title, &b.Endpoint, &b.LinkURL); err != nil {
			serverError(w, err)
			return
		}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": blocks})
}

// HomeBanners answers GET /api/home/banners: the carousel slides.
func (h *Handler) HomeBanners(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, title, subtitle, image_url, link_url FROM banners ORDER BY position, id`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()

	banners := make([]Banner, 0)
	for rows.Next() {
		var b Banner
		if err := rows.Scan(&b.ID, &b.Title, &b.Subtitle, &b.ImageURL, &b.LinkURL); err != nil {
			serverError(w, err)
			return
		}
		banners = append(banners, b)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": banners})
}

// HomeBestSellers answers GET /api/home/best-sellers?limit=12: most sold products in stock.
func (h *Handler) HomeBestSellers(w http.ResponseWriter, r *http.Request) {
	h.writeCardRow(w, r, "WHERE lowest_price_cents IS NOT NULL", sortOptions["popular"])
}

// HomeDeals answers GET /api/home/deals?limit=12: biggest discounts vs. MSRP, in stock.
func (h *Handler) HomeDeals(w http.ResponseWriter, r *http.Request) {
	h.writeCardRow(w, r, "WHERE lowest_price_cents IS NOT NULL AND lowest_price_cents < msrp_cents", sortOptions["discount"])
}

// writeCardRow is the shared code of the product-row endpoints.
func (h *Handler) writeCardRow(w http.ResponseWriter, r *http.Request, whereSQL, orderBy string) {
	limit := 12
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 24 {
			writeError(w, http.StatusBadRequest, "limit must be a whole number from 1 to 24")
			return
		}
		limit = n
	}

	items, err := h.queryCards(r.Context(), whereSQL, nil, orderBy, limit, 0)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
