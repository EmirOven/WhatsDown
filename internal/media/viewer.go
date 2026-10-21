package media

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/store"
)

func OpenFile(path string) error {
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	if err := gio.AppInfoLaunchDefaultForURI(uri, nil); err == nil {
		return nil
	}
	cmd := exec.Command("xdg-open", path)
	return cmd.Start()
}

func OpenURL(rawURL string) error {
	if err := gio.AppInfoLaunchDefaultForURI(rawURL, nil); err == nil {
		return nil
	}
	return exec.Command("xdg-open", rawURL).Start()
}

func FormatFileSize(bytes uint64) string {
	if bytes == 0 {
		return ""
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func OpenMedia(client Downloader, msg store.StoredMessage, onStatus func(string)) {
	waMsg := ParseE2EMessage(msg.Raw)
	if waMsg != nil && waMsg.GetLocationMessage() != nil {
		loc := waMsg.GetLocationMessage()
		lat := loc.GetDegreesLatitude()
		long := loc.GetDegreesLongitude()
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%f,%f", lat, long)
		_ = OpenURL(mapURL)
		return
	}

	if onStatus != nil {
		onStatus("Downloading…")
	}

	go func() {
		path, err := DownloadMedia(context.Background(), client, msg)
		if err != nil {
			log.Printf("opening media error: %v", err)
			glib.IdleAdd(func() {
				if onStatus != nil {
					onStatus("Failed to open")
				}
			})
			return
		}

		glib.IdleAdd(func() {
			if onStatus != nil {
				onStatus("Opened")
			}
			_ = OpenFile(path)
		})
	}()
}
