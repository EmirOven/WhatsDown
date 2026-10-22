package ui

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

func (ctrl *MainController) bindPopoverWidgets(builder *gtk.Builder) {
	ctrl.attachButton, _ = castObject[*gtk.MenuButton](builder, "attach_button")
	ctrl.attachPopover, _ = castObject[*gtk.Popover](builder, "attach_popover")
	ctrl.attachPhotoBtn, _ = castObject[*gtk.Button](builder, "attach_photo_btn")
	ctrl.attachDocBtn, _ = castObject[*gtk.Button](builder, "attach_doc_btn")
	ctrl.attachPollBtn, _ = castObject[*gtk.Button](builder, "attach_poll_btn")

	ctrl.emojiButton, _ = castObject[*gtk.MenuButton](builder, "emoji_button")
	ctrl.emojiChooser, _ = castObject[*gtk.EmojiChooser](builder, "emoji_chooser")

	ctrl.stickerButton, _ = castObject[*gtk.MenuButton](builder, "sticker_button")
	ctrl.stickerPopover, _ = castObject[*gtk.Popover](builder, "sticker_popover")
	ctrl.stickerFlowbox, _ = castObject[*gtk.FlowBox](builder, "sticker_flowbox")
	ctrl.stickerEmptyBox, _ = castObject[*gtk.Box](builder, "sticker_empty_box")
	ctrl.stickerScrolledWindow, _ = castObject[*gtk.ScrolledWindow](builder, "sticker_scrolled_window")
	ctrl.addStickerBtn, _ = castObject[*gtk.Button](builder, "add_sticker_btn")

	if ctrl.attachPhotoBtn != nil {
		ctrl.attachPhotoBtn.ConnectClicked(func() {
			if ctrl.attachPopover != nil {
				ctrl.attachPopover.Popdown()
			}
			ctrl.openFileChooserForMedia(true)
		})
	}
	if ctrl.attachDocBtn != nil {
		ctrl.attachDocBtn.ConnectClicked(func() {
			if ctrl.attachPopover != nil {
				ctrl.attachPopover.Popdown()
			}
			ctrl.openFileChooserForMedia(false)
		})
	}
	if ctrl.attachPollBtn != nil {
		ctrl.attachPollBtn.ConnectClicked(func() {
			if ctrl.attachPopover != nil {
				ctrl.attachPopover.Popdown()
			}
			ctrl.showCreatePollDialog()
		})
	}
	if ctrl.emojiChooser != nil {
		ctrl.emojiChooser.ConnectEmojiPicked(func(text string) {
			ctrl.insertEmoji(text)
		})
	}
	if ctrl.addStickerBtn != nil {
		ctrl.addStickerBtn.ConnectClicked(func() {
			ctrl.openFileChooserForSticker()
		})
	}
	if ctrl.stickerPopover != nil {
		ctrl.stickerPopover.ConnectShow(func() {
			ctrl.loadSavedStickersUI()
		})
		ctrl.stickerPopover.ConnectClosed(func() {
			// Release decoded textures and per-item menu callbacks when hidden.
			if ctrl.stickerFlowbox != nil {
				ctrl.releaseMessageWidget(ctrl.stickerFlowbox)
				ctrl.stickerFlowbox.RemoveAll()
			}
		})
	}
	ctrl.loadSavedStickersUI()
}
