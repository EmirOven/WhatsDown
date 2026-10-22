package avatar

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func notifyWaiters(jid string, tex *gdk.Texture) {
	mu.Lock()
	cbs := inFlight[jid]
	delete(inFlight, jid)
	mu.Unlock()
	if len(cbs) == 0 {
		return
	}
	glib.IdleAdd(func() {
		for _, cb := range cbs {
			if cb != nil {
				cb(tex)
			}
		}
	})
}

func fetchRemote(client ClientProvider, jid string, isCommunity bool) {
	fetchSem <- struct{}{}
	defer func() {
		time.Sleep(150 * time.Millisecond)
		<-fetchSem
	}()

	targetJID, err := types.ParseJID(jid)
	if err != nil {
		notifyWaiters(jid, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	params := &whatsmeow.GetProfilePictureParams{
		Preview:     true,
		IsCommunity: isCommunity,
	}
	picInfo, err := client.GetProfilePictureInfo(ctx, targetJID, params)
	if err != nil || picInfo == nil || picInfo.URL == "" {
		mu.Lock()
		negativeCache[jid] = time.Now()
		mu.Unlock()
		notifyWaiters(jid, nil)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, picInfo.URL, nil)
	if err != nil {
		notifyWaiters(jid, nil)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		notifyWaiters(jid, nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		notifyWaiters(jid, nil)
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil || len(data) == 0 {
		notifyWaiters(jid, nil)
		return
	}

	_ = os.WriteFile(DiskPath(jid), data, 0o600)

	glib.IdleAdd(func() {
		var tex *gdk.Texture
		if t, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data)); err == nil {
			tex = t
			mu.Lock()
			memoryCache[jid] = tex
			mu.Unlock()
		}
		notifyWaiters(jid, tex)
	})
}
