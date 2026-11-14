package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

const (
	maxCartLines    = 20 // different offers in one cart
	maxLineQuantity = 10 // keys per offer; the frontend uses the same limit
)

// CartItem is what the browser sends: which offer and how many. Never a price.
// Even if a tampered request adds "price_cents": 1, the JSON decoder ignores fields
// that aren't in this struct, so the server never sees it.
type CartItem struct {
	OfferID  int64 `json:"offer_id"`
	Quantity int   `json:"quantity"`
}

// QuoteLine is one cart line priced by the server.
type QuoteLine struct {
	OfferID        int64  `json:"offer_id"`
	Quantity       int    `json:"quantity"`
	ProductSlug    string `json:"product_slug"`
	ProductTitle   string `json:"product_title"`
	ImageURL       string `json:"image_url"`
	Platform       string `json:"platform"`
	Region         string `json:"region"`
	SellerName     string `json:"seller_name"`
	UnitPriceCents int    `json:"unit_price_cents"`
	Stock          int    `json:"stock"`
	LineTotalCents int    `json:"line_total_cents"`
	// Why this line can't be bought as it is: "" (fine), "not_found",
	// "out_of_stock" or "not_enough_stock".
	Problem string `json:"problem"`
}

// Quote is a whole priced cart.
type Quote struct {
	Items       []QuoteLine `json:"items"`
	TotalCents  int         `json:"total_cents"`  // sum of the lines without problems
	CanCheckout bool        `json:"can_checkout"` // at least one line, and no problems
}

// querier is anything that can run a single-row query: both *sql.DB and *sql.Tx can.
// This lets priceCart run on its own (cart page) or inside the checkout transaction.
// (An "interface" in Go = a list of methods; any type that has them fits, automatically.)
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// validateCartItems checks the shape of a cart before we touch the database.
func validateCartItems(items []CartItem) error {
	if len(items) > maxCartLines {
		return fmt.Errorf("a cart can have at most %d different offers", maxCartLines)
	}
	seen := map[int64]bool{}
	for _, item := range items {
		if item.OfferID <= 0 {
			return errors.New("every cart line needs a valid offer_id")
		}
		if item.Quantity < 1 || item.Quantity > maxLineQuantity {
			return fmt.Errorf("quantity must be between 1 and %d", maxLineQuantity)
		}
		if seen[item.OfferID] {
			return fmt.Errorf("offer %d appears twice in the cart", item.OfferID)
		}
		seen[item.OfferID] = true
	}
	return nil
}

// priceCart looks up the CURRENT price and stock of every line in the database.
// This is the only place prices come from: the cart page, order creation and payment
// all use database prices, never anything the browser sent.
func priceCart(ctx context.Context, q querier, items []CartItem) (Quote, error) {
	quote := Quote{Items: make([]QuoteLine, 0, len(items))}

	for _, item := range items {
		line := QuoteLine{OfferID: item.OfferID, Quantity: item.Quantity}
		err := q.QueryRowContext(ctx, `
			SELECT o.price_cents, p.slug, p.title, p.image_url, p.platform, p.region, s.name,
			       (SELECT COUNT(*) FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available')
			FROM offers o
			JOIN products p ON p.id = o.product_id
			JOIN sellers s ON s.id = o.seller_id
			WHERE o.id = ?`, item.OfferID,
		).Scan(&line.UnitPriceCents, &line.ProductSlug, &line.ProductTitle, &line.ImageURL,
			&line.Platform, &line.Region, &line.SellerName, &line.Stock)

		switch {
		case errors.Is(err, sql.ErrNoRows):
			line.Problem = "not_found" // e.g. an old cart after the database was re-seeded
		case err != nil:
			return Quote{}, err
		case line.Stock == 0:
			line.Problem = "out_of_stock"
		case item.Quantity > line.Stock:
			line.Problem = "not_enough_stock"
		}

		line.LineTotalCents = line.UnitPriceCents * line.Quantity
		if line.Problem == "" {
			quote.TotalCents += line.LineTotalCents
		}
		quote.Items = append(quote.Items, line)
	}

	quote.CanCheckout = len(quote.Items) > 0
	for _, line := range quote.Items {
		if line.Problem != "" {
			quote.CanCheckout = false
		}
	}
	return quote, nil
}

// CartQuote answers POST /api/cart/quote {"items": [{"offer_id": 12, "quantity": 2}]}.
// Public (no login needed): guests can see their cart too.
func (h *Handler) CartQuote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []CartItem `json:"items"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if err := validateCartItems(body.Items); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	quote, err := priceCart(r.Context(), h.DB, body.Items)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, quote)
}
