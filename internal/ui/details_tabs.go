package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDetailsTabs(dialog *adw.Dialog, chat store.Chat, contentBox *gtk.Box) {
	ctx := context.Background()
	mediaMsgs, _ := c.store.ListMediaMessages(ctx, chat.JID)
	docMsgs, _ := c.store.ListDocMessages(ctx, chat.JID)
	audioMsgs, _ := c.store.ListAudioMessages(ctx, chat.JID)
	starredMsgs, _ := c.store.ListStarredMessages(ctx, chat.JID)

	stack := gtk.NewStack()
	stack.SetTransitionType(gtk.StackTransitionTypeSlideLeftRight)
	stack.SetInterpolateSize(true)

	stack.AddTitled(c.buildDetailsMediaTab(dialog, mediaMsgs), "media", fmt.Sprintf("Media (%d)", len(mediaMsgs)))
	stack.AddTitled(c.buildDetailsDocsTab(dialog, docMsgs), "docs", fmt.Sprintf("Docs (%d)", len(docMsgs)))
	stack.AddTitled(c.buildDetailsAudioTab(dialog, audioMsgs), "audio", fmt.Sprintf("Audio (%d)", len(audioMsgs)))
	stack.AddTitled(c.buildDetailsStarredTab(dialog, starredMsgs), "starred", fmt.Sprintf("Starred (%d)", len(starredMsgs)))

	switcher := gtk.NewStackSwitcher()
	switcher.SetStack(stack)
	switcher.SetHAlign(gtk.AlignCenter)
	switcher.AddCSSClass("details-tabs-switcher")
	contentBox.Append(switcher)
	contentBox.Append(stack)
}
