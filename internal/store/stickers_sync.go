package store

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
)

func savedStickersDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "whatsdown", "stickers")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func (s *MessageStore) SyncHistoryStickers(ctx context.Context, client *whatsmeow.Client, favorites, recents []*waHistorySync.StickerMetadata) {
	if client == nil {
		return
	}
	s.stickerSyncMu.Lock()
	defer s.stickerSyncMu.Unlock()

	targetDir := savedStickersDir()
	imported := 0

	processList := func(list []*waHistorySync.StickerMetadata) {
		for _, st := range list {
			if st == nil {
				continue
			}
			sha := st.GetFileSHA256()
			if len(sha) == 0 {
				continue
			}
			fileHash := hex.EncodeToString(sha)
			if saved, _ := s.IsStickerSaved(ctx, fileHash); saved {
				continue
			}

			destPath := filepath.Join(targetDir, fileHash+".webp")
			var data []byte
			if fi, err := os.Stat(destPath); err == nil && fi.Size() > 0 {
				data, _ = os.ReadFile(destPath)
			}
			if len(data) == 0 {
				dlCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				d, err := client.Download(dlCtx, st)
				cancel()
				if err != nil || len(d) == 0 {
					continue
				}
				data = d
				if err := os.WriteFile(destPath, data, 0o600); err != nil {
					continue
				}
			}

			w, h := int(st.GetWidth()), int(st.GetHeight())
			if w <= 0 {
				w = 512
			}
			if h <= 0 {
				h = 512
			}

			if err := s.SaveSticker(ctx, SavedSticker{
				ID:       fileHash,
				FileHash: fileHash,
				FilePath: destPath,
				Width:    w,
				Height:   h,
				SavedAt:  time.Now(),
			}); err == nil {
				imported++
				if imported%10 == 0 && s.OnStickersUpdated != nil {
					s.OnStickersUpdated()
				}
			}
		}
	}

	processList(favorites)
	processList(recents)

	if imported > 0 && s.OnStickersUpdated != nil {
		s.OnStickersUpdated()
	}
}
