package handlers

import (
	"context"
	"strings"
)

// ProductCard is one product as shown in a list (store page, search dropdown, homepage rows).
// The `json:"..."` tags set the key names in the JSON output.
type ProductCard struct {
	ID       int64  `json:"id"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Platform string `json:"platform"`
	Region   string `json:"region"`
	Genre    string `json:"genre"`
	ImageURL string `json:"image_url"`
	// MSRP = official price, shown crossed out next to our price.
	MSRPCents int `json:"msrp_cents"`
	// Cheapest price among offers that still have keys. It's a pointer (*int)
	// because it can be missing: if every offer is sold out, SQL returns NULL,
	// and a nil pointer becomes `null` in JSON. A plain int can't be "nothing".
	LowestPriceCents *int `json:"lowest_price_cents"`
	// How much cheaper than MSRP, in whole percent (0 if not cheaper or sold out).
	// Calculated here so every page shows the same number.
	DiscountPercent int `json:"discount_percent"`
	OfferCount      int `json:"offer_count"` // offers that are in stock
}

// cardsCTE is a "Common Table Expression": a named sub-query we can use like a table.
// `WITH cards AS (...)` builds one row per product with its lowest in-stock price,
// and the queries that follow can filter and sort `cards` by that price.
//
// How the inner query works:
//   - LEFT JOIN keeps products even if no offer matches (they get NULL price).
//   - The join only accepts offers that have at least one available key (EXISTS).
//   - GROUP BY p.id squeezes all offers of a product into one row, and
//     MIN()/COUNT() summarize them: cheapest price and number of offers.
const cardsCTE = `
	WITH cards AS (
		SELECT p.id, p.slug, p.title, p.platform, p.region, p.genre, p.image_url,
		       p.msrp_cents, p.release_date, p.sales_count,
		       MIN(o.price_cents) AS lowest_price_cents,
		       COUNT(o.id)        AS offer_count
		FROM products p
		LEFT JOIN offers o
		       ON o.product_id = p.id
		      AND EXISTS (SELECT 1 FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available')
		GROUP BY p.id
	)`

// queryCards runs `SELECT ... FROM cards <whereSQL> ORDER BY <orderBy> LIMIT ? OFFSET ?`.
//
// whereSQL and orderBy are pieces of SQL text. They must NEVER contain user input directly:
// user values go into `args` and are referenced with `?` placeholders.
// orderBy only ever comes from our own fixed list (see sortOptions in products.go).
func (h *Handler) queryCards(ctx context.Context, whereSQL string, args []any, orderBy string, limit, offset int) ([]ProductCard, error) {
	query := cardsCTE + `
		SELECT id, slug, title, platform, region, genre, image_url, msrp_cents, lowest_price_cents, offer_count
		FROM cards ` + whereSQL + `
		ORDER BY ` + orderBy + `
		LIMIT ? OFFSET ?`

	// append(args, limit, offset) would modify the caller's slice, so we copy first.
	allArgs := append(append([]any{}, args...), limit, offset)

	rows, err := h.DB.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // always release the rows when the function ends

	// make([]T, 0) instead of `var items []T`, so an empty result is `[]` in JSON, not `null`.
	items := make([]ProductCard, 0)
	for rows.Next() {
		var p ProductCard
		// Scan copies the columns of the current row into our variables, in order.
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Platform, &p.Region, &p.Genre,
			&p.ImageURL, &p.MSRPCents, &p.LowestPriceCents, &p.OfferCount); err != nil {
			return nil, err
		}
		p.DiscountPercent = discountPercent(p.MSRPCents, p.LowestPriceCents)
		items = append(items, p)
	}
	// rows.Err reports an error that stopped the loop early (e.g. the DB connection dropped).
	return items, rows.Err()
}

// countCards returns how many cards match whereSQL (used for "total" and page count).
func (h *Handler) countCards(ctx context.Context, whereSQL string, args []any) (int, error) {
	var total int
	err := h.DB.QueryRowContext(ctx, cardsCTE+` SELECT COUNT(*) FROM cards `+whereSQL, args...).Scan(&total)
	return total, err
}

// discountPercent: MSRP 5999, price 3299 -> 45. Integer math only, rounded to nearest.
func discountPercent(msrpCents int, priceCents *int) int {
	if priceCents == nil || *priceCents >= msrpCents || msrpCents <= 0 {
		return 0
	}
	saved := msrpCents - *priceCents
	return (saved*100 + msrpCents/2) / msrpCents // "+ msrp/2" rounds instead of cutting off
}

// placeholders(3) -> "?, ?, ?". Used for SQL `IN (...)` lists, one ? per value.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
