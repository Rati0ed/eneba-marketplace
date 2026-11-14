// Package search is our typo-tolerant product search. It works fully in memory:
// the catalog is small (80 products), so we index all titles at startup and score every
// product for every query. No database and no external search engine.
//
// How a query is matched (see Search):
//  1. normalize: "Baldur's Gate 3!" -> ["baldur", "s", "gate", "3"]
//  2. expand nicknames: "gta 5" -> ["grand", "theft", "auto", "5"]
//  3. every query word must match some word of the title (AND), via one of:
//     exact word > prefix > word with 1-2 typos > prefix with a typo
//  4. sum the points per product, sort best first; ties -> more popular product first.
//  5. drop weak matches that score under half of the best one.
package search

import (
	"sort"
	"strings"
)

// Document is what we index: one product.
type Document struct {
	ID         int64
	Title      string
	Popularity int // sales count, used to break ties
}

// Result is one matching product and how well it matched (higher = better).
type Result struct {
	ID    int64
	Score int
}

// Index holds every product with its title already split into searchable words.
type Index struct {
	docs []indexedDoc
}

type indexedDoc struct {
	Document
	normalizedTitle string   // "elden ring pc steam global" (for the starts-with bonus)
	words           []string // every word we can match against, incl. extra forms (see titleWords)
}

// Points for each kind of word match. Exact beats prefix beats typo.
const (
	scoreExact         = 10 // "elden" = "elden"
	scorePrefix        = 7  // "cyber" -> "cyberpunk" (user is still typing)
	scoreOneTypo       = 5  // "eldn" -> "elden"
	scoreTwoTypos      = 3  // "cybrpnk" -> "cyberpunk"
	scoreTypoPrefix    = 2  // "hogwrt" -> "hogwarts" (still typing AND a typo)
	scoreTitleStart    = 5  // bonus: the whole query is the start of the title
	minFuzzyPrefixRune = 4  // typo+prefix only for 4+ letter words, otherwise too many false matches
)

// NewIndex prepares documents for searching. Called once at startup.
func NewIndex(docs []Document) *Index {
	idx := &Index{}
	for _, d := range docs {
		idx.docs = append(idx.docs, indexedDoc{
			Document:        d,
			normalizedTitle: strings.Join(normalize(d.Title), " "),
			words:           titleWords(d.Title),
		})
	}
	return idx
}

// Search returns matching products, best first. limit <= 0 means "all matches".
func (idx *Index) Search(query string, limit int) []Result {
	queryWords := expandAliases(normalize(query))
	if len(queryWords) == 0 {
		return nil
	}
	normalizedQuery := strings.Join(queryWords, " ")

	var results []Result
	popularity := map[int64]int{} // for the tie-break sort below

	for _, doc := range idx.docs {
		total := 0
		matchedAll := true
		for _, qw := range queryWords {
			best := bestWordScore(qw, doc.words)
			if best == 0 {
				matchedAll = false // AND: one unmatched word rejects the product
				break
			}
			total += best
		}
		if !matchedAll {
			continue
		}
		if strings.HasPrefix(doc.normalizedTitle, normalizedQuery) {
			total += scoreTitleStart
		}
		results = append(results, Result{ID: doc.ID, Score: total})
		popularity[doc.ID] = doc.Popularity
	}

	// Highest score first; equal scores -> more popular first; then by id so the order is stable.
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if popularity[a.ID] != popularity[b.ID] {
			return popularity[a.ID] > popularity[b.ID]
		}
		return a.ID < b.ID
	})

	// Drop weak guesses when there are strong matches: if "star" matches "Stardew" and
	// "Starfield" perfectly, "Scarlet" (1 typo away from "star") is just noise.
	// Anything scoring under half of the best result is removed.
	if len(results) > 0 {
		cutoff := results[0].Score
		kept := results[:0] // reuse the same memory; we only ever remove items
		for _, r := range results {
			if r.Score*2 >= cutoff {
				kept = append(kept, r)
			}
		}
		results = kept
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// Closest finds the product the user most likely MEANT when Search found nothing.
// Used for "Did you mean ...?". ok=false if nothing is reasonably close.
//
// It only runs when there are zero results, so it can be much more forgiving than Search:
//   - short words may have a typo too ("gpa" ~ "gta"),
//   - nicknames are matched with a typo as well ("rdr3" ~ "rdr2", "cdo" ~ "cod"),
//   - words typed without vowels match the consonants of a title word ("mncrft" ~ "minecraft").
//
// Each product gets a COST: for every query word, how many edits it is away from the
// best-matching title word (or nickname). The cheapest product wins, if the cost is at most
// half the number of letters typed. Ties: a nickname match wins, then the more popular product.
func (idx *Index) Closest(query string) (id int64, ok bool) {
	// No alias expansion here: closestWordCost matches nicknames itself, typos included.
	queryWords := normalize(query)
	if len(queryWords) == 0 {
		return 0, false
	}
	queryLetters := 0
	for _, qw := range queryWords {
		queryLetters += len([]rune(qw))
	}

	found := false
	bestCost, bestPopularity, bestViaAlias := 0, 0, false
	for _, doc := range idx.docs {
		cost, viaAlias := 0, false
		for _, qw := range queryWords {
			c, alias := closestWordCost(qw, doc.words)
			cost += c
			viaAlias = viaAlias || alias
		}

		better := !found ||
			cost < bestCost ||
			(cost == bestCost && viaAlias && !bestViaAlias) || // "re5" -> Resident Evil 4 (nickname), not "Red" Dead
			(cost == bestCost && viaAlias == bestViaAlias && doc.Popularity > bestPopularity)
		if better {
			found, id = true, doc.ID
			bestCost, bestPopularity, bestViaAlias = cost, doc.Popularity, viaAlias
		}
	}

	// At most one edit per two letters typed; otherwise it's a guess, not a suggestion.
	return id, found && bestCost*2 <= queryLetters
}

// closestWordCost: how many edits is ONE query word away from this product?
// It tries three ways and keeps the cheapest. viaAlias reports whether a nickname won.
func closestWordCost(queryWord string, titleWords []string) (cost int, viaAlias bool) {
	cost = len([]rune(queryWord)) // worst case: every letter is wrong

	// 1. Plain edit distance to each title word (short words included: 1 typo in "gya" is fine here).
	for _, tw := range titleWords {
		cost = min(cost, editDistance(queryWord, tw))
	}

	// 2. Typed without vowels: "mncrft" vs the consonants of "minecraft" ("mncrft") = 0 edits.
	if !hasVowel(queryWord) {
		for _, tw := range titleWords {
			if skeleton := consonants(tw); len(skeleton) >= 2 {
				cost = min(cost, editDistance(queryWord, skeleton))
			}
		}
	}

	// 3. Nicknames with a typo: "gpa" is 1 edit from "gta" -> compare "grand theft auto" instead.
	//    Cost = typos in the nickname + how far each of its words is from this title.
	for nickname, expansion := range aliases {
		allowed := 0
		if len(nickname) >= 3 {
			allowed = 1 // "ff" or "gow" must be exact or nearly; 2-letter nicknames exactly
		}
		d := editDistance(queryWord, nickname)
		if d > allowed {
			continue
		}
		c := d
		for _, word := range strings.Fields(expansion) {
			best := len([]rune(word))
			for _, tw := range titleWords {
				best = min(best, editDistance(word, tw))
			}
			c += best
		}
		// On a tie with a plain word match, count it as a nickname match (stronger signal).
		if c <= cost {
			cost, viaAlias = c, true
		}
	}
	return cost, viaAlias
}

// hasVowel: "mncrft" -> false, "minecraft" -> true.
func hasVowel(word string) bool {
	return strings.ContainsAny(word, "aeiou")
}

// consonants removes vowels: "minecraft" -> "mncrft", "pokemon" -> "pkmn".
func consonants(word string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune("aeiou", r) {
			return -1 // -1 = drop this character
		}
		return r
	}, word)
}

// bestWordScore = the best way ONE query word matches ANY title word (0 = no match).
func bestWordScore(queryWord string, titleWords []string) int {
	best := 0
	qLen := len([]rune(queryWord))
	allowed := allowedTypos(qLen)

	for _, tw := range titleWords {
		score := 0
		switch {
		case tw == queryWord:
			score = scoreExact
		case strings.HasPrefix(tw, queryWord):
			score = scorePrefix
		case allowed > 0:
			// Full word with typos: "wticher" vs "witcher".
			if d := editDistance(queryWord, tw); d <= allowed {
				if d == 1 {
					score = scoreOneTypo
				} else {
					score = scoreTwoTypos
				}
			} else if qLen >= minFuzzyPrefixRune && prefixDistance(queryWord, tw) <= allowed {
				// Still typing AND a typo: "hogwrt" vs the start of "hogwarts".
				score = scoreTypoPrefix
			}
		}
		if score > best {
			best = score
		}
	}
	return best
}

// allowedTypos: how many letter mistakes we forgive, based on word length.
// Short words must be exact: one wrong letter in "cod" is a third of the word ("god"? "cat"?).
//
//	1-3 letters -> 0,   4-6 letters -> 1,   7+ letters -> 2
func allowedTypos(length int) int {
	switch {
	case length <= 3:
		return 0
	case length <= 6:
		return 1
	default:
		return 2
	}
}

// prefixDistance compares a half-typed word with the START of a title word.
// We try title prefixes one letter shorter, equal and longer, because a typo can
// add or drop a letter: "hogwrt" vs "hogwar"(2), "hogwart"(1) -> 1.
func prefixDistance(queryWord, titleWord string) int {
	q, t := []rune(queryWord), []rune(titleWord)
	best := len(q)
	for n := len(q) - 1; n <= len(q)+1; n++ {
		if n <= 0 || n >= len(t) { // n >= len(t): that would be the whole word, already checked
			continue
		}
		if d := editDistance(queryWord, string(t[:n])); d < best {
			best = d
		}
	}
	return best
}

// editDistance counts the fewest single-letter edits to turn a into b:
// insert a letter, delete a letter, replace a letter, or SWAP two neighbouring letters.
// (Levenshtein + swaps = "optimal string alignment" / restricted Damerau-Levenshtein.)
//
//	editDistance("eldn", "elden")       = 1  (insert e)
//	editDistance("wticher", "witcher")  = 1  (swap t,i)
//	editDistance("kitten", "sitting")   = 3
//
// It fills a table where dist[i][j] = distance between the first i letters of a and the
// first j letters of b. Each cell is built from its neighbours, so the bottom-right cell
// is the answer. ("Dynamic programming": solve small sub-problems, reuse their answers.)
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b) // runes, not bytes, so "é" counts as one letter
	rows, cols := len(ra)+1, len(rb)+1

	dist := make([][]int, rows)
	for i := range dist {
		dist[i] = make([]int, cols)
		dist[i][0] = i // turning i letters into "" = i deletions
	}
	for j := 0; j < cols; j++ {
		dist[0][j] = j // turning "" into j letters = j insertions
	}

	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1 // replacing costs 1...
			if ra[i-1] == rb[j-1] {
				cost = 0 // ...unless the letters are already the same
			}
			dist[i][j] = min(
				dist[i-1][j]+1,      // delete a letter from a
				dist[i][j-1]+1,      // insert a letter into a
				dist[i-1][j-1]+cost, // replace (or keep) a letter
			)
			// Swap: "ti" vs "it" costs 1 instead of 2 replacements.
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				dist[i][j] = min(dist[i][j], dist[i-2][j-2]+1)
			}
		}
	}
	return dist[rows-1][cols-1]
}
