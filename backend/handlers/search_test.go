package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type suggestResponse struct {
	Items      []ProductCard `json:"items"`
	Total      int           `json:"total"`
	Suggestion *ProductCard  `json:"suggestion"`
}

func getSuggest(t *testing.T, h *Handler, query string) (int, suggestResponse) {
	t.Helper()
	// url.QueryEscape turns "eldn ring" into "eldn+ring", as a browser would.
	request := httptest.NewRequest(http.MethodGet, "/api/search/suggest?q="+url.QueryEscape(query), nil)
	recorder := httptest.NewRecorder()
	h.SearchSuggest(recorder, request)
	var body suggestResponse
	json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder.Code, body
}

// End-to-end through the real seeded catalog: typo -> right product first.
func TestSuggestWithRealCatalog(t *testing.T) {
	h := newTestHandler(t)

	cases := map[string]string{ // query -> first result's slug must start with this
		"eldn ring":    "elden-ring",
		"wticher":      "the-witcher-3",
		"gta 5":        "grand-theft-auto-v",
		"spiderman":    "marvel-s-spider-man-2",
		"hogwarts leg": "hogwarts-legacy",
		"stardw":       "stardew-valley",
		"game pass":    "xbox-game-pass",
	}
	for query, wantPrefix := range cases {
		code, res := getSuggest(t, h, query)
		if code != http.StatusOK || len(res.Items) == 0 {
			t.Errorf("%q: status %d, %d items", query, code, len(res.Items))
			continue
		}
		if !strings.HasPrefix(res.Items[0].Slug, wantPrefix) {
			t.Errorf("%q: first is %s, want %s...", query, res.Items[0].Slug, wantPrefix)
		}
		if len(res.Items) > suggestLimit {
			t.Errorf("%q: %d items, max is %d", query, len(res.Items), suggestLimit)
		}
	}
}

func TestSuggestTotalAndEmptyQuery(t *testing.T) {
	h := newTestHandler(t)

	// "steam" matches many products: only 6 shown, but total tells the dropdown how many exist.
	_, res := getSuggest(t, h, "steam")
	if len(res.Items) != suggestLimit || res.Total <= suggestLimit {
		t.Errorf("steam: items=%d total=%d", len(res.Items), res.Total)
	}

	code, res := getSuggest(t, h, "   ")
	if code != http.StatusOK || len(res.Items) != 0 || res.Total != 0 {
		t.Errorf("blank query: code=%d items=%d", code, len(res.Items))
	}

	if code, _ := getSuggest(t, h, strings.Repeat("a", 101)); code != http.StatusBadRequest {
		t.Errorf("101-char query: status %d, want 400", code)
	}
}

func TestSuggestDidYouMean(t *testing.T) {
	h := newTestHandler(t)
	_, res := getSuggest(t, h, "red ded redemtion")
	if len(res.Items) != 0 {
		t.Skip("search now finds this directly")
	}
	if res.Suggestion == nil || !strings.HasPrefix(res.Suggestion.Slug, "red-dead-redemption-2") {
		t.Errorf("suggestion = %+v, want Red Dead Redemption 2", res.Suggestion)
	}
}

// Search + store filters together: ?q=elden&platform=PlayStation
func TestListProductsWithSearch(t *testing.T) {
	h := newTestHandler(t)

	_, res := getProducts(t, h, "q=elden")
	if res.Total != 3 || res.Sort != "relevance" {
		t.Errorf("q=elden: total=%d sort=%s, want 3 products sorted by relevance", res.Total, res.Sort)
	}

	_, res = getProducts(t, h, "q=elden&platform=PlayStation")
	if res.Total != 1 || res.Items[0].Platform != "PlayStation" {
		t.Errorf("q=elden&platform=PlayStation: total=%d", res.Total)
	}

	// Search results can still be sorted by price.
	_, res = getProducts(t, h, "q=steam&sort=price_asc")
	for i := 1; i < len(res.Items); i++ {
		a, b := res.Items[i-1].LowestPriceCents, res.Items[i].LowestPriceCents
		if a != nil && b != nil && *a > *b {
			t.Fatalf("q=steam&sort=price_asc not sorted at %d", i)
		}
	}

	// Nonsense: no items, total 0, still 200.
	code, res := getProducts(t, h, "q=qwrtzpx")
	if code != http.StatusOK || res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("nonsense query: code=%d total=%d", code, res.Total)
	}

	// relevance without a search quietly behaves like "popular".
	code, _ = getProducts(t, h, "sort=relevance")
	if code != http.StatusOK {
		t.Errorf("sort=relevance without q: status %d", code)
	}
}
