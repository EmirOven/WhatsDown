package ui

func (ctrl *MainController) bindChatSearch() {
	if ctrl.chatSearchBtn == nil || ctrl.chatSearchBar == nil || ctrl.chatSearchEntry == nil {
		return
	}
	ctrl.chatSearchBtn.SetSensitive(false)
	ctrl.chatSearchBar.ConnectEntry(ctrl.chatSearchEntry)
	ctrl.updateSearchUI()

	ctrl.chatSearchBtn.ConnectToggled(func() {
		active := ctrl.chatSearchBtn.Active()
		ctrl.chatSearchBar.SetSearchMode(active)
		if active {
			ctrl.chatSearchEntry.GrabFocus()
		} else {
			ctrl.chatSearchEntry.SetText("")
			ctrl.messagesSearchFilter = ""
			ctrl.searchMatchIndices = nil
			ctrl.currentSearchMatchIndex = 0
			ctrl.updateSearchUI()
		}
	})

	ctrl.chatSearchEntry.ConnectSearchChanged(func() {
		ctrl.performMessageSearch(ctrl.chatSearchEntry.Text())
	})

	ctrl.chatSearchEntry.ConnectActivate(func() {
		ctrl.navigateSearchMatch(1)
		ctrl.chatSearchEntry.GrabFocus()
	})

	if ctrl.chatSearchNextBtn != nil {
		ctrl.chatSearchNextBtn.ConnectClicked(func() {
			ctrl.navigateSearchMatch(1)
			ctrl.chatSearchEntry.GrabFocus()
		})
	}
	if ctrl.chatSearchPrevBtn != nil {
		ctrl.chatSearchPrevBtn.ConnectClicked(func() {
			ctrl.navigateSearchMatch(-1)
			ctrl.chatSearchEntry.GrabFocus()
		})
	}
}

func (ctrl *MainController) bindScrollAdjustment() {
	if ctrl.scrollToBottomBtn != nil {
		ctrl.scrollToBottomBtn.ConnectClicked(func() {
			ctrl.autoScrollBottom = true
			ctrl.scrollToBottomAnimated()
		})
	}

	if ctrl.messagesScrolledWindow == nil {
		return
	}
	adj := ctrl.messagesScrolledWindow.VAdjustment()
	if adj == nil {
		return
	}
	adj.ConnectValueChanged(func() {
		maxVal := adj.Upper() - adj.PageSize()
		isAtBottom := maxVal <= 10 || adj.Value() >= maxVal-30
		if !ctrl.isAutoScrolling {
			if maxVal > 10 && adj.Value() < maxVal-40 {
				ctrl.autoScrollBottom = false
			} else if isAtBottom {
				ctrl.autoScrollBottom = true
			}
		}
		if ctrl.scrollToBottomBtn != nil {
			if ctrl.isAutoScrolling || isAtBottom {
				ctrl.scrollToBottomBtn.SetVisible(false)
			} else {
				ctrl.scrollToBottomBtn.SetVisible(true)
			}
		}
	})
	adj.ConnectChanged(func() {
		if ctrl.autoScrollBottom || ctrl.isAutoScrolling {
			ctrl.scrollToBottom()
		}
	})
	adj.NotifyProperty("page-size", func() {
		if ctrl.autoScrollBottom || ctrl.isAutoScrolling {
			ctrl.scrollToBottom()
		}
	})
	adj.NotifyProperty("upper", func() {
		if ctrl.autoScrollBottom || ctrl.isAutoScrolling {
			ctrl.scrollToBottom()
		}
	})
}
