package emoji

import (
	"strings"
	"sync"
)

var (
	singleEmojiSet     map[string]struct{}
	singleEmojiNoVSSet map[string]struct{}
	singleEmojiOnce    sync.Once
)

func initSingleEmojiSets() {
	all := All()
	singleEmojiSet = make(map[string]struct{}, len(all))
	singleEmojiNoVSSet = make(map[string]struct{}, len(all))

	for _, item := range all {
		singleEmojiSet[item.Char] = struct{}{}
		noVS := stripVS(item.Char)
		singleEmojiNoVSSet[noVS] = struct{}{}
	}
}

func stripVS(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0xFE0E || r == 0xFE0F {
			return -1
		}
		return r
	}, s)
}

func stripSkinTones(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0x1F3FB && r <= 0x1F3FF {
			return -1
		}
		return r
	}, s)
}

// IsSingleEmoji reports whether the given text consists solely of a single emoji.
// It handles simple emojis, emojis with variation selectors, skin tone modifiers,
// flag sequences (regional indicators), keycaps, and zero-width joiner (ZWJ) sequences.
func IsSingleEmoji(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	singleEmojiOnce.Do(initSingleEmojiSets)

	// Check catalog directly
	if _, ok := singleEmojiSet[s]; ok {
		return true
	}

	// Strip skin tones and test
	noSkin := stripSkinTones(s)
	if _, ok := singleEmojiSet[noSkin]; ok {
		return true
	}

	// Strip variation selectors and test
	noVS := stripVS(s)
	if _, ok := singleEmojiSet[noVS]; ok {
		return true
	}

	// Strip both skin tones and variation selectors
	noSkinNoVS := stripVS(noSkin)
	if _, ok := singleEmojiNoVSSet[noSkinNoVS]; ok {
		return true
	}

	// Fallback to structural emoji verification for newly added or unlisted emojis
	return isSingleEmojiStructural(s)
}

func isSingleEmojiStructural(s string) bool {
	runes := []rune(s)
	if len(runes) == 0 {
		return false
	}

	// Flag: exactly 2 Regional Indicator runes
	if len(runes) == 2 && isRegionalIndicator(runes[0]) && isRegionalIndicator(runes[1]) {
		return true
	}

	// Keycap: ('0'-'9' or '#' or '*') + (optional VS) + 0x20E3
	if (runes[0] >= '0' && runes[0] <= '9') || runes[0] == '#' || runes[0] == '*' {
		if len(runes) == 2 && runes[1] == 0x20E3 {
			return true
		}
		if len(runes) == 3 && (runes[1] == 0xFE0E || runes[1] == 0xFE0F) && runes[2] == 0x20E3 {
			return true
		}
		return false
	}

	// Tag flag sequence: e.g. Black Flag 🏴 (0x1F3F4) + tag characters (0xE0020-0xE007E) + cancel tag (0xE007F)
	if runes[0] == 0x1F3F4 && len(runes) > 2 && runes[len(runes)-1] == 0xE007F {
		allTags := true
		for _, r := range runes[1 : len(runes)-1] {
			if r < 0xE0020 || r > 0xE007E {
				allTags = false
				break
			}
		}
		if allTags {
			return true
		}
	}

	// General ZWJ emoji sequence: Component (ZWJ Component)*
	// where Component is: BaseEmoji Modifier*
	var components [][]rune
	var current []rune
	for _, r := range runes {
		if r == 0x200D {
			if len(current) == 0 {
				return false
			}
			components = append(components, current)
			current = nil
		} else {
			current = append(current, r)
		}
	}
	if len(current) == 0 {
		return false
	}
	components = append(components, current)

	for _, comp := range components {
		base := comp[0]
		if !isBaseEmojiRune(base) {
			return false
		}
		for _, mod := range comp[1:] {
			if !isEmojiModifierRune(mod) {
				return false
			}
		}
	}
	return true
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

func isEmojiModifierRune(r rune) bool {
	return (r >= 0x1F3FB && r <= 0x1F3FF) || // Fitzpatrick skin tone
		r == 0xFE0E || r == 0xFE0F || // Variation selectors
		r == 0x20E3 || // Combining enclosing keycap
		(r >= 0xE0020 && r <= 0xE007F) // Tag characters
}

func isBaseEmojiRune(r rune) bool {
	switch {
	case r >= 0x1F600 && r <= 0x1F64F: // Emoticons
		return true
	case r >= 0x1F300 && r <= 0x1F5FF: // Misc Symbols and Pictographs
		return true
	case r >= 0x1F680 && r <= 0x1F6FF: // Transport and Map
		return true
	case r >= 0x1F700 && r <= 0x1F7FF: // Geometric Shapes Extended
		return true
	case r >= 0x1F800 && r <= 0x1F8FF: // Supplemental Arrows-C
		return true
	case r >= 0x1F900 && r <= 0x1F9FF: // Supplemental Symbols and Pictographs
		return true
	case r >= 0x1FA00 && r <= 0x1FA6F: // Symbols and Pictographs Extended-B
		return true
	case r >= 0x1FA70 && r <= 0x1FAFF: // Symbols and Pictographs Extended-A
		return true
	case r >= 0x2600 && r <= 0x26FF: // Misc Symbols (☕, ✈, ⚡, ❄, etc.)
		return true
	case r >= 0x2700 && r <= 0x27BF: // Dingbats (✨, ❌, etc.)
		return true
	case r >= 0x2300 && r <= 0x23FF: // Misc Technical (⏰, ⌛, etc.)
		return true
	case r >= 0x2B00 && r <= 0x2BFF: // Misc Symbols and Arrows (⭐, etc.)
		return true
	case r >= 0x1F000 && r <= 0x1F02F: // Mahjong Tiles
		return true
	case r >= 0x1F0A0 && r <= 0x1F0FF: // Playing Cards
		return true
	case r == 0x203C || r == 0x2049 || r == 0x2122 || r == 0x2139 || r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299:
		return true
	case (r >= 0x2194 && r <= 0x2199) || r == 0x21A9 || r == 0x21AA:
		return true
	case r >= 0x25AA && r <= 0x25AB || (r >= 0x25B6 && r <= 0x25C0) || (r >= 0x25FB && r <= 0x25FE):
		return true
	case r >= 0x2934 && r <= 0x2935:
		return true
	case r == 0x00A9 || r == 0x00AE: // Copyright, Registered
		return true
	default:
		return false
	}
}
