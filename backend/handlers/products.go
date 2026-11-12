package handlers

import "net/http"

// ProductCard is one product as shown in a list (store page, homepage rows).
// The `json:"..."` tags set the key names in the JSON output.
type ProductCard struct {
	ID       int64  `json:"id"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Platform string `json:"platform"`
	Region   string `json:"region"`
	Genre    string `json:"genre"`
	ImageURL string `json:"image_url"`
	// MSRP = official price; the frontend uses it to show "-35%".
	MSRPCents int `json:"msrp_cents"`
	// Cheapest price among offers that still have keys. It's a pointer (*int)
	// because it can be missing: if every offer is sold out, SQL returns NULL,
	// and a nil pointer becomes `null` in JSON. A plain int can't be "nothing".
	LowestPriceCents *int `json:"lowest_price_cents"`
	OfferCount       int  `json:"offer_count"` // offers that are in stock
}

// ListProducts answers GET /api/products.
// Day 1: every product with its lowest price. Filters, sorting and pages come on Day 2.
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	// How the query works:
	//   - LEFT JOIN keeps products even if no offer matches (they get NULL price).
	//   - The join only accepts offers that have at least one available key (EXISTS).
	//   - GROUP BY p.id squeezes all offers of a product into one row, and
	//     MIN()/COUNT() summarize them: cheapest price and number of offers.
	const query = `
		SELECT p.id, p.slug, p.title, p.platform, p.region, p.genre, p.image_url, p.msrp_cents,
		       MIN(o.price_cents) AS lowest_price_cents,
		       COUNT(o.id)        AS offer_count
		FROM products p
		LEFT JOIN offers o
		       ON o.product_id = p.id
		      AND EXISTS (SELECT 1 FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available')
		GROUP BY p.id
		ORDER BY p.sales_count DESC`

	rows, err := h.DB.QueryContext(r.Context(), query)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close() // always release the rows when the function ends

	// make([]T, 0) instead of `var items []T`, so an empty result is `[]` in JSON, not `null`.
	items := make([]ProductCard, 0)
	for rows.Next() {
		var p ProductCard
		// Scan copies the columns of the current row into our variables, in order.
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Platform, &p.Region, &p.Genre,
			&p.ImageURL, &p.MSRPCents, &p.LowestPriceCents, &p.OfferCount); err != nil {
			serverError(w, err)
			return
		}
		items = append(items, p)
	}
	// rows.Err reports an error that stopped the loop early (e.g. the DB connection dropped).
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}

	// Wrapped in an object ({"items": [...]}) instead of a bare list, so on Day 2
	// we can add "total" and "page" without breaking the frontend.
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
