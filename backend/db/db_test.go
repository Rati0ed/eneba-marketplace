package db

import (
	"path/filepath"
	"testing"
)

// Run with: go test ./...
// Go finds every function named TestXxx(t *testing.T) in *_test.go files and runs it.
// Python comparison: like pytest functions named test_xxx.

func TestSeedCreatesExpectedData(t *testing.T) {
	// t.TempDir() gives a fresh folder that Go deletes after the test.
	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// Table-driven test: a list of cases, checked in one loop.
	checks := []struct {
		query string
		want  int
	}{
		{"SELECT COUNT(*) FROM products", 80},
		{"SELECT COUNT(*) FROM sellers", 8},
		{"SELECT COUNT(*) FROM homepage_blocks", 3},
		// the demo account: 1 user, 1 paid order with 2 sold keys linked to it
		{"SELECT COUNT(*) FROM users WHERE email = '" + DemoEmail + "'", 1},
		{"SELECT COUNT(*) FROM orders WHERE status = 'paid' AND card_last4 = '4242'", 1},
		{"SELECT COUNT(*) FROM game_keys WHERE status = 'sold' AND order_item_id IS NOT NULL", 2},
		{"SELECT COUNT(*) FROM orders o WHERE total_cents != (SELECT SUM(unit_price_cents * quantity) FROM order_items WHERE order_id = o.id)", 0},
		// every product has between 2 and 5 offers
		{"SELECT COUNT(*) FROM (SELECT COUNT(*) AS c FROM offers GROUP BY product_id) WHERE c NOT BETWEEN 2 AND 5", 0},
		// every banner links to a product that exists
		{"SELECT COUNT(*) FROM banners b WHERE NOT EXISTS (SELECT 1 FROM products p WHERE '/product/' || p.slug = b.link_url)", 0},
	}
	for _, c := range checks {
		var got int
		if err := database.QueryRow(c.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", c.query, err)
		}
		if got != c.want {
			t.Errorf("%s = %d, want %d", c.query, got, c.want)
		}
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// Product 99999 doesn't exist, so the database must refuse this offer.
	_, err = database.Exec(`INSERT INTO offers (product_id, seller_id, price_cents) VALUES (99999, 1, 100)`)
	if err == nil {
		t.Fatal("expected a FOREIGN KEY error, got none: foreign keys are OFF")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Baldur's Gate 3 PC Steam Global":     "baldur-s-gate-3-pc-steam-global",
		"Pokémon Scarlet Switch":              "pokemon-scarlet-switch",
		"God of War Ragnarök":                 "god-of-war-ragnarok",
		"Diablo IV Xbox Series X|S Xbox Live": "diablo-iv-xbox-series-x-s-xbox-live",
		"  Call of Duty: Black Ops 6  ":       "call-of-duty-black-ops-6",
	}
	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}
