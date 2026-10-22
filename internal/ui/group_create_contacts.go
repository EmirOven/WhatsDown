package ui

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"sort"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

type groupContactItem struct {
	JID         string
	DisplayName string
	Subtitle    string
	Row         *adw.ActionRow
	Check       *gtk.CheckButton
}

func (c *MainController) collectSelectableContacts() []groupContactItem {
	seen := make(map[string]bool)
	var items []groupContactItem

	addCandidate := func(jid, name string) {
		if !strings.HasSuffix(jid, "@s.whatsapp.net") || c.isSelfJID(jid) || seen[jid] {
			return
		}
		seen[jid] = true
		disp := strings.TrimSpace(name)
		if disp == "" || disp == "WhatsApp User" {
			disp = c.resolveContactName(jid)
		}
		sub := cleanPhoneNumber(jid)
		if disp == "" {
			disp = sub
		}
		items = append(items, groupContactItem{JID: jid, DisplayName: disp, Subtitle: sub})
	}

	for _, ch := range c.chats {
		addCandidate(ch.JID, formatDisplayName(ch))
	}
	for _, ch := range c.archivedChats {
		addCandidate(ch.JID, formatDisplayName(ch))
	}
	if c.store != nil {
		if contacts, err := c.store.ListAllContacts(context.Background()); err == nil {
			for _, cnt := range contacts {
				addCandidate(cnt.JID, cnt.Name)
			}
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(items[i].DisplayName) < strings.ToLower(items[j].DisplayName)
	})
	return items
}

func (c *MainController) buildContactRow(item *groupContactItem) *adw.ActionRow {
	row := adw.NewActionRow()
	row.SetTitle(item.DisplayName)
	if item.Subtitle != "" && item.Subtitle != item.DisplayName {
		row.SetSubtitle(item.Subtitle)
	}
	av := adw.NewAvatar(36, item.DisplayName, true)
	avatar.LoadAvatar(c.client, item.JID, item.DisplayName, av)
	row.AddPrefix(av)

	chk := gtk.NewCheckButton()
	chk.SetVAlign(gtk.AlignCenter)
	row.AddSuffix(chk)
	row.SetActivatableWidget(chk)
	item.Row, item.Check = row, chk
	return row
}

func loadAsJPEG(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err == nil {
			return buf.Bytes(), nil
		}
	}
	return raw, nil
}
