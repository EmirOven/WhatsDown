package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/markdown"
)

func appendEditHistoryCard(mainBox *gtk.Box, tagText, timeText, msgText string, isLatest bool) {
	card := gtk.NewBox(gtk.OrientationVertical, 8)
	card.AddCSSClass("card")
	card.AddCSSClass("edit-history-card")
	if isLatest {
		card.AddCSSClass("edit-history-card-latest")
	}
	card.SetMarginBottom(8)

	topRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	tag := gtk.NewLabel(tagText)
	tag.AddCSSClass("caption")
	if isLatest {
		tag.AddCSSClass("accent")
	} else {
		tag.AddCSSClass("dim-label")
	}
	tag.SetHExpand(true)
	tag.SetXAlign(0)

	timeLbl := gtk.NewLabel(timeText)
	timeLbl.AddCSSClass("caption")
	timeLbl.AddCSSClass("dim-label")
	timeLbl.SetXAlign(1)

	topRow.Append(tag)
	topRow.Append(timeLbl)
	card.Append(topRow)

	markup, _ := markdown.FormatForBubble(msgText, nil, nil)
	textLbl := gtk.NewLabel("")
	textLbl.SetWrap(true)
	textLbl.SetWrapMode(pango.WrapWordChar)
	textLbl.SetXAlign(0)
	textLbl.SetSelectable(true)
	if _, _, _, err := pango.ParseMarkup(markup, 0); err == nil {
		textLbl.SetMarkup(markup)
	} else {
		textLbl.SetText(msgText)
	}
	card.Append(textLbl)
	mainBox.Append(card)
}
