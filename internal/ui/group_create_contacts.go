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
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

type groupContactItem struct {
	JID         string
	DisplayName string
	Subtitle    string
	LastActive  time.Time
	Row         *adw.ActionRow
	Check       *gtk.CheckButton
}

func (c *MainController) collectSelectableContacts() []groupContactItem {
	seen := make(map[string]bool)
	var items []groupContactItem

	addCandidate := func(jid, name string, lastActive time.Time) {
		if !strings.HasSuffix(jid, "@s.whatsapp.net") || c.isSelfJID(jid) || seen[jid] {
			return
		}
		seen[jid] = true
		disp, sub := strings.TrimSpace(name), cleanPhoneNumber(jid)
		if disp == "" || disp == "WhatsApp User" {
			if disp = c.resolveContactName(jid); disp == "" {
				disp = sub
			}
		}
		items = append(items, groupContactItem{JID: jid, DisplayName: disp, Subtitle: sub, LastActive: lastActive})
	}

	for _, ch := range append(c.chats, c.archivedChats...) {
		addCandidate(ch.JID, formatDisplayName(ch), ch.LastMessageAt)
	}
	if c.store != nil {
		contacts, _ := c.store.ListAllContacts(context.Background())
		for _, cnt := range contacts {
			addCandidate(cnt.JID, cnt.Name, cnt.LastMessageAt)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].LastActive.Equal(items[j].LastActive) {
			return items[i].LastActive.After(items[j].LastActive)
		}
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
	if pixbuf, err := gdkpixbuf.NewPixbufFromFileAtScale(path, 640, 640, true); err == nil && pixbuf != nil {
		if data, err := pixbuf.SaveToBufferv("jpeg", []string{"quality"}, []string{"85"}); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	raw, err := os.ReadFile(path)
	if err == nil {
		if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
			var buf bytes.Buffer
			if jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}) == nil {
				return buf.Bytes(), nil
			}
		}
	}
	return raw, err
}
