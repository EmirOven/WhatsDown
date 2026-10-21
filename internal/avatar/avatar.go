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
func LoadAvatar(client ClientProvider, jid, displayName string, avatar *adw.Avatar, isCommunity ...bool) {
	if avatar == nil {
		return
	}
	avatar.SetText(displayName)
	avatar.SetShowInitials(true)

	isComm := len(isCommunity) > 0 && isCommunity[0]
	LoadTexture(client, jid, isComm, func(tex *gdk.Texture) {
		avatar.SetCustomImage(tex)
	})
}

// LoadTexture loads an avatar texture from cache or network.
func LoadTexture(client ClientProvider, jid string, isCommunity bool, onLoaded func(*gdk.Texture)) {
	if onLoaded == nil {
		return
	}
	if jid == "" || jid == "0@s.whatsapp.net" || strings.HasPrefix(jid, "0@") {
		onLoaded(nil)
		return
	}

	mu.RLock()
	if tex, ok := memoryCache[jid]; ok {
		mu.RUnlock()
		onLoaded(tex)
		return
	}
	if lastFailed, ok := negativeCache[jid]; ok {
		if time.Since(lastFailed) < 1*time.Hour {
			mu.RUnlock()
			onLoaded(nil)
			return
		}
	}
	mu.RUnlock()

	diskPath := DiskPath(jid)
	if data, err := os.ReadFile(diskPath); err == nil && len(data) > 0 {
		if texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data)); err == nil {
			mu.Lock()
			memoryCache[jid] = texture
			mu.Unlock()
			onLoaded(texture)
			return
		}
	}

	if client == nil || !client.IsConnected() {
		onLoaded(nil)
		return
	}

	mu.Lock()
	if inFlight[jid] {
		mu.Unlock()
		return
	}
	inFlight[jid] = true
	mu.Unlock()

	go fetchRemote(client, jid, isCommunity, onLoaded)
}
