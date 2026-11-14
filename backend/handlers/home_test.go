package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// call sends a request through the real router and decodes the JSON answer into `into`.
// body may be "" for GET requests. token "" = not logged in.
func call(t *testing.T, h *Handler, method, path, body, token string, into any) int {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	h.Routes().ServeHTTP(recorder, request)
	if into != nil {
		if err := json.Unmarshal(recorder.Body.Bytes(), into); err != nil {
			t.Fatalf("%s %s: bad JSON %q: %v", method, path, recorder.Body.String(), err)
		}
	}
	return recorder.Code
}

func TestHomeBlocksComeFromTheDatabase(t *testing.T) {
	h := newTestHandler(t)

	var res struct{ Items []HomeBlock }
	call(t, h, "GET", "/api/home/blocks", "", "", &res)
	if len(res.Items) != 3 || res.Items[0].Type != "banner" || res.Items[2].Title != "Top deals" {
		t.Fatalf("blocks = %+v", res.Items)
	}
	if res.Items[0].LinkURL != nil || res.Items[1].LinkURL == nil {
		t.Errorf("banner should have no link, best sellers should")
	}

	// Change the database like an admin would: hide best sellers, move deals to the top, rename it.
	h.DB.Exec(`UPDATE homepage_blocks SET is_active = 0 WHERE endpoint = '/api/home/best-sellers'`)
	h.DB.Exec(`UPDATE homepage_blocks SET position = 0, title = 'Black Friday' WHERE endpoint = '/api/home/deals'`)

	call(t, h, "GET", "/api/home/blocks", "", "", &res)
	if len(res.Items) != 2 || res.Items[0].Title != "Black Friday" || res.Items[1].Type != "banner" {
		t.Errorf("after DB change: %+v", res.Items)
	}
}

func TestHomeRows(t *testing.T) {
	h := newTestHandler(t)

	var deals struct{ Items []ProductCard }
	call(t, h, "GET", "/api/home/deals?limit=8", "", "", &deals)
	if len(deals.Items) != 8 {
		t.Fatalf("deals: %d items, want 8", len(deals.Items))
	}
	for i, p := range deals.Items {
		if p.DiscountPercent <= 0 || p.LowestPriceCents == nil {
			t.Errorf("deal %s has no discount or no stock", p.Slug)
		}
		if i > 0 && p.DiscountPercent > deals.Items[i-1].DiscountPercent {
			t.Errorf("deals not sorted by discount at %d", i)
		}
	}

	var best struct{ Items []ProductCard }
	call(t, h, "GET", "/api/home/best-sellers", "", "", &best)
	if len(best.Items) != 12 {
		t.Errorf("best sellers: %d items, want 12 (default limit)", len(best.Items))
	}

	var banners struct{ Items []Banner }
	call(t, h, "GET", "/api/home/banners", "", "", &banners)
	if len(banners.Items) != 3 || !strings.Contains(banners.Items[0].ImageURL, "layout=wide") {
		t.Errorf("banners = %+v", banners.Items)
	}

	if code := call(t, h, "GET", "/api/home/deals?limit=500", "", "", nil); code != http.StatusBadRequest {
		t.Errorf("limit=500: status %d, want 400", code)
	}
}

func TestProductPageEndpoints(t *testing.T) {
	h := newTestHandler(t)

	var p ProductDetail
	code := call(t, h, "GET", "/api/products/elden-ring-pc-steam-global", "", "", &p)
	if code != http.StatusOK || p.Title != "Elden Ring (PC) Steam Key GLOBAL" || p.Description == "" || p.ReleaseDate != "2022-02-25" {
		t.Errorf("product: code=%d %+v", code, p)
	}

	var offers struct{ Items []Offer }
	call(t, h, "GET", "/api/products/elden-ring-pc-steam-global/offers", "", "", &offers)
	if len(offers.Items) < 2 {
		t.Fatalf("offers: %d", len(offers.Items))
	}
	// In-stock offers first, cheapest first; then sold-out offers.
	seenSoldOut := false
	for i, o := range offers.Items {
		if o.Stock == 0 {
			seenSoldOut = true
			continue
		}
		if seenSoldOut {
			t.Errorf("in-stock offer after a sold-out one at %d", i)
		}
		if i > 0 && offers.Items[i-1].Stock > 0 && o.PriceCents < offers.Items[i-1].PriceCents {
			t.Errorf("offers not cheapest-first at %d", i)
		}
	}
	// The product's "from" price equals the cheapest in-stock offer.
	if p.LowestPriceCents == nil || *p.LowestPriceCents != offers.Items[0].PriceCents {
		t.Errorf("lowest price %v != first offer %d", p.LowestPriceCents, offers.Items[0].PriceCents)
	}

	// /filters must still reach the filters handler, not be treated as a slug.
	var filters map[string]any
	call(t, h, "GET", "/api/products/filters", "", "", &filters)
	if _, ok := filters["platforms"]; !ok {
		t.Error("/api/products/filters was routed to the product handler")
	}

	for _, path := range []string{"/api/products/no-such-game", "/api/products/no-such-game/offers"} {
		if code := call(t, h, "GET", path, "", "", nil); code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, code)
		}
	}
}

func TestWideBannerCover(t *testing.T) {
	h := newTestHandler(t)
	request := httptest.NewRequest("GET", "/api/covers/elden-ring-pc-steam-global.svg?layout=wide", nil)
	recorder := httptest.NewRecorder()
	h.Routes().ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), `width="1200"`) {
		t.Error("wide layout is not 1200 wide")
	}
}
