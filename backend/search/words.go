package search

import (
	"strings"
	"unicode"
)

// This file turns text into the words we compare: normalizing, extra word forms and nicknames.

// accents maps accented letters in our catalog to plain ones, so "pokemon" finds "Pokémon".
var accents = strings.NewReplacer("é", "e", "è", "e", "ö", "o", "ü", "u", "ä", "a", "á", "a", "ó", "o", "í", "i")

// normalize lowercases text, removes accents and splits it into words.
// Everything that isn't a letter or digit separates words:
//
//	"Baldur's Gate 3 (PC)"  -> ["baldur", "s", "gate", "3", "pc"]
//	"Spider-Man 2"          -> ["spider", "man", "2"]
func normalize(text string) []string {
	text = accents.Replace(strings.ToLower(text))
	// strings.FieldsFunc splits wherever the function returns true.
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// ignoredTitleWords appear in (almost) every title, so matching them says nothing.
// Without this, a typo like "kay" would match "key" in all 80 products.
var ignoredTitleWords = map[string]bool{"key": true}

// romanNumerals: titles say "V", people type "5" (and the other way round).
// No "i" or "x": too often a normal letter ("Xbox Series X|S").
var romanNumerals = map[string]string{
	"ii": "2", "iii": "3", "iv": "4", "v": "5", "vi": "6", "vii": "7", "viii": "8", "ix": "9",
}

// titleWords returns every word a title can be matched by. Besides the normal words we add:
//   - joined neighbours: "spider man" also gives "spiderman", "baldur s" gives "baldurs",
//     because people often type names without spaces or apostrophes;
//   - numbers for Roman numerals: "v" also gives "5".
func titleWords(title string) []string {
	words := normalize(title)
	seen := map[string]bool{}
	var result []string
	add := func(w string) {
		if !seen[w] && !ignoredTitleWords[w] {
			seen[w] = true
			result = append(result, w)
		}
	}

	for i, w := range words {
		add(w)
		if number, ok := romanNumerals[w]; ok {
			add(number)
		}
		if i+1 < len(words) {
			add(w + words[i+1])
		}
	}
	return result
}

// aliases are nicknames gamers use. A query word that equals a key is replaced by its words.
// Only full query words are replaced, so "god" is never turned into "god of war".
var aliases = map[string]string{
	"gta":    "grand theft auto",
	"cod":    "call of duty",
	"bo6":    "black ops 6",
	"rdr":    "red dead redemption",
	"rdr2":   "red dead redemption 2",
	"bg3":    "baldurs gate 3",
	"gow":    "god of war",
	"totk":   "tears of the kingdom",
	"mk8":    "mario kart 8",
	"ff":     "final fantasy",
	"ff7":    "final fantasy vii",
	"re4":    "resident evil 4",
	"fifa":   "ea sports fc",
	"acnh":   "animal crossing new horizons",
	"cp2077": "cyberpunk 2077",
}

// expandAliases: ["gta", "5"] -> ["grand", "theft", "auto", "5"].
func expandAliases(words []string) []string {
	var result []string
	for _, w := range words {
		if expansion, ok := aliases[w]; ok {
			result = append(result, strings.Fields(expansion)...)
		} else {
			result = append(result, w)
		}
	}
	return result
}
