package ui

func (ctrl *MainController) switchSidebarView(view string) {
	ctrl.sidebarStack.SetVisibleChildName(view)
	ctrl.chatsNavBtn.RemoveCSSClass("active")
	if ctrl.callsNavBtn != nil {
		ctrl.callsNavBtn.RemoveCSSClass("active")
	}
	ctrl.archivedNavBtn.RemoveCSSClass("active")
	ctrl.settingsNavBtn.RemoveCSSClass("active")

	switch view {
	case "chats":
		ctrl.chatsNavBtn.AddCSSClass("active")
	case "calls":
		if ctrl.callsNavBtn != nil {
			ctrl.callsNavBtn.AddCSSClass("active")
		}
	case "archived":
		ctrl.archivedNavBtn.AddCSSClass("active")
	case "settings":
		ctrl.settingsNavBtn.AddCSSClass("active")
	}

	if view == "settings" {
		ctrl.contentStack.SetVisibleChildName("settings")
	} else if ctrl.currentChat != nil {
		ctrl.contentStack.SetVisibleChildName("chat")
	} else {
		ctrl.contentStack.SetVisibleChildName("empty")
	}
}
