package ui

import "postbetasoftware.com/whatsdown/internal/store"

type chatRowItem struct {
	folderID          string
	isFolderHeader    bool
	isLooseDropZone   bool
	isCommunityHeader bool
	communityJID      string
	chat              store.Chat
}
