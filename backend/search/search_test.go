package search

import "testing"

// A slice of the real catalog (same titles as the seed data), enough to test realistic typos.
var testDocs = []Document{
	{1, "Elden Ring (PC) Steam Key GLOBAL", 9200},
	{2, "Elden Ring (PS5) PSN Key EUROPE", 7360},
	{3, "Cyberpunk 2077 (PC) Steam Key GLOBAL", 8100},
	{4, "The Witcher 3: Wild Hunt Complete Edition (PC) Steam Key GLOBAL", 7400},
	{5, "Grand Theft Auto V (PC) Steam Key GLOBAL", 9900},
	{6, "Baldur's Gate 3 (PC) Steam Key GLOBAL", 8600},
	{7, "Hogwarts Legacy (PC) Steam Key GLOBAL", 7900},
	{8, "Marvel's Spider-Man 2 (PS5) PSN Key EUROPE", 6300},
	{9, "Pokémon Scarlet (Switch) Nintendo eShop Key EUROPE", 6000},
	{10, "The Legend of Zelda: Tears of the Kingdom (Switch) Nintendo eShop Key EUROPE", 7100},
	{11, "Red Dead Redemption 2 (PC) Steam Key GLOBAL", 8800},
	{12, "Call of Duty: Black Ops 6 (PC) Steam Key GLOBAL", 8000},
	{13, "Minecraft (Xbox One) Xbox Live Key GLOBAL", 9500},
	{14, "God of War Ragnarök (PS5) PSN Key EUROPE", 6700},
	{15, "Steam Gift Card 20 EUR Steam Key EUROPE", 7700},
	{16, "Diablo IV (PS5) PSN Key EUROPE", 4800},
	{17, "Stardew Valley (PC) Steam Key GLOBAL", 7300},
	{18, "Starfield (PC) Steam Key GLOBAL", 5200},
	{19, "Final Fantasy VII Rebirth (PS5) PSN Key EUROPE", 4600},
	{20, "Mario Kart 8 Deluxe (Switch) Nintendo eShop Key EUROPE", 7600},
}

// The typo table: what people really type -> the product that must come FIRST.
func TestSearchFindsTheRightProductFirst(t *testing.T) {
	idx := NewIndex(testDocs)

	cases := []struct {
		query  string
		wantID int64
		why    string
	}{
		{"elden ring", 1, "exact; PC version first because it's more popular"},
		{"ELDEN RING", 1, "case doesn't matter"},
		{"eldn ring", 1, "missing letter"},
		{"elden rign", 1, "swapped letters in a 4-letter word"},
		{"wticher", 4, "swapped letters"},
		{"witcher 3", 4, "partial name"},
		{"cyberpnk", 3, "missing letter in a long word"},
		{"cybrpnk", 3, "two missing letters in a 7-letter word"},
		{"cyber", 3, "prefix while typing"},
		{"hogwats legacy", 7, "missing letter"},
		{"hogwrt", 7, "half-typed AND a typo"},
		{"bladurs gate", 6, "swap + no apostrophe (matches joined 'baldurs')"},
		{"baldurs gate 3", 6, "no apostrophe"},
		{"spiderman", 8, "no hyphen (matches joined 'spiderman')"},
		{"spider man", 8, "words split"},
		{"pokemon", 9, "no accent"},
		{"pokémon scarlet", 9, "with accent"},
		{"ragnarok", 14, "no umlaut"},
		{"zelda tears", 10, "words from the middle of the title"},
		{"tears of the kingdom", 10, "subtitle only"},
		{"gta 5", 5, "nickname + number for Roman V"},
		{"gta v", 5, "nickname + Roman numeral"},
		{"grand theft auto", 5, "full name"},
		{"rdr2", 11, "nickname"},
		{"red dead", 11, "partial"},
		{"cod black ops", 12, "nickname"},
		{"bg3", 6, "nickname"},
		{"diablo 4", 16, "number for Roman IV"},
		{"ff7", 19, "nickname with numeral"},
		{"minecraf", 13, "prefix"},
		{"mincraft", 13, "missing letter"},
		{"mario kart", 20, "two words"},
		{"steam gift card", 15, "gift card"},
		{"  elden   ring  ", 1, "extra spaces"},
		{"elden-ring!!", 1, "punctuation"},
	}

	for _, c := range cases {
		results := idx.Search(c.query, 0)
		if len(results) == 0 {
			t.Errorf("%q (%s): no results, want product %d", c.query, c.why, c.wantID)
			continue
		}
		if results[0].ID != c.wantID {
			t.Errorf("%q (%s): first result is %d, want %d", c.query, c.why, results[0].ID, c.wantID)
		}
	}
}

// Things that must NOT match: a search that returns everything is as useless as one that returns nothing.
func TestSearchDoesNotMatchNonsense(t *testing.T) {
	idx := NewIndex(testDocs)

	cases := []struct {
		query string
		why   string
	}{
		{"xyzzy", "nothing like it"},
		{"kay", "would match 'key' in every title if 'key' were indexed"},
		{"god of wor ragnarok fortnite", "one word (fortnite) doesn't match: AND logic"},
		{"cat", "3-letter words must be exact"},
		{"", "empty query"},
		{"!!!", "only punctuation"},
	}
	for _, c := range cases {
		if results := idx.Search(c.query, 0); len(results) != 0 {
			t.Errorf("%q (%s): got %d results, want 0", c.query, c.why, len(results))
		}
	}
}

func TestSearchFiltersAndLimits(t *testing.T) {
	idx := NewIndex(testDocs)

	// "star" is a prefix of "stardew" and "starfield": both, but nothing else.
	results := idx.Search("star", 0)
	if len(results) != 2 {
		t.Fatalf("'star' gave %d results, want 2", len(results))
	}

	// Both Elden Ring versions match equally; the more popular (PC, 9200) must come first.
	results = idx.Search("elden", 0)
	if len(results) != 2 || results[0].ID != 1 || results[1].ID != 2 {
		t.Errorf("'elden' = %v, want [1 2]", results)
	}

	// The limit cuts the list.
	if got := len(idx.Search("steam", 3)); got != 3 {
		t.Errorf("limit 3 gave %d", got)
	}

	// Exact beats typo: "elden" matches 'elden' exactly, so it scores higher than a 1-typo match.
	exact := idx.Search("elden ring", 1)[0].Score
	typo := idx.Search("eldn ring", 1)[0].Score
	if exact <= typo {
		t.Errorf("exact score %d should beat typo score %d", exact, typo)
	}
}

func TestClosestSuggestsSomethingSensible(t *testing.T) {
	idx := NewIndex(testDocs)

	// Too many typos for Search ("ded" is 3 letters, so it must be exact)...
	if len(idx.Search("red ded redemtion", 0)) != 0 {
		t.Skip("search became more tolerant; pick another example")
	}
	// ...but Closest still finds what was meant.
	if id, ok := idx.Closest("red ded redemtion"); !ok || id != 11 {
		t.Errorf("Closest = %d, %v; want 11, true", id, ok)
	}

	// Random letters shouldn't produce a confident suggestion.
	if id, ok := idx.Closest("qwrtzpx"); ok {
		t.Errorf("Closest(qwrtzpx) suggested %d", id)
	}
}

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"elden", "elden", 0},
		{"eldn", "elden", 1},      // insert
		{"eldenn", "elden", 1},    // delete
		{"eldan", "elden", 1},     // replace
		{"wticher", "witcher", 1}, // swap
		{"kitten", "sitting", 3},  // the textbook example
		{"pokémon", "pokemon", 1}, // é is one letter (rune), not two bytes
		{"cybrpnk", "cyberpunk", 2},
	}
	for _, c := range cases {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestNormalizeAndTitleWords(t *testing.T) {
	got := normalize("Marvel's Spider-Man 2 (PS5)")
	want := []string{"marvel", "s", "spider", "man", "2", "ps5"}
	if len(got) != len(want) {
		t.Fatalf("normalize = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalize = %q, want %q", got, want)
		}
	}

	words := titleWords("Grand Theft Auto V (PC) Steam Key GLOBAL")
	has := map[string]bool{}
	for _, w := range words {
		has[w] = true
	}
	for _, w := range []string{"grand", "v", "5", "grandtheft", "steam"} {
		if !has[w] {
			t.Errorf("titleWords is missing %q (got %q)", w, words)
		}
	}
	if has["key"] {
		t.Error("'key' should be ignored")
	}
}
