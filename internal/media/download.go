package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"go.mau.fi/whatsmeow"
	"postbetasoftware.com/whatsdown/internal/store"
)

var mediaDownloadMu sync.Mutex

type Downloader interface {
	Download(ctx context.Context, msg whatsmeow.DownloadableMessage) ([]byte, error)
}

func DownloadMedia(ctx context.Context, client Downloader, msg store.StoredMessage) (string, error) {
	targetPath := GetLocalMediaPath(msg)
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return targetPath, nil
	}
	if client == nil {
		return "", errors.New("WhatsApp client is not connected")
	}

	waMsg := ParseE2EMessage(msg.Raw)
	if waMsg == nil {
		return "", errors.New("message payload is empty or invalid")
	}

	var downloadable whatsmeow.DownloadableMessage
	switch {
	case waMsg.GetImageMessage() != nil:
		downloadable = waMsg.GetImageMessage()
	case waMsg.GetVideoMessage() != nil:
		downloadable = waMsg.GetVideoMessage()
	case waMsg.GetAudioMessage() != nil:
		downloadable = waMsg.GetAudioMessage()
	case waMsg.GetDocumentMessage() != nil:
		downloadable = waMsg.GetDocumentMessage()
	case waMsg.GetStickerMessage() != nil:
		downloadable = waMsg.GetStickerMessage()
	default:
		return "", errors.New("unsupported downloadable media type")
	}

	mediaDownloadMu.Lock()
	defer mediaDownloadMu.Unlock()

	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return targetPath, nil
	}

	data, err := client.Download(ctx, downloadable)
	if err != nil {
		return "", fmt.Errorf("downloading media: %w", err)
	}

	if err := os.WriteFile(targetPath, data, 0o600); err != nil {
		return "", fmt.Errorf("saving media to disk: %w", err)
	}

	return targetPath, nil
}
