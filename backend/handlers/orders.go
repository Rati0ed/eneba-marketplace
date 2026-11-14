package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Checkout happens in two steps:
//
//  1. POST /api/orders           -> a "pending" order. Prices are looked up in the database
//                                   and COPIED into the order (the browser never sends prices).
//  2. POST /api/orders/{id}/pay  -> simulated card payment. On success, ONE database transaction
//                                   claims the keys, links them to the order and marks it paid.
//
// Two steps let a declined card be retried on the same order, and keep each step simple.

// OrderItem is one line of an order. Keys are filled in once the order is paid.
type OrderItem struct {
	ID             int64    `json:"id"`
	OfferID        int64    `json:"offer_id"`
	ProductTitle   string   `json:"product_title"`
	ProductSlug    string   `json:"product_slug"`
	ImageURL       string   `json:"image_url"`
	Platform       string   `json:"platform"`
	Quantity       int      `json:"quantity"`
	UnitPriceCents int      `json:"unit_price_cents"` // the price at the moment of purchase
	Keys           []string `json:"keys"`
}

// Order is a whole order with its lines.
type Order struct {
	ID         int64       `json:"id"`
	Status     string      `json:"status"` // pending, paid, failed
	TotalCents int         `json:"total_cents"`
	CreatedAt  string      `json:"created_at"`
	PaidAt     *string     `json:"paid_at"`    // null until paid
	CardLast4  *string     `json:"card_last4"` // null until paid
	Items      []OrderItem `json:"items"`
}

// dbRunner is what both *sql.DB and *sql.Tx can do, so loadOrder works in and outside a transaction.
type dbRunner interface {
	querier
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ---------- 1. Create an order ----------

// CreateOrder answers POST /api/orders {"items": [{"offer_id": 12, "quantity": 1}]}.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var body struct {
		Items []CartItem `json:"items"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if len(body.Items) == 0 {
		writeError(w, http.StatusBadRequest, "Your cart is empty.")
		return
	}
	if err := validateCartItems(body.Items); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback() // undoes everything unless we reach tx.Commit()

	// Price every line from the DATABASE. Whatever prices the browser showed (or a tampered
	// request claims) don't matter: only offer ids and quantities are read from the request.
	quote, err := priceCart(r.Context(), tx, body.Items)
	if err != nil {
		serverError(w, err)
		return
	}
	if !quote.CanCheckout {
		writeError(w, http.StatusConflict, cartProblemMessage(quote))
		return
	}

	res, err := tx.ExecContext(r.Context(),
		`INSERT INTO orders (user_id, status, total_cents) VALUES (?, 'pending', ?)`, user.ID, quote.TotalCents)
	if err != nil {
		serverError(w, err)
		return
	}
	orderID, _ := res.LastInsertId()

	for _, line := range quote.Items {
		// The price and title are copied, so the order keeps them even if the seller changes them later.
		if _, err := tx.ExecContext(r.Context(),
			`INSERT INTO order_items (order_id, offer_id, quantity, unit_price_cents, product_title) VALUES (?, ?, ?, ?, ?)`,
			orderID, line.OfferID, line.Quantity, line.UnitPriceCents, line.ProductTitle); err != nil {
			serverError(w, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	h.writeOrder(w, r, http.StatusCreated, user.ID, orderID)
}

// cartProblemMessage explains the first problem in a quote in plain words.
func cartProblemMessage(q Quote) string {
	for _, line := range q.Items {
		switch line.Problem {
		case "not_found":
			return "An item in your cart is no longer available. Please remove it and try again."
		case "out_of_stock":
			return fmt.Sprintf("“%s” from %s just sold out. Please choose another seller.", line.ProductTitle, line.SellerName)
		case "not_enough_stock":
			return fmt.Sprintf("Only %d left of “%s” from %s. Please lower the quantity.", line.Stock, line.ProductTitle, line.SellerName)
		}
	}
	return "Your cart can't be checked out right now."
}

// ---------- 2. Pay ----------

// PaymentDetails is the (fake) card form. The full number is only checked, never stored or logged.
type PaymentDetails struct {
	Cardholder string `json:"cardholder"`
	CardNumber string `json:"card_number"`
	Expiry     string `json:"expiry"` // "MM/YY"
	CVC        string `json:"cvc"`
}

// declineSuffix: like real payment providers' test cards, a card ending in 0002 is always declined,
// so the failure path can be demonstrated (e.g. 4000 0000 0000 0002).
const declineSuffix = "0002"

// errSoldOut means a key we needed was taken before we could claim it.
var errSoldOut = errors.New("sold out")

// PayOrder answers POST /api/orders/{id}/pay with PaymentDetails.
func (h *Handler) PayOrder(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "Order not found.")
		return
	}
	var payment PaymentDetails
	if !readJSON(w, r, &payment) {
		return
	}

	// Look at the order first. Someone else's order gets the SAME 404 as a missing one,
	// so nobody can find out which order numbers exist by trying them.
	var status string
	err = h.DB.QueryRowContext(r.Context(),
		`SELECT status FROM orders WHERE id = ? AND user_id = ?`, orderID, user.ID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Order not found.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	switch status {
	case "paid":
		// Paying twice (double-click, retry after a timeout) does nothing: we just answer with
		// the already-paid order. No second charge, no second set of keys. ("Idempotent")
		h.writeOrder(w, r, http.StatusOK, user.ID, orderID)
		return
	case "failed":
		writeError(w, http.StatusConflict, "This order couldn't be completed because an item sold out. Please start a new checkout.")
		return
	}

	cardDigits, err := validatePayment(payment, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.HasSuffix(cardDigits, declineSuffix) {
		// 402 Payment Required. The order stays "pending", so the user can try another card.
		writeError(w, http.StatusPaymentRequired, "Your card was declined. No money was taken. Please try another card.")
		return
	}

	// "Charge" succeeded: deliver the keys.
	err = h.completeOrder(r.Context(), user.ID, orderID, cardDigits[len(cardDigits)-4:])
	if errors.Is(err, errSoldOut) {
		writeError(w, http.StatusConflict, "Sorry, an item in your order sold out while you were paying. Your card was not charged.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	h.writeOrder(w, r, http.StatusOK, user.ID, orderID)
}

// completeOrder does everything that must happen together, in ONE transaction:
// claim the keys, link them to the order lines, update sales counters, mark the order paid.
// If any step fails, Rollback undoes all of them: we never end up with "keys sold, order unpaid".
func (h *Handler) completeOrder(ctx context.Context, userID, orderID int64, cardLast4 string) error {
	// _txlock=immediate (see db.go): BEGIN waits until no other write transaction is running.
	// Two buyers of the last key are handled one after the other, never at the same time.
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check the status again INSIDE the transaction: a double-click may have paid it
	// between our first check and now. Then there's nothing left to do.
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ? AND user_id = ?`, orderID, userID).Scan(&status); err != nil {
		return err
	}
	if status != "pending" {
		return nil // already paid (or failed): don't touch anything
	}

	// Read all lines first, then update (we can't run UPDATEs while still reading rows).
	type line struct {
		itemID, offerID int64
		quantity        int
	}
	var lines []line
	rows, err := tx.QueryContext(ctx, `SELECT id, offer_id, quantity FROM order_items WHERE order_id = ?`, orderID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.itemID, &l.offerID, &l.quantity); err != nil {
			rows.Close()
			return err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, l := range lines {
		if err := claimKeys(ctx, tx, l.offerID, l.itemID, l.quantity); err != nil {
			if errors.Is(err, errSoldOut) {
				// Undo the keys claimed so far, then mark the order failed in its own small write.
				tx.Rollback()
				h.DB.ExecContext(ctx, `UPDATE orders SET status = 'failed' WHERE id = ? AND status = 'pending'`, orderID)
			}
			return err
		}
		// Keep "best sellers" up to date.
		if _, err := tx.ExecContext(ctx,
			`UPDATE products SET sales_count = sales_count + ? WHERE id = (SELECT product_id FROM offers WHERE id = ?)`,
			l.quantity, l.offerID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE sellers SET sales_count = sales_count + ? WHERE id = (SELECT seller_id FROM offers WHERE id = ?)`,
			l.quantity, l.offerID); err != nil {
			return err
		}
	}

	// "AND status = 'pending'" makes this update conditional: it only succeeds once.
	res, err := tx.ExecContext(ctx,
		`UPDATE orders SET status = 'paid', paid_at = datetime('now'), card_last4 = ? WHERE id = ? AND status = 'pending'`,
		cardLast4, orderID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("order was changed by another request")
	}
	return tx.Commit()
}

// testHookBeforeClaim runs between FINDING available keys and CLAIMING them: the most
// dangerous moment, where two buyers could pick the same key. It does nothing in production.
// TestLastKeyRace replaces it with a short pause so the race happens every time, which proves
// our protections work instead of hoping the timing happens to line up.
var testHookBeforeClaim = func() {}

// claimKeys marks `quantity` available keys of an offer as sold and links them to the order line.
func claimKeys(ctx context.Context, tx *sql.Tx, offerID, orderItemID int64, quantity int) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT id FROM game_keys WHERE offer_id = ? AND status = 'available' ORDER BY id LIMIT ?`, offerID, quantity)
	if err != nil {
		return err
	}
	var keyIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		keyIDs = append(keyIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(keyIDs) < quantity {
		return errSoldOut
	}

	testHookBeforeClaim()

	for _, keyID := range keyIDs {
		// Conditional update: "...AND status = 'available'". If another buyer took this key
		// first, 0 rows change and we stop. A key can never be sold twice, even if the
		// transaction lock above were missing (defense in depth: two independent protections).
		res, err := tx.ExecContext(ctx,
			`UPDATE game_keys SET status = 'sold', order_item_id = ? WHERE id = ? AND status = 'available'`,
			orderItemID, keyID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return errSoldOut
		}
	}
	return nil
}

// validatePayment checks the fake card form and returns the card number's digits.
func validatePayment(p PaymentDetails, now time.Time) (string, error) {
	if name := strings.TrimSpace(p.Cardholder); len(name) < 2 || len(name) > 60 {
		return "", errors.New("Please enter the name on the card.")
	}

	// Allow spaces and dashes as people type them: "4242 4242 4242 4242".
	digits := strings.NewReplacer(" ", "", "-", "").Replace(p.CardNumber)
	if len(digits) < 13 || len(digits) > 19 || !allDigits(digits) || !luhnValid(digits) {
		return "", errors.New("Card number is not valid. For a test payment use 4242 4242 4242 4242.")
	}

	month, year, ok := parseExpiry(p.Expiry)
	if !ok {
		return "", errors.New("Expiry date must look like MM/YY, e.g. 08/29.")
	}
	// A card is valid until the END of its expiry month: "08/29" works until 31 Aug 2029.
	firstDayAfter := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	if !now.Before(firstDayAfter) {
		return "", errors.New("This card has expired.")
	}

	if len(p.CVC) < 3 || len(p.CVC) > 4 || !allDigits(p.CVC) {
		return "", errors.New("CVC must be the 3 or 4 digits on the back of the card.")
	}
	return digits, nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// luhnValid runs the Luhn checksum every real card number passes. It catches typos
// like one wrong digit or two swapped digits before we ever "charge" anything.
// From the right: double every second digit (subtract 9 if it gets above 9), add everything up;
// the total must end in 0.
func luhnValid(digits string) bool {
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// parseExpiry: "08/29" -> 8, 2029, true.
func parseExpiry(s string) (month, year int, ok bool) {
	mm, yy, found := strings.Cut(strings.TrimSpace(s), "/")
	if !found || len(mm) != 2 || len(yy) != 2 || !allDigits(mm) || !allDigits(yy) {
		return 0, 0, false
	}
	month, _ = strconv.Atoi(mm)
	year, _ = strconv.Atoi(yy)
	if month < 1 || month > 12 {
		return 0, 0, false
	}
	return month, 2000 + year, true
}

// ---------- 3. Read orders ----------

// ListOrders answers GET /api/orders: the user's orders, newest first.
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id FROM orders WHERE user_id = ? ORDER BY id DESC LIMIT 50`, user.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			serverError(w, err)
			return
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		serverError(w, err)
		return
	}

	orders := make([]Order, 0, len(ids))
	for _, id := range ids {
		order, err := loadOrder(r.Context(), h.DB, user.ID, id)
		if err != nil {
			serverError(w, err)
			return
		}
		orders = append(orders, order)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": orders})
}

// GetOrder answers GET /api/orders/{id}: one order with its keys, owner only.
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "Order not found.")
		return
	}
	h.writeOrder(w, r, http.StatusOK, currentUser(r).ID, orderID)
}

// writeOrder loads an order and sends it, or 404 if it isn't this user's.
func (h *Handler) writeOrder(w http.ResponseWriter, r *http.Request, status int, userID, orderID int64) {
	order, err := loadOrder(r.Context(), h.DB, userID, orderID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Order not found.")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, status, order)
}

// loadOrder reads one order with its lines and keys. "WHERE id = ? AND user_id = ?" is the
// ownership check: another user's order simply isn't found (sql.ErrNoRows).
func loadOrder(ctx context.Context, db dbRunner, userID, orderID int64) (Order, error) {
	var o Order
	err := db.QueryRowContext(ctx,
		`SELECT id, status, total_cents, created_at, paid_at, card_last4 FROM orders WHERE id = ? AND user_id = ?`,
		orderID, userID).Scan(&o.ID, &o.Status, &o.TotalCents, &o.CreatedAt, &o.PaidAt, &o.CardLast4)
	if err != nil {
		return Order{}, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT oi.id, oi.offer_id, oi.product_title, p.slug, p.image_url, p.platform, oi.quantity, oi.unit_price_cents
		FROM order_items oi
		JOIN offers o ON o.id = oi.offer_id
		JOIN products p ON p.id = o.product_id
		WHERE oi.order_id = ?
		ORDER BY oi.id`, orderID)
	if err != nil {
		return Order{}, err
	}
	o.Items = []OrderItem{}
	position := map[int64]int{} // order item id -> index in o.Items, to attach keys below
	for rows.Next() {
		item := OrderItem{Keys: []string{}}
		if err := rows.Scan(&item.ID, &item.OfferID, &item.ProductTitle, &item.ProductSlug, &item.ImageURL,
			&item.Platform, &item.Quantity, &item.UnitPriceCents); err != nil {
			rows.Close()
			return Order{}, err
		}
		position[item.ID] = len(o.Items)
		o.Items = append(o.Items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Order{}, err
	}

	// Keys exist only for paid orders (they're linked at payment time).
	keyRows, err := db.QueryContext(ctx, `
		SELECT k.order_item_id, k.code
		FROM game_keys k
		JOIN order_items oi ON oi.id = k.order_item_id
		WHERE oi.order_id = ?
		ORDER BY k.id`, orderID)
	if err != nil {
		return Order{}, err
	}
	defer keyRows.Close()
	for keyRows.Next() {
		var itemID int64
		var code string
		if err := keyRows.Scan(&itemID, &code); err != nil {
			return Order{}, err
		}
		i := position[itemID]
		o.Items[i].Keys = append(o.Items[i].Keys, code)
	}
	return o, keyRows.Err()
}
