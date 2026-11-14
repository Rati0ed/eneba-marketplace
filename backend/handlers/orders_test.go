package handlers

import (
	"fmt"

	"github.com/Rati0ed/eneba-marketplace/backend/db"
	"net/http"
	"sync"
	"testing"
	"time"
)

const (
	goodCard     = `{"cardholder":"Ana Test","card_number":"4242 4242 4242 4242","expiry":"12/39","cvc":"123"}`
	declinedCard = `{"cardholder":"Ana Test","card_number":"4000 0000 0000 0002","expiry":"12/39","cvc":"123"}`
)

// offerWithStock finds an offer that has at least `min` available keys.
func offerWithStock(t *testing.T, h *Handler, min int) (offerID int64, priceCents int) {
	t.Helper()
	err := h.DB.QueryRow(`
		SELECT o.id, o.price_cents FROM offers o
		WHERE (SELECT COUNT(*) FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available') >= ?
		ORDER BY o.id LIMIT 1`, min).Scan(&offerID, &priceCents)
	if err != nil {
		t.Fatalf("no offer with %d keys: %v", min, err)
	}
	return offerID, priceCents
}

func availableKeys(t *testing.T, h *Handler, offerID int64) int {
	t.Helper()
	var n int
	h.DB.QueryRow(`SELECT COUNT(*) FROM game_keys WHERE offer_id = ? AND status = 'available'`, offerID).Scan(&n)
	return n
}

func createOrder(t *testing.T, h *Handler, token string, offerID int64, quantity int) (int, Order) {
	t.Helper()
	var order Order
	body := fmt.Sprintf(`{"items":[{"offer_id":%d,"quantity":%d}]}`, offerID, quantity)
	code := call(t, h, "POST", "/api/orders", body, token, &order)
	return code, order
}

func pay(t *testing.T, h *Handler, token string, orderID int64, card string) (int, Order) {
	t.Helper()
	var order Order
	code := call(t, h, "POST", fmt.Sprintf("/api/orders/%d/pay", orderID), card, token, &order)
	return code, order
}

func TestCheckoutHappyPath(t *testing.T) {
	h := newTestHandler(t)
	token := registerUser(t, h, "ana@example.com")
	offerID, price := offerWithStock(t, h, 3)
	stockBefore := availableKeys(t, h, offerID)

	// Tampered request: the client claims each key costs 1 cent. The server must ignore it.
	var order Order
	body := fmt.Sprintf(`{"items":[{"offer_id":%d,"quantity":2,"unit_price_cents":1,"price_cents":1}],"total_cents":2}`, offerID)
	code := call(t, h, "POST", "/api/orders", body, token, &order)
	if code != http.StatusCreated || order.Status != "pending" || order.TotalCents != 2*price {
		t.Fatalf("create: code=%d status=%s total=%d, want 201 pending %d", code, order.Status, order.TotalCents, 2*price)
	}
	if len(order.Items[0].Keys) != 0 {
		t.Error("a pending order must not show keys")
	}

	code, order = pay(t, h, token, order.ID, goodCard)
	if code != http.StatusOK || order.Status != "paid" || order.PaidAt == nil || *order.CardLast4 != "4242" {
		t.Fatalf("pay: code=%d %+v", code, order)
	}
	if len(order.Items[0].Keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(order.Items[0].Keys))
	}
	if got := availableKeys(t, h, offerID); got != stockBefore-2 {
		t.Errorf("stock %d -> %d, want -2", stockBefore, got)
	}

	// The keys are really marked sold and linked to this order line.
	var sold int
	h.DB.QueryRow(`SELECT COUNT(*) FROM game_keys WHERE status = 'sold' AND order_item_id = ?`, order.Items[0].ID).Scan(&sold)
	if sold != 2 {
		t.Errorf("%d keys linked to the order line, want 2", sold)
	}

	// Order history shows it with the keys.
	var history struct{ Items []Order }
	call(t, h, "GET", "/api/orders", "", token, &history)
	if len(history.Items) != 1 || len(history.Items[0].Items[0].Keys) != 2 {
		t.Errorf("history = %+v", history.Items)
	}
}

func TestPayingTwiceDoesNothing(t *testing.T) {
	h := newTestHandler(t)
	token := registerUser(t, h, "ana@example.com")
	offerID, _ := offerWithStock(t, h, 4)
	_, order := createOrder(t, h, token, offerID, 1)
	stockBefore := availableKeys(t, h, offerID)

	_, first := pay(t, h, token, order.ID, goodCard)
	code, second := pay(t, h, token, order.ID, goodCard)
	if code != http.StatusOK || second.Items[0].Keys[0] != first.Items[0].Keys[0] {
		t.Errorf("second payment: code %d, keys %v vs %v", code, second.Items[0].Keys, first.Items[0].Keys)
	}
	if got := availableKeys(t, h, offerID); got != stockBefore-1 {
		t.Errorf("paying twice used %d keys, want 1", stockBefore-got)
	}

	// A double-click: two payments for the SAME order at the SAME time.
	_, order2 := createOrder(t, h, token, offerID, 1)
	stockBefore = availableKeys(t, h, offerID)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := range codes {
		wg.Add(1)
		go func() { // goroutine = runs at the same time as the other one
			defer wg.Done()
			codes[i], _ = pay(t, h, token, order2.ID, goodCard)
		}()
	}
	wg.Wait() // wait until both goroutines are finished
	if codes[0] != http.StatusOK || codes[1] != http.StatusOK {
		t.Errorf("simultaneous payments: %v, want both 200", codes)
	}
	if got := availableKeys(t, h, offerID); got != stockBefore-1 {
		t.Errorf("simultaneous payments used %d keys, want 1", stockBefore-got)
	}
}

func TestDeclinedCardCanBeRetried(t *testing.T) {
	h := newTestHandler(t)
	token := registerUser(t, h, "ana@example.com")
	offerID, _ := offerWithStock(t, h, 2)
	_, order := createOrder(t, h, token, offerID, 1)
	stockBefore := availableKeys(t, h, offerID)

	var res struct{ Error string }
	code := call(t, h, "POST", fmt.Sprintf("/api/orders/%d/pay", order.ID), declinedCard, token, &res)
	if code != http.StatusPaymentRequired || res.Error == "" {
		t.Fatalf("declined card: status %d %q, want 402", code, res.Error)
	}
	var status string
	h.DB.QueryRow(`SELECT status FROM orders WHERE id = ?`, order.ID).Scan(&status)
	if status != "pending" || availableKeys(t, h, offerID) != stockBefore {
		t.Errorf("after decline: status %s, keys %d -> %d; want pending and no keys used", status, stockBefore, availableKeys(t, h, offerID))
	}

	// Same order, another card: works.
	if code, paid := pay(t, h, token, order.ID, goodCard); code != http.StatusOK || paid.Status != "paid" {
		t.Errorf("retry with good card: %d %s", code, paid.Status)
	}
}

func TestInvalidPaymentDetails(t *testing.T) {
	h := newTestHandler(t)
	token := registerUser(t, h, "ana@example.com")
	offerID, _ := offerWithStock(t, h, 1)
	_, order := createOrder(t, h, token, offerID, 1)

	bad := map[string]string{
		"luhn fails": `{"cardholder":"Ana","card_number":"4242 4242 4242 4241","expiry":"12/39","cvc":"123"}`,
		"letters":    `{"cardholder":"Ana","card_number":"4242 abcd 4242 4242","expiry":"12/39","cvc":"123"}`,
		"too short":  `{"cardholder":"Ana","card_number":"4242","expiry":"12/39","cvc":"123"}`,
		"expired":    `{"cardholder":"Ana","card_number":"4242424242424242","expiry":"01/20","cvc":"123"}`,
		"month 13":   `{"cardholder":"Ana","card_number":"4242424242424242","expiry":"13/39","cvc":"123"}`,
		"bad cvc":    `{"cardholder":"Ana","card_number":"4242424242424242","expiry":"12/39","cvc":"12"}`,
		"no name":    `{"cardholder":" ","card_number":"4242424242424242","expiry":"12/39","cvc":"123"}`,
		"not json":   `card please`,
	}
	for name, card := range bad {
		if code, _ := pay(t, h, token, order.ID, card); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, code)
		}
	}
}

func TestExpiryIsValidUntilEndOfMonth(t *testing.T) {
	card := PaymentDetails{Cardholder: "Ana", CardNumber: "4242424242424242", CVC: "123", Expiry: "11/26"}
	if _, err := validatePayment(card, time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC)); err != nil {
		t.Errorf("11/26 on 30 Nov 2026 should be valid: %v", err)
	}
	if _, err := validatePayment(card, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Error("11/26 on 1 Dec 2026 should be expired")
	}
}

func TestLastKeyRace(t *testing.T) {
	h := newTestHandler(t)
	offerID, _ := offerWithStock(t, h, 2)
	// Leave exactly ONE available key on this offer.
	h.DB.Exec(`DELETE FROM game_keys WHERE offer_id = ? AND id > (SELECT MIN(id) FROM game_keys WHERE offer_id = ?)`, offerID, offerID)
	if availableKeys(t, h, offerID) != 1 {
		t.Fatal("setup: want exactly 1 key")
	}

	// 8 different buyers each create an order while the 1 key is still available...
	const buyers = 8
	tokens := make([]string, buyers)
	orderIDs := make([]int64, buyers)
	for i := range buyers {
		tokens[i] = registerUser(t, h, fmt.Sprintf("buyer%d@example.com", i))
		code, order := createOrder(t, h, tokens[i], offerID, 1)
		if code != http.StatusCreated {
			t.Fatalf("buyer %d create: %d", i, code)
		}
		orderIDs[i] = order.ID
	}

	// Pause 20 ms right after a buyer has FOUND the free key and before it CLAIMS it.
	// Without protection, every buyer would find the same key during this pause.
	testHookBeforeClaim = func() { time.Sleep(20 * time.Millisecond) }
	t.Cleanup(func() { testHookBeforeClaim = func() {} })

	// ...and all pay at the same instant. Every goroutine waits on the `start` channel;
	// close(start) releases them together, like a starting gun, so the payments really overlap.
	start := make(chan struct{})
	codes := make([]int, buyers)
	var wg sync.WaitGroup
	for i := range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			codes[i], _ = pay(t, h, tokens[i], orderIDs[i], goodCard)
		}()
	}
	close(start)
	wg.Wait()

	// Exactly one buyer wins (200), all others get 409 "sold out", and the key is sold once.
	wins, losses := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			wins++
		case http.StatusConflict:
			losses++
		}
	}
	if wins != 1 || losses != buyers-1 {
		t.Errorf("codes = %v, want one 200 and %d x 409", codes, buyers-1)
	}
	var sold int
	h.DB.QueryRow(`SELECT COUNT(*) FROM game_keys WHERE offer_id = ? AND status = 'sold'`, offerID).Scan(&sold)
	if sold != 1 {
		t.Errorf("key sold %d times", sold)
	}
	var paid, failed int
	// Only count THIS test's orders (the seeded demo account has a paid order too).
	h.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE status = 'paid' AND id >= ?`, orderIDs[0]).Scan(&paid)
	h.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE status = 'failed' AND id >= ?`, orderIDs[0]).Scan(&failed)
	if paid != 1 || failed != buyers-1 {
		t.Errorf("orders: %d paid, %d failed; want 1 and %d", paid, failed, buyers-1)
	}
}

func TestOutOfStockAtOrderCreation(t *testing.T) {
	h := newTestHandler(t)
	token := registerUser(t, h, "ana@example.com")
	offerID, _ := offerWithStock(t, h, 1)
	h.DB.Exec(`DELETE FROM game_keys WHERE offer_id = ? AND id > (SELECT MIN(id) FROM game_keys WHERE offer_id = ?)`, offerID, offerID)

	if code, _ := createOrder(t, h, token, offerID, 2); code != http.StatusConflict {
		t.Errorf("2 keys when 1 left: status %d, want 409", code)
	}
	if code, _ := createOrder(t, h, token, 999999, 1); code != http.StatusConflict {
		t.Errorf("unknown offer: status %d, want 409", code)
	}
	if code := call(t, h, "POST", "/api/orders", `{"items":[]}`, token, nil); code != http.StatusBadRequest {
		t.Errorf("empty cart: status %d, want 400", code)
	}
}

func TestOrdersArePrivate(t *testing.T) {
	h := newTestHandler(t)
	ana := registerUser(t, h, "ana@example.com")
	bob := registerUser(t, h, "bob@example.com")
	offerID, _ := offerWithStock(t, h, 1)
	_, order := createOrder(t, h, ana, offerID, 1)
	pay(t, h, ana, order.ID, goodCard)

	path := fmt.Sprintf("/api/orders/%d", order.ID)
	// Bob gets 404, exactly as if the order didn't exist (no hint that it does).
	if code := call(t, h, "GET", path, "", bob, nil); code != http.StatusNotFound {
		t.Errorf("Bob reads Ana's order: %d, want 404", code)
	}
	if code, _ := pay(t, h, bob, order.ID, goodCard); code != http.StatusNotFound {
		t.Errorf("Bob pays Ana's order: %d, want 404", code)
	}
	var bobsHistory struct{ Items []Order }
	call(t, h, "GET", "/api/orders", "", bob, &bobsHistory)
	if len(bobsHistory.Items) != 0 {
		t.Errorf("Bob sees %d orders, want 0", len(bobsHistory.Items))
	}

	// Without a token: 401 everywhere.
	for _, req := range [][2]string{{"GET", "/api/orders"}, {"GET", path}, {"POST", "/api/orders"}, {"POST", path + "/pay"}} {
		if code := call(t, h, req[0], req[1], "", "", nil); code != http.StatusUnauthorized {
			t.Errorf("%s %s without login: %d, want 401", req[0], req[1], code)
		}
	}
	if code := call(t, h, "GET", "/api/orders/abc", "", ana, nil); code != http.StatusNotFound {
		t.Errorf("non-numeric id: %d, want 404", code)
	}
}

func TestLuhn(t *testing.T) {
	valid := []string{"4242424242424242", "4000000000000002", "5555555555554444", "378282246310005"}
	invalid := []string{"4242424242424241", "1234567812345678", "4000000000000003"}
	for _, n := range valid {
		if !luhnValid(n) {
			t.Errorf("%s should pass Luhn", n)
		}
	}
	for _, n := range invalid {
		if luhnValid(n) {
			t.Errorf("%s should fail Luhn", n)
		}
	}
}

func TestDemoAccount(t *testing.T) {
	h := newTestHandler(t)
	var login authResponse
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, db.DemoEmail, db.DemoPassword)
	if code := call(t, h, "POST", "/api/auth/login", body, "", &login); code != http.StatusOK {
		t.Fatalf("demo login: %d %+v", code, login)
	}
	var history struct{ Items []Order }
	call(t, h, "GET", "/api/orders", "", login.Token, &history)
	if len(history.Items) != 1 || history.Items[0].Status != "paid" {
		t.Fatalf("demo history = %+v", history.Items)
	}
	keys := 0
	for _, item := range history.Items[0].Items {
		keys += len(item.Keys)
	}
	if keys != 2 {
		t.Errorf("demo order has %d keys, want 2", keys)
	}
}
