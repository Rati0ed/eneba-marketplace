package handlers

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// pageSize = products per page on the store. Fixed on the server so nobody can ask
// for 1,000,000 rows at once.
const pageSize = 24

// sortOptions maps the `sort` URL value to an SQL ORDER BY clause.
//
// ORDER BY can't use `?` placeholders, so we never put user text into it. Instead the user
// picks a KEY, and we look up SQL that we wrote ourselves. An unknown key is rejected (400).
// This "allow-list" is how you stay safe from SQL injection when placeholders don't fit.
//
// `lowest_price_cents IS NULL` is 0 for in-stock and 1 for sold-out products, so sorting by it
// first always puts sold-out products at the end.
var sortOptions = map[string]string{
	"popular":    "lowest_price_cents IS NULL, sales_count DESC, id",
	"price_asc":  "lowest_price_cents IS NULL, lowest_price_cents ASC, sales_count DESC, id",
	"price_desc": "lowest_price_cents IS NULL, lowest_price_cents DESC, sales_count DESC, id",
	"newest":     "lowest_price_cents IS NULL, release_date DESC, sales_count DESC, id",
	// discount = how much of the MSRP you save. "* 1.0" forces decimal division (5/10 would be 0 in integers).
	"discount": "lowest_price_cents IS NULL, (msrp_cents - lowest_price_cents) * 1.0 / msrp_cents DESC, sales_count DESC, id",
}

// productFilter is everything the user can choose on the store page, already validated.
type productFilter struct {
	Query         string   // search text (typo-tolerant search is added on Day 3)
	Platforms     []string // any of these (OR); empty = all
	Regions       []string
	Genres        []string
	MinPriceCents *int // nil = no lower limit
	MaxPriceCents *int // nil = no upper limit
	Sort          string
	Page          int // 1 = first page
}

// parseProductFilter reads and validates the URL query, e.g.
// /api/products?platform=Steam&platform=Xbox&min_price=10&max_price=30&sort=price_asc&page=2
//
// Prices in the URL are WHOLE EUROS (min_price=10 = €10), because people read and share URLs.
// We convert to cents right here; everything after this point uses cents.
func parseProductFilter(query url.Values) (productFilter, error) {
	f := productFilter{
		Query: strings.TrimSpace(query.Get("q")),
		// query["platform"] returns ALL values of a repeated parameter: ?platform=Steam&platform=Xbox
		Platforms: nonEmpty(query["platform"]),
		Regions:   nonEmpty(query["region"]),
		Genres:    nonEmpty(query["genre"]),
		Sort:      query.Get("sort"),
		Page:      1,
	}

	if len(f.Query) > 100 {
		return f, fmt.Errorf("search text is too long (max 100 characters)")
	}

	if s := query.Get("page"); s != "" {
		page, err := strconv.Atoi(s)
		if err != nil || page < 1 {
			return f, fmt.Errorf("page must be a whole number of 1 or more")
		}
		f.Page = page
	}

	var err error
	if f.MinPriceCents, err = parseEuros(query.Get("min_price"), "min_price"); err != nil {
		return f, err
	}
	if f.MaxPriceCents, err = parseEuros(query.Get("max_price"), "max_price"); err != nil {
		return f, err
	}
	if f.MinPriceCents != nil && f.MaxPriceCents != nil && *f.MinPriceCents > *f.MaxPriceCents {
		return f, fmt.Errorf("min_price can't be higher than max_price")
	}

	// Default sort: best sellers. (On Day 3, a search defaults to best match.)
	if f.Sort == "" {
		f.Sort = "popular"
	}
	if _, ok := sortOptions[f.Sort]; !ok {
		return f, fmt.Errorf("unknown sort %q", f.Sort)
	}
	return f, nil
}

// parseEuros turns "15" into 1500 cents. Empty means "not set" (nil).
func parseEuros(s, name string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	euros, err := strconv.Atoi(s)
	if err != nil || euros < 0 || euros > 100000 {
		return nil, fmt.Errorf("%s must be a whole number of euros between 0 and 100000", name)
	}
	cents := euros * 100
	return &cents, nil
}

// nonEmpty drops blank values, so "?platform=" doesn't filter for platform "".
func nonEmpty(values []string) []string {
	var result []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			result = append(result, v)
		}
	}
	return result
}

// buildWhere turns the filter into an SQL WHERE clause plus its arguments.
// Each condition only gets added if the user chose that filter.
//
//	Platforms=[Steam Xbox], MinPrice=1000 ->
//	"WHERE platform IN (?, ?) AND lowest_price_cents >= ?", ["Steam", "Xbox", 1000]
func buildWhere(f productFilter) (string, []any) {
	var conditions []string
	var args []any

	// addIn adds "column IN (?, ?, ...)" for a list of values.
	addIn := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		conditions = append(conditions, column+" IN ("+placeholders(len(values))+")")
		for _, v := range values {
			args = append(args, v)
		}
	}
	addIn("platform", f.Platforms)
	addIn("region", f.Regions)
	addIn("genre", f.Genres)

	// A sold-out product (NULL price) never matches a price filter: NULL >= 1000 is not true.
	if f.MinPriceCents != nil {
		conditions = append(conditions, "lowest_price_cents >= ?")
		args = append(args, *f.MinPriceCents)
	}
	if f.MaxPriceCents != nil {
		conditions = append(conditions, "lowest_price_cents <= ?")
		args = append(args, *f.MaxPriceCents)
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// ListProducts answers GET /api/products: the store catalog with filters, sorting and pages.
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	f, err := parseProductFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	whereSQL, args := buildWhere(f)

	total, err := h.countCards(r.Context(), whereSQL, args)
	if err != nil {
		serverError(w, err)
		return
	}

	// Page 1 = rows 0-23, page 2 = rows 24-47, ...
	offset := (f.Page - 1) * pageSize
	items, err := h.queryCards(r.Context(), whereSQL, args, sortOptions[f.Sort], pageSize, offset)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":       items,
		"total":       total,
		"page":        f.Page,
		"page_size":   pageSize,
		"total_pages": int(math.Ceil(float64(total) / pageSize)),
		"sort":        f.Sort, // tells the frontend which sort was actually used
	})
}

// FilterOption is one choice in a filter list, with how many products have it.
type FilterOption struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// ProductFilters answers GET /api/products/filters: the options the store shows in its
// sidebar. The frontend builds the checkboxes from this, so adding a new platform to the
// database makes it appear in the UI with no frontend change.
func (h *Handler) ProductFilters(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{}

	// The column name comes from this fixed list, never from the user.
	for _, column := range []string{"platform", "region", "genre"} {
		rows, err := h.DB.QueryContext(r.Context(),
			`SELECT `+column+`, COUNT(*) FROM products GROUP BY `+column+` ORDER BY COUNT(*) DESC, `+column)
		if err != nil {
			serverError(w, err)
			return
		}
		options := make([]FilterOption, 0)
		for rows.Next() {
			var o FilterOption
			if err := rows.Scan(&o.Value, &o.Count); err != nil {
				rows.Close()
				serverError(w, err)
				return
			}
			options = append(options, o)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			serverError(w, err)
			return
		}
		result[column+"s"] = options // "platforms", "regions", "genres"
	}

	// Price range of in-stock products, in whole euros (rounded outwards so every product fits).
	var minCents, maxCents int
	err := h.DB.QueryRowContext(r.Context(),
		cardsCTE+` SELECT COALESCE(MIN(lowest_price_cents), 0), COALESCE(MAX(lowest_price_cents), 0) FROM cards`,
	).Scan(&minCents, &maxCents)
	if err != nil {
		serverError(w, err)
		return
	}
	result["price"] = map[string]int{
		"min": minCents / 100,        // 149 cents -> €1 (round down)
		"max": (maxCents + 99) / 100, // 7199 cents -> €72 (round up)
	}

	writeJSON(w, http.StatusOK, result)
}
