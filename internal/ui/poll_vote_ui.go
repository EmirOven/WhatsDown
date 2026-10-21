package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) updatePollOptionWidgets(
	widgets map[string]*pollOptWidget,
	totalVotesLbl *gtk.Label,
	userVotes map[string]bool,
	results *store.PollResults,
	allowMultiple bool,
) {
	totalVoters := 0
	if results != nil {
		totalVoters = results.TotalVoters
	}
	for name, w := range widgets {
		isSelected := userVotes[name]
		if isSelected {
			w.btn.AddCSSClass("selected")
			if allowMultiple {
				w.indicator.SetFromIconName("checkbox-checked-symbolic")
			} else {
				w.indicator.SetFromIconName("radio-checked-symbolic")
			}
		} else {
			w.btn.RemoveCSSClass("selected")
			if allowMultiple {
				w.indicator.SetFromIconName("checkbox-symbolic")
			} else {
				w.indicator.SetFromIconName("radio-symbolic")
			}
		}

		cnt := 0
		if results != nil {
			cnt = results.VoteCounts[name]
		}
		cText := "0"
		if cnt == 1 {
			cText = "1 vote"
		} else if cnt > 1 {
			cText = fmt.Sprintf("%d votes", cnt)
		}
		w.countLbl.SetText(cText)

		var frac float64
		if totalVoters > 0 {
			frac = float64(cnt) / float64(totalVoters)
		}
		w.progressBar.SetFraction(frac)
	}
	if totalVotesLbl != nil {
		if totalVoters == 0 {
			totalVotesLbl.SetText("No votes yet")
		} else if totalVoters == 1 {
			totalVotesLbl.SetText("1 person voted")
		} else {
			totalVotesLbl.SetText(fmt.Sprintf("%d people voted", totalVoters))
		}
	}
}
