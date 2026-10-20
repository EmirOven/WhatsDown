package main

import (
	"sort"
	"strings"
	"unicode"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type scoredEmoji struct {
	char  string
	score int
}

var popularEmojiBonus = map[string]int{
	"❤️":  200,
	"👍":  200,
	"😂":  200,
	"🔥":  200,
	"🎉":  200,
	"🙏":  200,
	"✨":  200,
	"⭐":  150,
	"👀":  150,
	"💯":  150,
	"😭":  150,
	"🤣":  150,
	"😍":  150,
	"😊":  150,
	"🥰":  150,
	"🥺":  120,
	"👏":  120,
	"💀":  120,
	"🪄":  180,
	"😎":  120,
	"🤔":  120,
	"😘":  120,
	"✔️":  100,
	"✅":  100,
	"💕":  100,
	"💔":  100,
	"💩":  100,
	"🐈":  120,
	"🐱":  120,
	"🐕":  120,
	"🐶":  120,
}

// scoreEmojiMatch scores how well the query matches an emoji's official name and aliases.
// Higher score indicates a better fit. Returns <= 0 if no match.
func scoreEmojiMatch(query, name string) int {
	q := strings.ToLower(query)
	n := strings.ToLower(name)
	if q == "" {
		return 0
	}

	// 1. Exact match on full name
	if n == q {
		return 2000
	}

	words := strings.FieldsFunc(n, func(r rune) bool {
		return r == ' ' || r == ':' || r == '-' || r == '_' || r == ','
	})

	// 2. Exact word match (e.g. "wand" in "magic wand")
	for _, w := range words {
		if w == q {
			return 1500 - len(w)*2
		}
	}

	// 3. Word prefix match (e.g. "wan" in "magic wand")
	bestPrefixScore := 0
	for _, w := range words {
		if strings.HasPrefix(w, q) {
			s := 1000 + (len(q) * 10) - (len(w) * 2)
			if s > bestPrefixScore {
				bestPrefixScore = s
			}
		}
	}
	if bestPrefixScore > 0 {
		return bestPrefixScore
	}

	// 4. Fuzzy match within a single word (e.g. "hrt" in "heart")
	qRunes := []rune(q)
	bestWordFuzzy := 0
	for _, w := range words {
		wRunes := []rune(w)
		qi := 0
		prevMatch := -2
		bonus := 0
		for i, c := range wRunes {
			if qi < len(qRunes) && c == qRunes[qi] {
				if i == 0 {
					bonus += 40 // matched first letter of word
				}
				if i == prevMatch+1 {
					bonus += 25 // consecutive letter bonus
				}
				prevMatch = i
				qi++
			}
		}
		if qi == len(qRunes) {
			score := 800 + bonus - (len(w)-len(q))*5
			if score > bestWordFuzzy {
				bestWordFuzzy = score
			}
		}
	}
	if bestWordFuzzy > 0 {
		return bestWordFuzzy
	}

	// 5. Word substring match (e.g. "and" in "magic wand")
	for _, w := range words {
		if strings.Contains(w, q) {
			return 500 - len(w)*2
		}
	}

	// 6. Fuzzy subsequence match across the full name
	qi := 0
	score := 200
	prevMatch := -2
	nRunes := []rune(n)
	for i, c := range nRunes {
		if qi < len(qRunes) && c == qRunes[qi] {
			if i == prevMatch+1 {
				score += 15 // consecutive match bonus
			}
			if i == 0 || nRunes[i-1] == ' ' || nRunes[i-1] == ':' || nRunes[i-1] == '_' || nRunes[i-1] == '-' {
				score += 25 // word start bonus
			}
			prevMatch = i
			qi++
		}
	}
	if qi == len(qRunes) {
		return score - len(n)
	}

	return -1
}

// searchEmojis finds up to maxResults emojis matching the query string,
// ranked from best fitting to worst fitting.
func searchEmojis(query string, maxResults int) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		defaultTop := []string{"👍", "❤️", "😂", "🔥", "🎉", "🙏", "🤔", "✨"}
		if len(defaultTop) > maxResults {
			return defaultTop[:maxResults]
		}
		return defaultTop
	}

	var matches []scoredEmoji
	seen := make(map[string]bool)

	for _, item := range allUnicodeEmojis {
		s := scoreEmojiMatch(query, item.Name)
		if s > 0 && !seen[item.Char] {
			seen[item.Char] = true
			if bonus, ok := popularEmojiBonus[item.Char]; ok {
				s += bonus
			}
			matches = append(matches, scoredEmoji{char: item.Char, score: s})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	var results []string
	for i := 0; i < len(matches) && i < maxResults; i++ {
		results = append(results, matches[i].char)
	}
	return results
}

// checkEmojiSuggestion inspects the message buffer for an active colon query.
func (c *whatsDownMainController) checkEmojiSuggestion() {
	if c.messageEntry == nil {
		return
	}
	buf := c.messageEntry.Buffer()
	if buf == nil {
		return
	}

	insertMark := buf.GetInsert()
	cursorIter := buf.IterAtMark(insertMark)
	startIter := buf.StartIter()
	textBefore := buf.Text(startIter, cursorIter, false)

	lastColon := strings.LastIndex(textBefore, ":")
	if lastColon == -1 {
		c.hideEmojiSuggestions()
		return
	}

	query := textBefore[lastColon+1:]
	for _, r := range query {
		if unicode.IsSpace(r) {
			c.hideEmojiSuggestions()
			return
		}
	}

	if lastColon > 0 {
		prevChar := rune(textBefore[lastColon-1])
		if !unicode.IsSpace(prevChar) && prevChar != '\n' && prevChar != '(' && prevChar != '[' {
			c.hideEmojiSuggestions()
			return
		}
	}

	c.emojiColonStartPos = lastColon
	emojis := searchEmojis(query, 8)
	if len(emojis) == 0 {
		c.hideEmojiSuggestions()
		return
	}

	c.showEmojiSuggestions(emojis)
}

// showEmojiSuggestions populates and renders the horizontal emoji recommendation bar.
func (c *whatsDownMainController) showEmojiSuggestions(emojis []string) {
	if c.emojiSuggestionBar == nil {
		return
	}
	c.emojiSuggestions = emojis
	if c.emojiSuggestionIndex >= len(emojis) {
		c.emojiSuggestionIndex = 0
	}
	if c.emojiSuggestionIndex < 0 {
		c.emojiSuggestionIndex = 0
	}

	for child := c.emojiSuggestionBar.FirstChild(); child != nil; {
		next := gtk.BaseWidget(child).NextSibling()
		c.emojiSuggestionBar.Remove(child)
		child = next
	}

	for i, e := range emojis {
		idx := i
		emojiChar := e
		btn := gtk.NewButton()
		btn.SetLabel(emojiChar)
		btn.AddCSSClass("flat")
		btn.AddCSSClass("emoji-suggestion-item")
		if idx == c.emojiSuggestionIndex {
			btn.AddCSSClass("selected")
		}
		btn.ConnectClicked(func() {
			c.emojiSuggestionIndex = idx
			c.commitSelectedEmoji()
		})
		c.emojiSuggestionBar.Append(btn)
	}

	c.emojiSuggestionBar.SetVisible(true)
}

// hideEmojiSuggestions conceals the suggestion bar and resets the selection index.
func (c *whatsDownMainController) hideEmojiSuggestions() {
	if c.emojiSuggestionBar != nil {
		c.emojiSuggestionBar.SetVisible(false)
	}
	c.emojiSuggestions = nil
	c.emojiSuggestionIndex = 0
}

// isEmojiSuggestionVisible returns true if the suggestion bar is currently visible.
func (c *whatsDownMainController) isEmojiSuggestionVisible() bool {
	return c.emojiSuggestionBar != nil && c.emojiSuggestionBar.Visible() && len(c.emojiSuggestions) > 0
}

// navigateEmojiSuggestion shifts the highlighted emoji left or right.
func (c *whatsDownMainController) navigateEmojiSuggestion(delta int) {
	n := len(c.emojiSuggestions)
	if n == 0 {
		return
	}
	c.emojiSuggestionIndex = (c.emojiSuggestionIndex + delta + n) % n
	c.renderEmojiSuggestionSelection()
}

// renderEmojiSuggestionSelection updates the visual selection state among suggestion buttons.
func (c *whatsDownMainController) renderEmojiSuggestionSelection() {
	if c.emojiSuggestionBar == nil {
		return
	}
	idx := 0
	for child := c.emojiSuggestionBar.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		if btn, ok := child.(*gtk.Button); ok {
			if idx == c.emojiSuggestionIndex {
				btn.AddCSSClass("selected")
			} else {
				btn.RemoveCSSClass("selected")
			}
		}
		idx++
	}
}

// commitSelectedEmoji replaces the active :query string with the selected emoji.
func (c *whatsDownMainController) commitSelectedEmoji() {
	if len(c.emojiSuggestions) == 0 || c.emojiSuggestionIndex < 0 || c.emojiSuggestionIndex >= len(c.emojiSuggestions) {
		c.hideEmojiSuggestions()
		return
	}
	selectedEmoji := c.emojiSuggestions[c.emojiSuggestionIndex]

	if c.messageEntry != nil {
		buf := c.messageEntry.Buffer()
		if buf != nil {
			insertMark := buf.GetInsert()
			cursorIter := buf.IterAtMark(insertMark)
			startIter := buf.StartIter()
			textBefore := buf.Text(startIter, cursorIter, false)

			lastColon := strings.LastIndex(textBefore, ":")
			if lastColon != -1 && lastColon <= int(cursorIter.Offset()) {
				colonIter := buf.IterAtOffset(lastColon)
				buf.Delete(colonIter, cursorIter)
				buf.InsertAtCursor(selectedEmoji)
			} else {
				buf.InsertAtCursor(selectedEmoji)
			}
		}
	}

	c.hideEmojiSuggestions()
	if c.messageEntry != nil {
		c.messageEntry.GrabFocus()
	}
}
