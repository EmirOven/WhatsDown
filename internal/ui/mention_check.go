package ui

import (
	"sort"
	"strings"
	"unicode"
)

func (c *MainController) checkMentionSuggestion() {
	if c.messageEntry == nil || c.currentChat == nil {
		c.hideMentionPopup()
		return
	}
	// Tagging should not be enabled in DMs (only in groups)
	if !strings.HasSuffix(c.currentChat.JID, "@g.us") {
		c.hideMentionPopup()
		return
	}

	buf := c.messageEntry.Buffer()
	if buf == nil {
		c.hideMentionPopup()
		return
	}

	insertMark := buf.GetInsert()
	cursorIter := buf.IterAtMark(insertMark)
	startIter := buf.StartIter()
	textBefore := buf.Text(startIter, cursorIter, false)

	lastAt := strings.LastIndex(textBefore, "@")
	if lastAt == -1 {
		c.hideMentionPopup()
		return
	}

	query := textBefore[lastAt+1:]
	for _, r := range query {
		if unicode.IsSpace(r) && r != ' ' {
			c.hideMentionPopup()
			return
		}
	}

	if lastAt > 0 {
		prevChar := rune(textBefore[lastAt-1])
		if !unicode.IsSpace(prevChar) && prevChar != '\n' && prevChar != '(' && prevChar != '[' {
			c.hideMentionPopup()
			return
		}
	}

	activeMentionState.atStartPos = lastAt
	c.filterAndShowMentions(query)
}

func (c *MainController) filterAndShowMentions(query string) {
	if len(activeMentionState.activeMembers) == 0 && c.currentChat != nil {
		activeMentionState.activeMembers = c.fetchGroupMembers(c.currentChat.JID)
	}

	type scoredItem struct {
		item  GroupMemberItem
		score int
	}

	var matches []scoredItem
	for _, m := range activeMentionState.activeMembers {
		nameMatch, s1 := FuzzyMatch(query, m.DisplayName)
		subMatch, s2 := FuzzyMatch(query, m.Subtitle)
		if nameMatch || subMatch {
			score := s1
			if s2 > score {
				score = s2
			}
			matches = append(matches, scoredItem{item: m, score: score})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	var filtered []GroupMemberItem
	for _, m := range matches {
		filtered = append(filtered, m.item)
	}

	activeMentionState.filteredMembers = filtered
	if len(filtered) == 0 {
		c.hideMentionPopup()
		return
	}

	c.showMentionPopup(filtered)
}
