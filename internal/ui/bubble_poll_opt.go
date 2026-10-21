package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildPollOptionRow(
	msg store.StoredMessage,
	opt string,
	allowMultiple bool,
	initialVotes map[string]bool,
	initialResults *store.PollResults,
	totalVoters int,
	optWidgets map[string]*pollOptWidget,
	totalVotesLbl *gtk.Label,
	pollBox *gtk.Box,
) {
	optCopy := opt
	optBtn := gtk.NewButton()
	optBtn.AddCSSClass("poll-option-btn")

	optCol := gtk.NewBox(gtk.OrientationVertical, 4)
	rowBox := gtk.NewBox(gtk.OrientationHorizontal, 8)

	isSelected := initialVotes[optCopy]
	iconName := "radio-symbolic"
	if allowMultiple {
		if isSelected {
			iconName = "checkbox-checked-symbolic"
		} else {
			iconName = "checkbox-symbolic"
		}
	} else if isSelected {
		iconName = "radio-checked-symbolic"
	}
	indicator := gtk.NewImageFromIconName(iconName)
	rowBox.Append(indicator)

	optLbl := gtk.NewLabel(optCopy)
	optLbl.SetXAlign(0)
	optLbl.SetHExpand(true)
	optLbl.SetWrap(true)
	rowBox.Append(optLbl)

	cnt := 0
	if initialResults != nil {
		cnt = initialResults.VoteCounts[optCopy]
	}
	cText := "0"
	if cnt == 1 {
		cText = "1 vote"
	} else if cnt > 1 {
		cText = fmt.Sprintf("%d votes", cnt)
	}
	countLbl := gtk.NewLabel(cText)
	countLbl.AddCSSClass("poll-vote-count")
	rowBox.Append(countLbl)

	optCol.Append(rowBox)

	pBar := gtk.NewProgressBar()
	pBar.AddCSSClass("poll-progress-bar")
	var frac float64
	if totalVoters > 0 {
		frac = float64(cnt) / float64(totalVoters)
	}
	pBar.SetFraction(frac)
	optCol.Append(pBar)

	optBtn.SetChild(optCol)
	if isSelected {
		optBtn.AddCSSClass("selected")
	}

	optWidgets[optCopy] = &pollOptWidget{
		btn:         optBtn,
		indicator:   indicator,
		countLbl:    countLbl,
		progressBar: pBar,
	}

	optBtn.ConnectClicked(func() {
		c.handlePollVote(msg, optCopy, allowMultiple, optWidgets, totalVotesLbl)
	})

	pollBox.Append(optBtn)
}
