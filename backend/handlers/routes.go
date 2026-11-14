package handlers

import "net/http"

// Routes connects every URL to its handler. It lives here (not in main.go) so the tests
// can send requests through the exact same router the real server uses.
//
// Since Go 1.22 the standard router supports methods and {wildcards}:
// "GET /api/products/{slug}" matches GET /api/products/elden-ring-pc-steam-global,
// and the handler reads the value with r.PathValue("slug").
// When two patterns match, the more specific one wins, so /api/products/filters is not a slug.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)

	// Catalog and search
	mux.HandleFunc("GET /api/products", h.ListProducts)
	mux.HandleFunc("GET /api/products/filters", h.ProductFilters)
	mux.HandleFunc("GET /api/products/{slug}", h.GetProduct)
	mux.HandleFunc("GET /api/products/{slug}/offers", h.GetProductOffers)
	mux.HandleFunc("GET /api/search/suggest", h.SearchSuggest)
	mux.HandleFunc("GET /api/covers/{file}", h.Cover)

	// Homepage blocks
	mux.HandleFunc("GET /api/home/blocks", h.HomeBlocks)
	mux.HandleFunc("GET /api/home/banners", h.HomeBanners)
	mux.HandleFunc("GET /api/home/best-sellers", h.HomeBestSellers)
	mux.HandleFunc("GET /api/home/deals", h.HomeDeals)

	return mux
}
