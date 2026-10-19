package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

var (
	avatarMu            sync.RWMutex
	avatarMemoryCache   = make(map[string]*gdk.Texture)
	avatarNegativeCache = make(map[string]time.Time)
	avatarInFlight      = make(map[string]bool)
)

// avatarCacheDir returns the directory path for cached avatar image files.
func avatarCacheDir() string {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown", "avatars")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func sanitizeJIDFilename(jid string) string {
	clean := strings.ReplaceAll(jid, "/", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	return clean
}

// loadAvatar sets the fallback initials and asynchronously fetches/applies the
// profile picture for the given JID to the provided Adw.Avatar.
func loadAvatar(jid string, displayName string, avatar *adw.Avatar) {
	if avatar == nil {
		return
	}

	avatar.SetText(displayName)
	avatar.SetShowInitials(true)

	if jid == "" {
		avatar.SetCustomImage(nil)
		return
	}

	// 1. Check in-memory texture cache
	avatarMu.RLock()
	if tex, ok := avatarMemoryCache[jid]; ok {
		avatarMu.RUnlock()
		avatar.SetCustomImage(tex)
		return
	}
	// Check negative cache (no picture / privacy restrictions)
	if lastFailed, ok := avatarNegativeCache[jid]; ok {
		if time.Since(lastFailed) < 1*time.Hour {
			avatarMu.RUnlock()
			avatar.SetCustomImage(nil)
			return
		}
	}
	avatarMu.RUnlock()

	// 2. Check local disk cache
	diskPath := filepath.Join(avatarCacheDir(), sanitizeJIDFilename(jid)+".jpg")
	if data, err := os.ReadFile(diskPath); err == nil && len(data) > 0 {
		texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data))
		if err == nil {
			avatarMu.Lock()
			avatarMemoryCache[jid] = texture
			avatarMu.Unlock()

			avatar.SetCustomImage(texture)
			return
		}
	}

	// 3. Fetch from WhatsApp servers if connected
	if waClient == nil || !waClient.IsConnected() {
		avatar.SetCustomImage(nil)
		return
	}

	avatarMu.Lock()
	if avatarInFlight[jid] {
		avatarMu.Unlock()
		return
	}
	avatarInFlight[jid] = true
	avatarMu.Unlock()

	go func() {
		defer func() {
			avatarMu.Lock()
			delete(avatarInFlight, jid)
			avatarMu.Unlock()
		}()

		targetJID, err := types.ParseJID(jid)
		if err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		picInfo, err := waClient.GetProfilePictureInfo(ctx, targetJID, &whatsmeow.GetProfilePictureParams{
			Preview: true,
		})
		if err != nil {
			if errors.Is(err, whatsmeow.ErrProfilePictureNotSet) || errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
				avatarMu.Lock()
				avatarNegativeCache[jid] = time.Now()
				avatarMu.Unlock()
			} else {
				log.Printf("fetching profile picture info for %s: %v", jid, err)
			}
			return
		}

		if picInfo == nil || picInfo.URL == "" {
			avatarMu.Lock()
			avatarNegativeCache[jid] = time.Now()
			avatarMu.Unlock()
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

		// Save to disk
		_ = os.WriteFile(diskPath, data, 0o600)

		// Create texture and update UI
		glib.IdleAdd(func() {
			texture, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(data))
			if err == nil {
				avatarMu.Lock()
				avatarMemoryCache[jid] = texture
				avatarMu.Unlock()

				avatar.SetCustomImage(texture)
			}
		})
	}()
}

// invalidateAvatar removes the cached avatar for the given JID (e.g. after a picture update).
func invalidateAvatar(jid string) {
	avatarMu.Lock()
	delete(avatarMemoryCache, jid)
	delete(avatarNegativeCache, jid)
	avatarMu.Unlock()

	diskPath := filepath.Join(avatarCacheDir(), sanitizeJIDFilename(jid)+".jpg")
	_ = os.Remove(diskPath)
}

// clearAvatarCache removes disk cached avatars and empties the in-memory texture cache.
func clearAvatarCache() {
	cacheDir := avatarCacheDir()
	_ = os.RemoveAll(cacheDir)
	_ = os.MkdirAll(cacheDir, 0o700)

	avatarMu.Lock()
	avatarMemoryCache = make(map[string]*gdk.Texture)
	avatarNegativeCache = make(map[string]time.Time)
	avatarMu.Unlock()
}

