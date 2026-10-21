package ui

func (ctrl *MainController) switchSidebarView(view string) {
	if ctrl.sidebarStack != nil {
		ctrl.sidebarStack.SetVisibleChildName(view)
	}

	if ctrl.chatsNavBtn != nil {
		ctrl.chatsNavBtn.RemoveCSSClass("active")
	}
	if ctrl.callsNavBtn != nil {
		ctrl.callsNavBtn.RemoveCSSClass("active")
	}
	if ctrl.archivedNavBtn != nil {
		ctrl.archivedNavBtn.RemoveCSSClass("active")
	}
	if ctrl.settingsNavBtn != nil {
		ctrl.settingsNavBtn.RemoveCSSClass("active")
	}

	switch view {
	case "chats":
		if ctrl.chatsNavBtn != nil {
			ctrl.chatsNavBtn.AddCSSClass("active")
		}
	case "calls":
		if ctrl.callsNavBtn != nil {
			ctrl.callsNavBtn.AddCSSClass("active")
		}
	case "archived":
		if ctrl.archivedNavBtn != nil {
			ctrl.archivedNavBtn.AddCSSClass("active")
		}
	case "settings":
		if ctrl.settingsNavBtn != nil {
			ctrl.settingsNavBtn.AddCSSClass("active")
		}
	}

	if ctrl.contentStack != nil {
		if view == "settings" {
			ctrl.contentStack.SetVisibleChildName("settings")
		} else if ctrl.currentChat != nil {
			ctrl.contentStack.SetVisibleChildName("chat")
		} else {
			ctrl.contentStack.SetVisibleChildName("empty")
		}
	}
}
