package avatar

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// NewCommunityAvatar creates a square avatar widget for a WhatsApp community.
func NewCommunityAvatar(client ClientProvider, jid, name string, size int) *gtk.Box {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.SetSizeRequest(size, size)
	box.SetOverflow(gtk.OverflowHidden)
	box.AddCSSClass("community-avatar")

	stack := gtk.NewStack()
	stack.SetSizeRequest(size, size)
	stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	stack.SetTransitionDuration(150)

	fallbackBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	fallbackBox.SetSizeRequest(size, size)
	fallbackBox.AddCSSClass("community-avatar-fallback")

	lbl := gtk.NewLabel(getCommunityInitials(name))
	lbl.AddCSSClass("community-avatar-initials")
	lbl.SetHAlign(gtk.AlignCenter)
	lbl.SetVAlign(gtk.AlignCenter)
	lbl.SetHExpand(true)
	lbl.SetVExpand(true)
	fallbackBox.Append(lbl)
	stack.AddNamed(fallbackBox, "fallback")

	img := gtk.NewImage()
	img.SetPixelSize(size)
	img.SetSizeRequest(size, size)
	stack.AddNamed(img, "image")
	stack.SetVisibleChildName("fallback")

	box.Append(stack)

	LoadTexture(client, jid, true, func(tex *gdk.Texture) {
		glib.IdleAdd(func() {
			if tex != nil {
				img.SetFromPaintable(tex)
				stack.SetVisibleChildName("image")
			} else {
				stack.SetVisibleChildName("fallback")
			}
		})
	})

	return box
}

func getCommunityInitials(name string) string {
	runes := []rune(strings.TrimSpace(name))
	if len(runes) == 0 {
		return "👥"
	}
	words := strings.Fields(string(runes))
	if len(words) >= 2 {
		r1 := []rune(words[0])
		r2 := []rune(words[1])
		if len(r1) > 0 && len(r2) > 0 {
			return strings.ToUpper(string(r1[0]) + string(r2[0]))
		}
	}
	if len(runes) >= 2 {
		return strings.ToUpper(string(runes[:2]))
	}
	return strings.ToUpper(string(runes[:1]))
}
