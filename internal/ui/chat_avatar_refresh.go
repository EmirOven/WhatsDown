package ui

import "postbetasoftware.com/whatsdown/internal/avatar"

// RefreshActiveChatAvatar reloads the avatar for the currently selected chat if it matches jidStr.
func (c *MainController) RefreshActiveChatAvatar(jidStr string) {
	if c == nil || c.currentChat == nil || c.activeChatAvatar == nil {
		return
	}
	if c.isSameChat(c.currentChat.JID, jidStr) {
		avatar.LoadAvatar(c.client, jidStr, c.currentChat.Name, c.activeChatAvatar)
	}
}
