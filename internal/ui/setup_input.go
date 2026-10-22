package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/markdown"
)

func (ctrl *MainController) bindInputWidgets(builder *gtk.Builder, messageEntry *gtk.TextView, sendButton *gtk.Button) {
	ctrl.messageEntry = messageEntry
	ctrl.sendButton = sendButton
	ctrl.messageEntryScrolled, _ = castObject[*gtk.ScrolledWindow](builder, "message_entry_scrolled")
	ctrl.messagePlaceholder, _ = castObject[*gtk.Label](builder, "message_placeholder")

	ctrl.replyPreviewBar, _ = castObject[*gtk.Box](builder, "reply_preview_bar")
	ctrl.replyPreviewSender, _ = castObject[*gtk.Label](builder, "reply_preview_sender")
	ctrl.replyPreviewText, _ = castObject[*gtk.Label](builder, "reply_preview_text")
	ctrl.cancelReplyBtn, _ = castObject[*gtk.Button](builder, "cancel_reply_btn")

	ctrl.markdownPreviewBar, _ = castObject[*gtk.Box](builder, "markdown_preview_bar")
	ctrl.markdownPreviewLabel, _ = castObject[*gtk.Label](builder, "markdown_preview_label")
	ctrl.emojiSuggestionBar, _ = castObject[*gtk.Box](builder, "emoji_suggestion_bar")

	ctrl.editBannerBar, _ = castObject[*gtk.Box](builder, "edit_banner_bar")
	ctrl.editBannerText, _ = castObject[*gtk.Label](builder, "edit_banner_text")
	ctrl.cancelEditBtn, _ = castObject[*gtk.Button](builder, "cancel_edit_btn")

	ctrl.sendButton.ConnectClicked(func() {
		ctrl.sendMessage()
	})
	if ctrl.cancelReplyBtn != nil {
		ctrl.cancelReplyBtn.ConnectClicked(func() {
			ctrl.cancelReply()
		})
	}
	if ctrl.cancelEditBtn != nil {
		ctrl.cancelEditBtn.ConnectClicked(func() {
			ctrl.cancelEditing()
		})
	}

	if buf := ctrl.messageEntry.Buffer(); buf != nil {
		buf.ConnectChanged(func() {
			text := ctrl.getMessageText()
			if ctrl.messagePlaceholder != nil {
				ctrl.messagePlaceholder.SetVisible(text == "")
			}
			ctrl.updateMessageEntryHeight()
			ctrl.checkEmojiSuggestion()
			ctrl.checkMentionSuggestion()

			if targetPath, ok := parseFilePathFromText(text); ok {
				ctrl.clearMessageText()
				glib.IdleAdd(func() {
					ctrl.showSendAttachmentDialog(targetPath)
				})
				return
			}

			if markdown.HasWhatsAppMarkdown(text) && strings.TrimSpace(text) != "" {
				previewMarkup, _ := markdown.FormatForBubble(text, nil, nil)
				if ctrl.markdownPreviewLabel != nil {
					ctrl.markdownPreviewLabel.SetMarkup(previewMarkup)
				}
				if ctrl.markdownPreviewBar != nil {
					ctrl.markdownPreviewBar.SetVisible(true)
				}
			} else {
				if ctrl.markdownPreviewBar != nil {
					ctrl.markdownPreviewBar.SetVisible(false)
				}
			}
		})
	}
	ctrl.updateMessageEntryHeight()
}
