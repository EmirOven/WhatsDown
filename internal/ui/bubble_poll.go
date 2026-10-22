package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildPollWidget(msg store.StoredMessage) *gtk.Box {
	pollBox := gtk.NewBox(gtk.OrientationVertical, 6)
	pollBox.AddCSSClass("chat-bubble-poll-box")

	headerBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	pollIcon := gtk.NewImageFromIconName("view-list-bullet-symbolic")
	headerBox.Append(pollIcon)

	question, options, selectableCount := extractPollDetails(msg.Raw)
	if question == "" || question == "Poll" {
		if msg.Text != "" {
			question = msg.Text
		} else {
			question = "Poll"
		}
	}

	qLbl := gtk.NewLabel(question)
	qLbl.AddCSSClass("heading")
	qLbl.SetWrap(true)
	qLbl.SetWrapMode(pango.WrapWordChar)
	qLbl.SetXAlign(0)
	headerBox.Append(qLbl)
	pollBox.Append(headerBox)

	allowMultiple := selectableCount > 1 || selectableCount == 0
	subText := "Select one"
	if allowMultiple {
		subText = "Select one or more"
	}
	subLbl := gtk.NewLabel(subText)
	addCSSClasses(subLbl, "caption", "dim-label")
	subLbl.SetXAlign(0)
	pollBox.Append(subLbl)

	voterJID := "me"
	if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
		voterJID = c.client.Store.ID.ToNonAD().String()
	}

	var initialVotes map[string]bool
	var initialResults *store.PollResults
	if c.store != nil {
		initialVotes, _ = c.store.GetPollVotes(context.Background(), msg.ChatJID, msg.ID, voterJID)
		initialResults, _ = c.store.GetPollResults(context.Background(), msg.ChatJID, msg.ID)
	}
	if initialVotes == nil {
		initialVotes = make(map[string]bool)
	}

	totalVoters := 0
	if initialResults != nil {
		totalVoters = initialResults.TotalVoters
	}

	optWidgets := make(map[string]*pollOptWidget)
	totalVotesLbl := gtk.NewLabel("No votes yet")
	addCSSClasses(totalVotesLbl, "caption", "dim-label")
	totalVotesLbl.SetHExpand(true)
	totalVotesLbl.SetXAlign(0)
	if totalVoters == 1 {
		totalVotesLbl.SetText("1 person voted")
	} else if totalVoters > 1 {
		totalVotesLbl.SetText(fmt.Sprintf("%d people voted", totalVoters))
	}

	for _, opt := range options {
		c.buildPollOptionRow(msg, opt, allowMultiple, initialVotes, initialResults, totalVoters, optWidgets, totalVotesLbl, pollBox)
	}

	summaryBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	summaryBox.AddCSSClass("poll-footer-box")
	summaryBox.SetMarginTop(4)

	viewVotesBtn := gtk.NewButtonWithLabel("View votes")
	addCSSClasses(viewVotesBtn, "flat", "poll-view-votes-btn", "pill")
	c.trackWidgetSignal(viewVotesBtn, viewVotesBtn, viewVotesBtn.ConnectClicked(func() {
		c.showPollVotesDialog(msg, question, options)
	}))

	summaryBox.Append(totalVotesLbl)
	summaryBox.Append(viewVotesBtn)
	pollBox.Append(summaryBox)

	return pollBox
}
