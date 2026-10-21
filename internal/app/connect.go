package app

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
)

func connectWhatsApp(client *whatsmeow.Client) {
	if client.Store.ID != nil {
		if err := client.Connect(); err != nil {
			log.Printf("connecting to WhatsApp: %v", err)
		} else {
			go func() {
				time.Sleep(500 * time.Millisecond)
				for _, name := range appstate.AllPatchNames {
					fullSync := name == appstate.WAPatchRegularLow
					_ = client.FetchAppState(context.Background(), name, fullSync, false)
				}
			}()
		}
		return
	}

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		log.Printf("starting QR login: %v", err)
		return
	}
	if err := client.Connect(); err != nil {
		log.Printf("connecting to WhatsApp: %v", err)
		return
	}

	for evt := range qrChan {
		if evt.Event == whatsmeow.QRChannelEventCode {
			pngBytes, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				continue
			}
			glib.IdleAdd(func() {
				texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(pngBytes))
				if err == nil {
					setQRTexture(texture)
				}
			})
		}
	}
}
