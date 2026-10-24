package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Caption suggestions use the same mention metadata as the message composer.
func (c *MainController) attachCaptionMentions(entry *gtk.Entry, box *gtk.Box) {
	if c.currentChat == nil || !strings.HasSuffix(c.currentChat.JID, "@g.us") {
		return
	}
	suggestions := gtk.NewBox(gtk.OrientationVertical, 2)
	box.Append(suggestions)
	members := c.fetchGroupMembers(c.currentChat.JID)
	activeMentionState.activeMembers = members
	entry.ConnectChanged(func() {
		for child := suggestions.FirstChild(); child != nil; child = suggestions.FirstChild() {
			suggestions.Remove(child)
		}
		text := []rune(entry.Text())
		pos := entry.Position()
		if pos < 0 || pos > len(text) {
			pos = len(text)
		}
		prefix := string(text[:pos])
		at := strings.LastIndex(prefix, "@")
		if at < 0 || (at > 0 && prefix[at-1] != ' ' && prefix[at-1] != '\n') {
			return
		}
		query := strings.ToLower(prefix[at+1:])
		count := 0
		for _, member := range members {
			if !strings.Contains(strings.ToLower(member.DisplayName), query) {
				continue
			}
			member := member
			button := gtk.NewButtonWithLabel("@" + member.DisplayName)
			button.AddCSSClass("flat")
			button.ConnectClicked(func() {
				tag := "@" + member.DisplayName + " "
				activeMentionState.insertedMentions[member.DisplayName] = member.JID
				entry.SetText(prefix[:at] + tag + string(text[pos:]))
				entry.SetPosition(utf8.RuneCountInString(prefix[:at] + tag))
				entry.GrabFocus()
			})
			suggestions.Append(button)
			count++
			if count == 5 {
				break
			}
		}
	})
}
