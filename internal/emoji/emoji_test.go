package emoji_test

import (
	"testing"

	"postbetasoftware.com/whatsdown/internal/emoji"
)

func TestSearch(t *testing.T) {
	defaults := emoji.Search("", 8)
	if len(defaults) != 8 {
		t.Fatalf("expected 8 default emojis, got %d", len(defaults))
	}
	if defaults[0] != "👍" {
		t.Errorf("expected first default emoji to be 👍, got %s", defaults[0])
	}

	wands := emoji.Search("wand", 8)
	if len(wands) == 0 {
		t.Fatalf("expected results for 'wand', got none")
	}
	if wands[0] != "🪄" {
		t.Errorf("expected top emoji for 'wand' to be 🪄, got %s", wands[0])
	}

	magics := emoji.Search("magic", 8)
	if len(magics) == 0 || magics[0] != "🪄" {
		t.Errorf("expected top emoji for 'magic' to be 🪄, got %+v", magics)
	}

	hearts := emoji.Search("hrt", 8)
	foundHeart := false
	for _, h := range hearts {
		if h == "❤️" {
			foundHeart = true
			break
		}
	}
	if !foundHeart {
		t.Errorf("expected ❤️ in results for fuzzy 'hrt', got %+v", hearts)
	}

	cats := emoji.Search("cat", 8)
	if len(cats) == 0 || (cats[0] != "🐈" && cats[0] != "🐱") {
		t.Errorf("expected top emoji for 'cat' to be 🐈 or 🐱, got %+v", cats)
	}

	none := emoji.Search("xyznonexistent123", 8)
	if len(none) != 0 {
		t.Errorf("expected 0 emojis for invalid query, got %d", len(none))
	}
}
