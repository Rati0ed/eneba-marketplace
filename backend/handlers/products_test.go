package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Rati0ed/eneba-marketplace/backend/db"
	"golang.org/x/crypto/bcrypt"
)

// newTestHandler gives each test its own freshly seeded database.
func newTestHandler(t *testing.T) *Handler {
	t.Helper() // if this fails, Go reports the line in the TEST, not in this helper
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() }) // runs when the test ends, like defer
	index, err := BuildSearchIndex(database)
	if err != nil {
		t.Fatalf("search index: %v", err)
	}
	bcryptCost = bcrypt.MinCost // tests don't need slow hashing; keeps the test run fast
	return &Handler{DB: database, Search: index, JWTSecret: []byte("test-secret-that-is-at-least-32-bytes!")}
}

// listResponse is the JSON shape of GET /api/products.
type listResponse struct {
	Items      []ProductCard `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	TotalPages int           `json:"total_pages"`
	Sort       string        `json:"sort"`
}

// getProducts calls the handler directly, without a real network server.
// httptest.NewRecorder is a fake ResponseWriter that remembers what was written.
func getProducts(t *testing.T, h *Handler, query string) (int, listResponse) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/products?"+query, nil)
	recorder := httptest.NewRecorder()
	h.ListProducts(recorder, request)

	var body listResponse
	if recorder.Code == http.StatusOK {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("bad JSON: %v", err)
		}
	}
	return recorder.Code, body
}

func TestListProductsPagination(t *testing.T) {
	h := newTestHandler(t)

	_, page1 := getProducts(t, h, "")
	if page1.Total != 80 || page1.TotalPages != 4 || len(page1.Items) != pageSize {
		t.Fatalf("page 1: total=%d pages=%d items=%d, want 80/4/%d", page1.Total, page1.TotalPages, len(page1.Items), pageSize)
	}

	_, page4 := getProducts(t, h, "page=4")
	if len(page4.Items) != 80-3*pageSize {
		t.Errorf("page 4 has %d items, want %d", len(page4.Items), 80-3*pageSize)
	}

	// Pages must not overlap: the first item of page 2 is not anywhere on page 1.
	_, page2 := getProducts(t, h, "page=2")
	for _, p := range page1.Items {
		if p.ID == page2.Items[0].ID {
			t.Errorf("product %d is on page 1 and page 2", p.ID)
		}
	}

	// A page past the end is not an error, just empty.
	code, page99 := getProducts(t, h, "page=99")
	if code != http.StatusOK || len(page99.Items) != 0 || page99.Total != 80 {
		t.Errorf("page 99: code=%d items=%d total=%d", code, len(page99.Items), page99.Total)
	}
}

func TestListProductsFilters(t *testing.T) {
	h := newTestHandler(t)

	// Two platforms = OR. Every item must be one of them.
	_, res := getProducts(t, h, "platform=Steam&platform=Nintendo")
	if res.Total == 0 {
		t.Fatal("no Steam/Nintendo products")
	}
	for _, p := range res.Items {
		if p.Platform != "Steam" && p.Platform != "Nintendo" {
			t.Errorf("got platform %q", p.Platform)
		}
	}

	// Different filters = AND.
	_, res = getProducts(t, h, "platform=PlayStation&region=Europe&genre=Gift+Card")
	for _, p := range res.Items {
		if p.Platform != "PlayStation" || p.Region != "Europe" || p.Genre != "Gift Card" {
			t.Errorf("filter leak: %s / %s / %s", p.Platform, p.Region, p.Genre)
		}
	}
	if res.Total != 2 { // PlayStation Store 20 EUR and 50 EUR
		t.Errorf("PlayStation EU gift cards = %d, want 2", res.Total)
	}

	// Price range in euros: every price within €10.00–€20.00, sold-out products excluded.
	_, res = getProducts(t, h, "min_price=10&max_price=20")
	if res.Total == 0 {
		t.Fatal("no products between €10 and €20")
	}
	for _, p := range res.Items {
		if p.LowestPriceCents == nil || *p.LowestPriceCents < 1000 || *p.LowestPriceCents > 2000 {
			t.Errorf("%s price %v outside €10-20", p.Slug, p.LowestPriceCents)
		}
	}

	// Unknown values just match nothing.
	_, res = getProducts(t, h, "platform=Dreamcast")
	if res.Total != 0 {
		t.Errorf("Dreamcast total = %d, want 0", res.Total)
	}
}

func TestListProductsSorting(t *testing.T) {
	h := newTestHandler(t)

	_, res := getProducts(t, h, "sort=price_asc&page=1")
	prev := 0
	for _, p := range res.Items {
		if p.LowestPriceCents == nil {
			continue // sold-out products are at the end
		}
		if *p.LowestPriceCents < prev {
			t.Fatalf("price_asc not sorted: %d after %d", *p.LowestPriceCents, prev)
		}
		prev = *p.LowestPriceCents
	}

	_, res = getProducts(t, h, "sort=discount")
	if first, second := res.Items[0].DiscountPercent, res.Items[1].DiscountPercent; first < second {
		t.Errorf("discount sort: %d%% before %d%%", first, second)
	}
}

func TestListProductsRejectsBadInput(t *testing.T) {
	h := newTestHandler(t)
	bad := []string{
		"page=0",
		"page=abc",
		"min_price=-5",
		"max_price=12.5",
		"min_price=50&max_price=10",
		"sort=price%3B+DROP+TABLE+products", // URL-encoded "price; DROP TABLE products"
	}
	for _, query := range bad {
		if code, _ := getProducts(t, h, query); code != http.StatusBadRequest {
			t.Errorf("%q: status %d, want 400", query, code)
		}
	}
}

func TestDiscountPercent(t *testing.T) {
	price := func(cents int) *int { return &cents }
	cases := []struct {
		msrp  int
		price *int
		want  int
	}{
		{5999, price(3299), 45},
		{1000, price(1000), 0}, // same price: no discount
		{1000, price(1200), 0}, // more expensive than MSRP: no "negative discount"
		{1000, nil, 0},         // sold out
		{1000, price(995), 1},  // 0.5% rounds up to 1
	}
	for _, c := range cases {
		if got := discountPercent(c.msrp, c.price); got != c.want {
			t.Errorf("discountPercent(%d, %v) = %d, want %d", c.msrp, c.price, got, c.want)
		}
	}
}

func TestProductFilters(t *testing.T) {
	h := newTestHandler(t)
	recorder := httptest.NewRecorder()
	h.ProductFilters(recorder, httptest.NewRequest(http.MethodGet, "/api/products/filters", nil))

	var body struct {
		Platforms []FilterOption `json:"platforms"`
		Regions   []FilterOption `json:"regions"`
		Genres    []FilterOption `json:"genres"`
		Price     map[string]int `json:"price"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Platforms) != 4 || len(body.Regions) != 3 {
		t.Errorf("platforms=%d regions=%d, want 4 and 3", len(body.Platforms), len(body.Regions))
	}
	sum := 0
	for _, p := range body.Platforms {
		sum += p.Count
	}
	if sum != 80 {
		t.Errorf("platform counts add up to %d, want 80", sum)
	}
	if body.Price["min"] < 0 || body.Price["max"] <= body.Price["min"] {
		t.Errorf("bad price range %v", body.Price)
	}
}
