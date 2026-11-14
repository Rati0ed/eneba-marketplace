package handlers

import (
	"database/sql"
	"errors"
	"net/http"
)

// ProductDetail = everything on a ProductCard plus the long text fields.
// Embedding ProductCard (no field name) copies its fields in, and the JSON stays flat:
// {"id":..., "title":..., "description":..., "release_date":...}
type ProductDetail struct {
	ProductCard
	Description string `json:"description"`
	ReleaseDate string `json:"release_date"`
}

// Offer = one seller's price for a product, with how many keys are left.
type Offer struct {
	ID                  int64  `json:"id"`
	SellerName          string `json:"seller_name"`
	SellerRatingPercent int    `json:"seller_rating_percent"`
	SellerSalesCount    int    `json:"seller_sales_count"`
	PriceCents          int    `json:"price_cents"`
	Stock               int    `json:"stock"` // available keys; 0 = sold out
}

// GetProduct answers GET /api/products/{slug}.
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	var p ProductDetail
	err := h.DB.QueryRowContext(r.Context(), cardsCTE+`
		SELECT c.id, c.slug, c.title, c.platform, c.region, c.genre, c.image_url, c.msrp_cents,
		       c.lowest_price_cents, c.offer_count, p.description, p.release_date
		FROM cards c
		JOIN products p ON p.id = c.id
		WHERE c.slug = ?`, r.PathValue("slug"),
	).Scan(&p.ID, &p.Slug, &p.Title, &p.Platform, &p.Region, &p.Genre, &p.ImageURL, &p.MSRPCents,
		&p.LowestPriceCents, &p.OfferCount, &p.Description, &p.ReleaseDate)

	// QueryRow + Scan returns sql.ErrNoRows when nothing matched: that's a 404, not a server error.
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	p.DiscountPercent = discountPercent(p.MSRPCents, p.LowestPriceCents)
	writeJSON(w, http.StatusOK, p)
}

// GetProductOffers answers GET /api/products/{slug}/offers: every seller's offer,
// in-stock ones first, then cheapest first, then best-rated seller first.
// Sold-out offers are included (the page shows them disabled), they're just last.
func (h *Handler) GetProductOffers(w http.ResponseWriter, r *http.Request) {
	var productID int64
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id FROM products WHERE slug = ?`, r.PathValue("slug")).Scan(&productID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	// The (SELECT COUNT(*) ...) inside the SELECT is a "correlated sub-query":
	// it runs once per offer row and counts that offer's available keys.
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT o.id, s.name, s.rating_percent, s.sales_count, o.price_cents,
		       (SELECT COUNT(*) FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available') AS stock
		FROM offers o
		JOIN sellers s ON s.id = o.seller_id
		WHERE o.product_id = ?
		ORDER BY stock = 0, o.price_cents, s.rating_percent DESC`, productID)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()

	offers := make([]Offer, 0)
	for rows.Next() {
		var o Offer
		if err := rows.Scan(&o.ID, &o.SellerName, &o.SellerRatingPercent, &o.SellerSalesCount, &o.PriceCents, &o.Stock); err != nil {
			serverError(w, err)
			return
		}
		offers = append(offers, o)
	}
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": offers})
}
