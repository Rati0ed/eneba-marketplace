package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"net/http"
	"strings"
)

// platformColors are the badge colors shown at the bottom of each cover.
var platformColors = map[string]string{
	"Steam":       "#1b2838",
	"PlayStation": "#0070d1",
	"Xbox":        "#107c10",
	"Nintendo":    "#e60012",
}

// Cover answers GET /api/covers/{file}, e.g. /api/covers/elden-ring-pc-steam-global.svg.
// Add ?layout=wide for a 1200x400 homepage banner version.
// It DRAWS a placeholder cover as an SVG image: a colored gradient + the game name +
// a platform badge. We use these instead of real game art, which we have no rights to.
func (h *Handler) Cover(w http.ResponseWriter, r *http.Request) {
	// Go's router matches whole path segments, so {file} is "elden-ring-...-global.svg".
	slug, isSVG := strings.CutSuffix(r.PathValue("file"), ".svg")
	if !isSVG {
		writeError(w, http.StatusNotFound, "cover not found")
		return
	}

	var title, platform string
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT title, platform FROM products WHERE slug = ?`, slug,
	).Scan(&title, &platform)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "cover not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	// Each product gets its own color: hash the slug into a number, then take 0-359
	// as a hue on the color wheel. The same slug always gives the same color.
	hasher := fnv.New32a()
	hasher.Write([]byte(slug))
	hue := hasher.Sum32() % 360

	// Build the text lines. html.EscapeString turns characters like < & ' into safe
	// codes (&lt; &amp; &#39;), so a title can never inject markup into the SVG.
	var textLines strings.Builder
	for i, line := range wrapWords(coverName(title), 14, 4) {
		fmt.Fprintf(&textLines, `<text x="24" y="%d">%s</text>`, 70+i*35, html.EscapeString(line))
	}

	badgeColor := platformColors[platform]
	if badgeColor == "" {
		badgeColor = "#333333"
	}

	// The inside of a 300x400 portrait cover: gradient, name, platform badge.
	cover := fmt.Sprintf(`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
<stop offset="0" stop-color="hsl(%d, 70%%, 45%%)"/><stop offset="1" stop-color="#1a0b3d"/>
</linearGradient></defs>
<rect width="300" height="400" fill="url(#g)"/>
<g font-family="Arial, Helvetica, sans-serif" font-size="27" font-weight="700" fill="#ffffff">%s</g>
<rect x="0" y="352" width="300" height="48" fill="%s"/>
<text x="24" y="383" font-family="Arial, Helvetica, sans-serif" font-size="18" font-weight="700" fill="#ffffff">%s</text>`,
		hue, textLines.String(), badgeColor, html.EscapeString(strings.ToUpper(platform)))

	var svg string
	if r.URL.Query().Get("layout") == "wide" {
		// Homepage banner, 1200x400: a dark gradient with two soft circles, and the portrait
		// cover placed on the right (a nested <svg> scales it). The left side stays empty
		// because the frontend writes the banner's title and button there.
		svg = fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="400" viewBox="0 0 1200 400">
<defs><linearGradient id="bg" x1="0" y1="0" x2="1" y2="0">
<stop offset="0" stop-color="#1a0b3d"/><stop offset="1" stop-color="hsl(%d, 65%%, 35%%)"/>
</linearGradient></defs>
<rect width="1200" height="400" fill="url(#bg)"/>
<circle cx="980" cy="200" r="260" fill="hsl(%d, 70%%, 50%%)" opacity="0.18"/>
<circle cx="760" cy="420" r="180" fill="#4618ac" opacity="0.35"/>
<svg x="880" y="50" width="225" height="300" viewBox="0 0 300 400">%s</svg>
</svg>`, hue, hue, cover)
	} else {
		svg = `<svg xmlns="http://www.w3.org/2000/svg" width="300" height="400" viewBox="0 0 300 400">` + cover + `</svg>`
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	// Covers never change while the server runs, so browsers may cache them for a day.
	w.Header().Set("Cache-Control", "public, max-age=86400")
	// Extra safety if someone opens the SVG directly: forbid scripts inside it.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.Write([]byte(svg))
}

// coverName shortens a product title to the part worth printing on the cover:
//
//	"ELDEN RING (PC) Steam Key GLOBAL"        -> "ELDEN RING"
//	"Steam Gift Card 20 EUR Steam Key EUROPE" -> "Steam Gift Card 20 EUR"
func coverName(title string) string {
	if name, _, found := strings.Cut(title, " ("); found {
		return name // games: everything before "(PC)"
	}
	// Gift cards: drop " Key REGION", then drop the repeated store name before it.
	name, _, _ := strings.Cut(title, " Key ")
	for _, store := range []string{" Nintendo eShop", " Xbox Live", " PSN", " Steam"} {
		if trimmed, found := strings.CutSuffix(name, store); found {
			return trimmed
		}
	}
	return name
}

// wrapWords splits text into lines of at most maxChars characters (without
// breaking words), and keeps at most maxLines lines, adding "…" if text was cut.
func wrapWords(text string, maxChars, maxLines int) []string {
	var lines []string
	current := ""
	for _, word := range strings.Fields(text) {
		if current == "" {
			current = word
		} else if len(current)+1+len(word) <= maxChars {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] += "…"
	}
	return lines
}
