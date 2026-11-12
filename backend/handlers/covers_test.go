package handlers

import (
	"reflect"
	"testing"
)

func TestCoverName(t *testing.T) {
	cases := map[string]string{
		"ELDEN RING (PC) Steam Key GLOBAL":                     "ELDEN RING",
		"Steam Gift Card 20 EUR Steam Key EUROPE":              "Steam Gift Card 20 EUR",
		"Xbox Game Pass Ultimate 1 Month Xbox Live Key GLOBAL": "Xbox Game Pass Ultimate 1 Month",
		"Nintendo eShop Card 25 EUR Nintendo eShop Key EUROPE": "Nintendo eShop Card 25 EUR",
	}
	for title, want := range cases {
		if got := coverName(title); got != want {
			t.Errorf("coverName(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestWrapWords(t *testing.T) {
	got := wrapWords("The Legend of Zelda: Tears of the Kingdom", 14, 4)
	want := []string{"The Legend of", "Zelda: Tears", "of the Kingdom"}
	// reflect.DeepEqual compares two slices element by element (== doesn't work on slices).
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	// Too long for 2 lines: cut and end with "…"
	got = wrapWords("one two three four five six", 7, 2)
	want = []string{"one two", "three…"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
