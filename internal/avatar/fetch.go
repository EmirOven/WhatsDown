package avatar

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func fetchRemote(client ClientProvider, jid string, isCommunity bool, onLoaded func(*gdk.Texture)) {
	defer func() {
		mu.Lock()
		delete(inFlight, jid)
		mu.Unlock()
	}()

	targetJID, err := types.ParseJID(jid)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	params := &whatsmeow.GetProfilePictureParams{
		Preview:     true,
		IsCommunity: isCommunity,
	}
	picInfo, err := client.GetProfilePictureInfo(ctx, targetJID, params)
	if err != nil && !isCommunity && strings.HasSuffix(jid, "@g.us") {
		params.IsCommunity = true
		picInfo, err = client.GetProfilePictureInfo(ctx, targetJID, params)
	}
	if err != nil {
		if errors.Is(err, whatsmeow.ErrProfilePictureNotSet) || errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
			mu.Lock()
			negativeCache[jid] = time.Now()
			mu.Unlock()
		} else {
			log.Printf("fetching profile picture info for %s: %v", jid, err)
		}
		return
	}

	if picInfo == nil || picInfo.URL == "" {
		mu.Lock()
		negativeCache[jid] = time.Now()
		mu.Unlock()
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, picInfo.URL, nil)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("downloading avatar for %s: %v", jid, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil || len(data) == 0 {
		return
	}

	_ = os.WriteFile(DiskPath(jid), data, 0o600)

	glib.IdleAdd(func() {
		if texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data)); err == nil {
			mu.Lock()
			memoryCache[jid] = texture
			mu.Unlock()
			if onLoaded != nil {
				onLoaded(texture)
			}
		}
	})
}
