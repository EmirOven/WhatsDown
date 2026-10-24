package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDetailsMediaTab(dialog *adw.Dialog, mediaMsgs []store.StoredMessage) *gtk.Box {
	mediaBox := gtk.NewBox(gtk.OrientationVertical, 8)
	mediaBox.SetMarginTop(8)
	if len(mediaMsgs) == 0 {
		emptyGroup := adw.NewPreferencesGroup()
		emptyRow := adw.NewActionRow()
		emptyRow.SetTitle("No photos or videos")
		emptyRow.SetSubtitle("Shared photos and videos will appear here")
		emptyRow.AddPrefix(gtk.NewImageFromIconName("folder-pictures-symbolic"))
		emptyGroup.Add(emptyRow)
		mediaBox.Append(emptyGroup)
		return mediaBox
	}

	flowBox := gtk.NewFlowBox()
	flowBox.SetSelectionMode(gtk.SelectionNone)
	flowBox.SetHomogeneous(true)
	flowBox.SetMinChildrenPerLine(3)
	flowBox.SetMaxChildrenPerLine(3)
	flowBox.SetColumnSpacing(8)
	flowBox.SetRowSpacing(8)
	flowBox.AddCSSClass("media-gallery-flowbox")

	for _, m := range mediaMsgs {
		overlay := gtk.NewOverlay()
		overlay.AddCSSClass("media-tile")

		pic := gtk.NewPicture()
		pic.SetKeepAspectRatio(true)
		pic.SetCanShrink(true)
		pic.AddCSSClass("media-tile-picture")
		pic.SetSizeRequest(128, 128)
		media.LoadMediaPreview(c.client, m, pic, nil)
		overlay.SetChild(pic)

		if m.Kind == store.MessageKindVideo {
			badge := gtk.NewLabel("▶ Video")
			if e2e := media.ParseE2EMessage(m.Raw); e2e != nil && e2e.GetVideoMessage() != nil {
				if s := e2e.GetVideoMessage().GetSeconds(); s > 0 {
					badge.SetText(fmt.Sprintf("▶ %d:%02d", s/60, s%60))
				}
			}
			badge.AddCSSClass("media-video-badge")
			badge.SetHAlign(gtk.AlignStart)
			badge.SetVAlign(gtk.AlignEnd)
			badge.SetMarginStart(6)
			badge.SetMarginBottom(6)
			overlay.AddOverlay(badge)
		}

		mCopy := m
		leftClick := gtk.NewGestureClick()
		leftClick.SetButton(1)
		leftClick.ConnectPressed(func(nPress int, x, y float64) {
			if nPress == 1 {
				dialog.Close()
				c.openMediaViewer(mCopy)
			}
		})
		overlay.AddController(leftClick)
		c.attachMediaContextMenu(overlay, mCopy, dialog)
		flowBox.Append(overlay)
	}
	mediaBox.Append(flowBox)
	return mediaBox
}
