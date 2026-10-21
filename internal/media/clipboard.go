package media

import (
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/store"
)

func CopyMediaToClipboard(client Downloader, msg store.StoredMessage) {
	go func() {
		path, err := DownloadMedia(context.Background(), client, msg)
		if err != nil {
			log.Printf("copyMediaToClipboard error: %v", err)
			return
		}
		glib.IdleAdd(func() {
			clipboard := gdk.DisplayGetDefault().Clipboard()
			if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".webp") {
				if tex, err := gdk.NewTextureFromFilename(path); err == nil && tex != nil {
					clipboard.SetTexture(tex)
					return
				}
			}
			clipboard.SetText(path)
		})
	}()
}

func ShowMediaInFileManager(client Downloader, msg store.StoredMessage) {
	go func() {
		path, err := DownloadMedia(context.Background(), client, msg)
		if err != nil {
			log.Printf("showMediaInFileManager error: %v", err)
			return
		}
		glib.IdleAdd(func() {
			if _, err := exec.LookPath("nautilus"); err == nil {
				_ = exec.Command("nautilus", "--select", path).Start()
				return
			}
			dir := filepath.Dir(path)
			_ = exec.Command("xdg-open", dir).Start()
		})
	}()
}
