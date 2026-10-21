package ui

import "postbetasoftware.com/whatsdown/internal/store"

type chatRowItem struct {
	isCommunityHeader bool
	communityJID      string
	chat              store.Chat
}
