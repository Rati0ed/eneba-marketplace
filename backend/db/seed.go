package db

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
)

// seed fills the empty database with the sample data from seed_data.go.
//
// Everything runs inside ONE transaction:
//   - if any insert fails, nothing is saved (no half-filled database), and
//   - it is much faster: SQLite writes to disk once at Commit instead of once per row.
func seed(database *sql.DB) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	// If we return early with an error, Rollback undoes everything.
	// After a successful Commit, Rollback does nothing, so this is always safe.
	defer tx.Rollback()

	// A random generator with a FIXED seed (42) gives the same "random" prices,
	// offers and keys on every start. Same data every time = easier to test and debug.
	rng := rand.New(rand.NewSource(42))

	// 1. Sellers. We remember their ids to create offers later.
	var sellerIDs []int64
	for _, s := range sellers {
		res, err := tx.Exec(
			`INSERT INTO sellers (name, rating_percent, sales_count) VALUES (?, ?, ?)`,
			s.Name, s.RatingPercent, s.SalesCount,
		)
		if err != nil {
			return fmt.Errorf("insert seller %q: %w", s.Name, err)
		}
		id, _ := res.LastInsertId()
		sellerIDs = append(sellerIDs, id)
	}

	// 2. Turn games and gift cards into product rows.
	var products []productRow
	for _, g := range games {
		for i, v := range g.Variants {
			products = append(products, productRow{
				Title:       fmt.Sprintf("%s (%s) %s Key %s", g.Name, v.Device, storeName(v.Platform), strings.ToUpper(v.Region)),
				Slug:        slugify(g.Name + " " + v.Device + " " + storeName(v.Platform) + " " + v.Region),
				Platform:    v.Platform,
				Region:      v.Region,
				Genre:       g.Genre,
				Description: g.Description,
				MSRPCents:   g.MSRPCents,
				ReleaseDate: g.ReleaseDate,
				// The first variant is the most popular one; each next one sells a bit less.
				SalesCount: g.Popularity * (10 - 2*i) / 10,
				// Games are discounted 8-55%. 0.45 + [0, 0.47) = a price factor of 0.45-0.92.
				MinFactor: 0.45, MaxFactor: 0.92,
			})
		}
	}
	for _, c := range giftCards {
		products = append(products, productRow{
			Title:       fmt.Sprintf("%s %s Key %s", c.Name, c.Store, strings.ToUpper(c.Region)),
			Slug:        slugify(c.Name + " " + c.Store + " " + c.Region),
			Platform:    c.Platform,
			Region:      c.Region,
			Genre:       c.Genre,
			Description: fmt.Sprintf("Digital code for %s. Redeem it on your %s account in the %s region.", c.Name, c.Store, c.Region),
			MSRPCents:   c.FaceCents,
			ReleaseDate: "2020-01-01",
			SalesCount:  c.Popularity,
			// Gift cards are only slightly discounted: 1-10% off.
			MinFactor: 0.90, MaxFactor: 0.99,
		})
	}

	// 3. Insert products, then 2-5 offers per product, then keys per offer.
	for _, p := range products {
		res, err := tx.Exec(
			`INSERT INTO products (slug, title, platform, region, genre, description, image_url, msrp_cents, release_date, sales_count)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.Slug, p.Title, p.Platform, p.Region, p.Genre, p.Description,
			"/api/covers/"+p.Slug+".svg", p.MSRPCents, p.ReleaseDate, p.SalesCount,
		)
		if err != nil {
			return fmt.Errorf("insert product %q: %w", p.Slug, err)
		}
		productID, _ := res.LastInsertId()

		offerCount := 2 + rng.Intn(4) // rng.Intn(4) gives 0..3, so 2..5 offers
		// rng.Perm(8) is a shuffled list of 0..7. Taking the first N picks N different sellers.
		for _, sellerIndex := range rng.Perm(len(sellerIDs))[:offerCount] {
			factor := p.MinFactor + rng.Float64()*(p.MaxFactor-p.MinFactor)
			price := int(float64(p.MSRPCents) * factor)
			price = price/10*10 + 9 // make prices look like shop prices: 3412 -> 3419

			res, err := tx.Exec(
				`INSERT INTO offers (product_id, seller_id, price_cents) VALUES (?, ?, ?)`,
				productID, sellerIDs[sellerIndex], price,
			)
			if err != nil {
				return fmt.Errorf("insert offer: %w", err)
			}
			offerID, _ := res.LastInsertId()

			// About 1 in 10 offers is sold out (0 keys), so the UI must handle "out of stock".
			keyCount := 0
			if rng.Intn(10) != 0 {
				keyCount = 2 + rng.Intn(14) // 2..15 keys
			}
			for k := 0; k < keyCount; k++ {
				if _, err := tx.Exec(
					`INSERT INTO game_keys (offer_id, code) VALUES (?, ?)`,
					offerID, randomKeyCode(rng),
				); err != nil {
					return fmt.Errorf("insert key: %w", err)
				}
			}
		}
	}

	// 4. Homepage content.
	for i, b := range banners {
		if _, err := tx.Exec(
			`INSERT INTO banners (title, subtitle, image_url, link_url, position) VALUES (?, ?, ?, ?, ?)`,
			b.Title, b.Subtitle, "/api/covers/"+b.Slug+".svg", "/product/"+b.Slug, i+1,
		); err != nil {
			return fmt.Errorf("insert banner: %w", err)
		}
	}
	for i, b := range homepageBlocks {
		if _, err := tx.Exec(
			`INSERT INTO homepage_blocks (type, title, endpoint, position) VALUES (?, ?, ?, ?)`,
			b.Type, b.Title, b.Endpoint, i+1,
		); err != nil {
			return fmt.Errorf("insert homepage block: %w", err)
		}
	}

	return tx.Commit()
}

// productRow is a product ready to be inserted, plus the discount range for its offers.
type productRow struct {
	Title, Slug, Platform, Region, Genre, Description, ReleaseDate string
	MSRPCents, SalesCount                                          int
	MinFactor, MaxFactor                                           float64
}

// storeName returns the name used in product titles for a platform.
func storeName(platform string) string {
	switch platform {
	case "PlayStation":
		return "PSN"
	case "Xbox":
		return "Xbox Live"
	case "Nintendo":
		return "Nintendo eShop"
	default:
		return platform // "Steam"
	}
}

// slugify turns "Baldur's Gate 3 PC Steam Global" into "baldur-s-gate-3-pc-steam-global".
// Every character that is not a-z or 0-9 becomes a dash, and repeated dashes collapse into one.
func slugify(s string) string {
	// Replace the accented letters our catalog uses: "Pokémon" -> "pokemon", "Ragnarök" -> "ragnarok".
	s = strings.NewReplacer("é", "e", "ö", "o").Replace(strings.ToLower(s))
	var b strings.Builder
	lastWasDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastWasDash = false
		} else if !lastWasDash {
			b.WriteRune('-')
			lastWasDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// randomKeyCode makes a fake key like "7KQ2M-XP9RT-B4ZHW".
// The alphabet has no 0/O or 1/I, so keys are easy to read (real stores do this too).
func randomKeyCode(rng *rand.Rand) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var b strings.Builder
	for group := 0; group < 3; group++ {
		if group > 0 {
			b.WriteByte('-')
		}
		for i := 0; i < 5; i++ {
			b.WriteByte(alphabet[rng.Intn(len(alphabet))])
		}
	}
	return b.String()
}
