package avatar

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ClientProvider provides profile picture information.
type ClientProvider interface {
	IsConnected() bool
	GetProfilePictureInfo(ctx context.Context, jid types.JID, params *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error)
}

// LoadAvatar sets the fallback initials and asynchronously fetches/applies the
// profile picture for the given JID to the provided Adw.Avatar.
func LoadAvatar(client ClientProvider, jid, displayName string, avatar *adw.Avatar) {
	if avatar == nil {
		return
	}

	avatar.SetText(displayName)
	avatar.SetShowInitials(true)

	if jid == "" || jid == "0@s.whatsapp.net" || strings.HasPrefix(jid, "0@") {
		avatar.SetCustomImage(nil)
		return
	}

	mu.RLock()
	if tex, ok := memoryCache[jid]; ok {
		mu.RUnlock()
		avatar.SetCustomImage(tex)
		return
	}
	if lastFailed, ok := negativeCache[jid]; ok {
		if time.Since(lastFailed) < 1*time.Hour {
			mu.RUnlock()
			avatar.SetCustomImage(nil)
			return
		}
	}
	mu.RUnlock()

	diskPath := DiskPath(jid)
	if data, err := os.ReadFile(diskPath); err == nil && len(data) > 0 {
		texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data))
		if err == nil {
			mu.Lock()
			memoryCache[jid] = texture
			mu.Unlock()
			avatar.SetCustomImage(texture)
			return
		}
	}

	if client == nil || !client.IsConnected() {
		avatar.SetCustomImage(nil)
		return
	}

	mu.Lock()
	if inFlight[jid] {
		mu.Unlock()
		return
	}
	inFlight[jid] = true
	mu.Unlock()

	go fetchRemote(client, jid, avatar)
}
