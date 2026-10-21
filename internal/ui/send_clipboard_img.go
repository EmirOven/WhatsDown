package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

func (c *MainController) readClipboardImage(clipboard *gdk.Clipboard) {
	fallbackStream := func() {
		clipboard.ReadAsync(context.Background(), []string{"image/png", "image/jpeg", "image/webp"}, int(glib.PriorityDefault), func(res gio.AsyncResulter) {
			_, stream, err := clipboard.ReadFinish(res)
			if err != nil || stream == nil {
				return
			}
			base := gio.BaseInputStream(stream)
			base.ReadBytesAsync(context.Background(), 20*1024*1024, int(glib.PriorityDefault), func(res gio.AsyncResulter) {
				defer base.Close(context.Background())
				b, err := base.ReadBytesFinish(res)
				if err != nil || b == nil || b.Size() == 0 {
					return
				}
				temp := filepath.Join(os.TempDir(), fmt.Sprintf("whatsdown_paste_%d.png", time.Now().UnixNano()))
				if err := os.WriteFile(temp, b.Data(), 0644); err == nil {
					glib.IdleAdd(func() { c.showSendAttachmentDialog(temp) })
				}
			})
		})
	}

	clipboard.ReadTextureAsync(context.Background(), func(res gio.AsyncResulter) {
		texturer, err := clipboard.ReadTextureFinish(res)
		if err == nil && texturer != nil {
			if tex := gdk.BaseTexture(texturer); tex != nil {
				temp := filepath.Join(os.TempDir(), fmt.Sprintf("whatsdown_paste_%d.png", time.Now().UnixNano()))
				if tex.SaveToPNG(temp) {
					glib.IdleAdd(func() { c.showSendAttachmentDialog(temp) })
					return
				}
			}
		}
		fallbackStream()
	})
}
