package db

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// The demo account lets reviewers sign in immediately and see a finished order with keys,
// even right after the free server restarted (which resets the database).
// It's created on every startup, like the rest of the sample data.
const (
	DemoEmail    = "demo@gamekey.test" // ".test" is a reserved domain: it can never be a real inbox
	DemoPassword = "gamekey-demo"
)

// demoOrderSlugs: the products in the demo account's finished order (1 key each).
var demoOrderSlugs = []string{"elden-ring-pc-steam-global", "steam-gift-card-20-eur-steam-europe"}

// seedDemoAccount creates the demo user and one PAID order, taking real keys from stock
// exactly like a purchase would (keys marked sold and linked to the order lines).
func seedDemoAccount(tx *sql.Tx) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(DemoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := tx.Exec(`INSERT INTO users (email, password_hash, display_name) VALUES (?, ?, 'Demo Reviewer')`,
		DemoEmail, string(hash))
	if err != nil {
		return fmt.Errorf("insert demo user: %w", err)
	}
	userID, _ := res.LastInsertId()

	// Paid "yesterday", with the success test card.
	res, err = tx.Exec(`INSERT INTO orders (user_id, status, total_cents, created_at, paid_at, card_last4)
		VALUES (?, 'paid', 0, datetime('now', '-1 day'), datetime('now', '-1 day'), '4242')`, userID)
	if err != nil {
		return fmt.Errorf("insert demo order: %w", err)
	}
	orderID, _ := res.LastInsertId()

	total := 0
	for _, slug := range demoOrderSlugs {
		// The cheapest offer that has a key available.
		var offerID int64
		var price int
		var title string
		err := tx.QueryRow(`
			SELECT o.id, o.price_cents, p.title
			FROM offers o JOIN products p ON p.id = o.product_id
			WHERE p.slug = ?
			  AND EXISTS (SELECT 1 FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available')
			ORDER BY o.price_cents LIMIT 1`, slug).Scan(&offerID, &price, &title)
		if err != nil {
			return fmt.Errorf("demo offer for %s: %w", slug, err)
		}

		res, err := tx.Exec(`INSERT INTO order_items (order_id, offer_id, quantity, unit_price_cents, product_title)
			VALUES (?, ?, 1, ?, ?)`, orderID, offerID, price, title)
		if err != nil {
			return fmt.Errorf("insert demo order item: %w", err)
		}
		itemID, _ := res.LastInsertId()

		if _, err := tx.Exec(`UPDATE game_keys SET status = 'sold', order_item_id = ?
			WHERE id = (SELECT id FROM game_keys WHERE offer_id = ? AND status = 'available' ORDER BY id LIMIT 1)`,
			itemID, offerID); err != nil {
			return fmt.Errorf("claim demo key: %w", err)
		}
		total += price
	}

	_, err = tx.Exec(`UPDATE orders SET total_cents = ? WHERE id = ?`, total, orderID)
	return err
}
