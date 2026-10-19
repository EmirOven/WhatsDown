package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

var (
	mediaTextureMu    sync.RWMutex
	mediaTextureCache = make(map[string]*gdk.Texture)
	mediaDownloadMu   sync.Mutex
)

// mediaCacheDir returns the directory path where downloaded media files are cached.
func mediaCacheDir() string {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown", "media")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// parseE2EMessage parses a raw protobuf byte slice into a waE2E.Message.
func parseE2EMessage(raw []byte) *waE2E.Message {
	if len(raw) == 0 {
		return nil
	}
	var msg waE2E.Message
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return nil
	}
	return &msg
}

// extractMentionedJIDs returns any JIDs explicitly mentioned in a message's context.
func extractMentionedJIDs(raw []byte) []string {
	waMsg := parseE2EMessage(raw)
	if waMsg == nil {
		return nil
	}
	var ctxInfo *waE2E.ContextInfo
	switch {
	case waMsg.GetExtendedTextMessage() != nil:
		ctxInfo = waMsg.GetExtendedTextMessage().GetContextInfo()
	case waMsg.GetImageMessage() != nil:
		ctxInfo = waMsg.GetImageMessage().GetContextInfo()
	case waMsg.GetVideoMessage() != nil:
		ctxInfo = waMsg.GetVideoMessage().GetContextInfo()
	case waMsg.GetAudioMessage() != nil:
		ctxInfo = waMsg.GetAudioMessage().GetContextInfo()
	case waMsg.GetDocumentMessage() != nil:
		ctxInfo = waMsg.GetDocumentMessage().GetContextInfo()
	case waMsg.GetStickerMessage() != nil:
		ctxInfo = waMsg.GetStickerMessage().GetContextInfo()
	}
	if ctxInfo != nil {
		return ctxInfo.GetMentionedJID()
	}
	return nil
}

// getThumbCachePath computes the local file path for caching a media thumbnail.
func getThumbCachePath(msg StoredMessage) string {
	filename := fmt.Sprintf("thumb_%s_%s.jpg", sanitizeJIDFilename(msg.ChatJID), sanitizeJIDFilename(msg.ID))
	return filepath.Join(mediaCacheDir(), filename)
}

// getMediaThumbnailTexture returns a GdkTexture for a message's thumbnail if available.
func getMediaThumbnailTexture(msg StoredMessage) *gdk.Texture {
	key := msg.ChatJID + "_" + msg.ID
	mediaTextureMu.RLock()
	if tex, ok := mediaTextureCache[key]; ok && tex != nil {
		mediaTextureMu.RUnlock()
		return tex
	}
	mediaTextureMu.RUnlock()

	// 1. Check if thumbnail was already cached on disk
	thumbPath := getThumbCachePath(msg)
	if fi, err := os.Stat(thumbPath); err == nil && fi.Size() > 0 {
		tex, err := gdk.NewTextureFromFilename(thumbPath)
		if err == nil && tex != nil {
			mediaTextureMu.Lock()
			mediaTextureCache[key] = tex
			mediaTextureMu.Unlock()
			return tex
		}
	}

	// 2. Check if embedded thumbnail bytes exist in protobuf
	waMsg := parseE2EMessage(msg.Raw)
	if waMsg != nil {
		var thumbBytes []byte
		switch {
		case waMsg.GetImageMessage() != nil:
			thumbBytes = waMsg.GetImageMessage().GetJPEGThumbnail()
		case waMsg.GetVideoMessage() != nil:
			thumbBytes = waMsg.GetVideoMessage().GetJPEGThumbnail()
		case waMsg.GetDocumentMessage() != nil:
			thumbBytes = waMsg.GetDocumentMessage().GetJPEGThumbnail()
		}

		if len(thumbBytes) > 0 {
			_ = os.WriteFile(thumbPath, thumbBytes, 0o600)
			tex, err := gdk.NewTextureFromFilename(thumbPath)
			if err == nil && tex != nil {
				mediaTextureMu.Lock()
				mediaTextureCache[key] = tex
				mediaTextureMu.Unlock()
				return tex
			}
		}
	}

	// 3. Check if full media file is already cached on disk
	localPath := getLocalMediaPath(msg)
	if localPath != "" {
		if fi, err := os.Stat(localPath); err == nil && fi.Size() > 0 {
			lower := strings.ToLower(localPath)
			if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
				tex, err := gdk.NewTextureFromFilename(localPath)
				if err == nil && tex != nil {
					mediaTextureMu.Lock()
					mediaTextureCache[key] = tex
					mediaTextureMu.Unlock()
					return tex
				}
			} else if strings.HasSuffix(lower, ".pdf") {
				if tex := getPDFThumbnailTexture(localPath); tex != nil {
					mediaTextureMu.Lock()
					mediaTextureCache[key] = tex
					mediaTextureMu.Unlock()
					return tex
				}
			} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".mkv") || strings.HasSuffix(lower, ".webm") {
				if tex := getVideoThumbnailTexture(localPath); tex != nil {
					mediaTextureMu.Lock()
					mediaTextureCache[key] = tex
					mediaTextureMu.Unlock()
					return tex
				}
			}
		}
	}

	return nil
}

// getPDFPageCount returns the total number of pages in the given PDF document.
func getPDFPageCount(path string) int {
	if path == "" {
		return 1
	}
	out, err := exec.Command("pdfinfo", path).Output()
	if err != nil {
		return 1
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
					return n
				}
			}
		}
	}
	return 1
}

// getPDFPageTexture extracts or retrieves a cached preview texture for a specific page of a PDF document.
func getPDFPageTexture(path string, pageNum int) *gdk.Texture {
	if path == "" {
		return nil
	}
	if pageNum < 1 {
		pageNum = 1
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", path, pageNum)))
	thumbName := fmt.Sprintf("pdf_page_%x.png", h[:8])
	thumbPath := filepath.Join(mediaCacheDir(), thumbName)
	if fi, err := os.Stat(thumbPath); err == nil && fi.Size() > 0 {
		if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil && tex != nil {
			return tex
		}
	}

	prefix := filepath.Join(os.TempDir(), fmt.Sprintf("pdf_page_%x", h[:8]))
	defer func() {
		if matches, err := filepath.Glob(prefix + "*"); err == nil {
			for _, m := range matches {
				_ = os.Remove(m)
			}
		}
	}()

	pageStr := strconv.Itoa(pageNum)
	// -singlefile ensures output is directly <prefix>.png
	cmd := exec.Command("pdftoppm", "-png", "-f", pageStr, "-l", pageStr, "-singlefile", "-scale-to", "600", path, prefix)
	if err := cmd.Run(); err == nil {
		targetPNG := prefix + ".png"
		if fi, err := os.Stat(targetPNG); err == nil && fi.Size() > 0 {
			if data, err := os.ReadFile(targetPNG); err == nil {
				_ = os.WriteFile(thumbPath, data, 0o600)
				if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil {
					return tex
				}
			}
		}
	}

	// Fallback without -singlefile in case of older pdftoppm version
	cmdFallback := exec.Command("pdftoppm", "-png", "-f", pageStr, "-l", pageStr, "-scale-to", "600", path, prefix)
	if err := cmdFallback.Run(); err == nil {
		if matches, err := filepath.Glob(prefix + "*.png"); err == nil && len(matches) > 0 {
			if data, err := os.ReadFile(matches[0]); err == nil {
				_ = os.WriteFile(thumbPath, data, 0o600)
				if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil {
					return tex
				}
			}
		}
	}
	return nil
}

// getPDFThumbnailTexture extracts or retrieves a cached thumbnail for a PDF document.
func getPDFThumbnailTexture(path string) *gdk.Texture {
	return getPDFPageTexture(path, 1)
}

// getVideoThumbnailTexture extracts or retrieves a cached frame thumbnail for a video.
func getVideoThumbnailTexture(path string) *gdk.Texture {
	if path == "" {
		return nil
	}
	h := sha256.Sum256([]byte(path))
	thumbName := fmt.Sprintf("vid_thumb_%x.jpg", h[:8])
	thumbPath := filepath.Join(mediaCacheDir(), thumbName)
	if fi, err := os.Stat(thumbPath); err == nil && fi.Size() > 0 {
		if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil && tex != nil {
			return tex
		}
	}

	cmd := exec.Command("ffmpeg", "-y", "-ss", "00:00:01", "-i", path, "-vframes", "1", "-q:v", "2", thumbPath)
	if err := cmd.Run(); err == nil {
		if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil && tex != nil {
			return tex
		}
	}
	cmd = exec.Command("ffmpeg", "-y", "-ss", "00:00:00", "-i", path, "-vframes", "1", "-q:v", "2", thumbPath)
	if err := cmd.Run(); err == nil {
		if tex, err := gdk.NewTextureFromFilename(thumbPath); err == nil && tex != nil {
			return tex
		}
	}
	return nil
}

var (
	currentAudioCmd  *exec.Cmd
	currentAudioPath string
	currentAudioStop func()
	audioPlayingMu   sync.Mutex
)

// stopCurrentAudio stops any ongoing audio preview playback.
func stopCurrentAudio() {
	audioPlayingMu.Lock()
	defer audioPlayingMu.Unlock()
	if currentAudioCmd != nil && currentAudioCmd.Process != nil {
		_ = currentAudioCmd.Process.Kill()
		currentAudioCmd = nil
	}
	if currentAudioStop != nil {
		currentAudioStop()
		currentAudioStop = nil
	}
	currentAudioPath = ""
}

// playAudioPreview plays audio for preview using gst-play-1.0 or ffplay.
func playAudioPreview(path string, onPlaybackState func(playing bool)) {
	audioPlayingMu.Lock()
	if currentAudioPath == path && currentAudioCmd != nil {
		audioPlayingMu.Unlock()
		stopCurrentAudio()
		return
	}
	audioPlayingMu.Unlock()

	stopCurrentAudio()

	audioPlayingMu.Lock()
	currentAudioPath = path
	currentAudioStop = func() {
		if onPlaybackState != nil {
			glib.IdleAdd(func() {
				onPlaybackState(false)
			})
		}
	}
	audioPlayingMu.Unlock()

	if onPlaybackState != nil {
		onPlaybackState(true)
	}

	go func() {
		var cmd *exec.Cmd
		if _, err := exec.LookPath("gst-play-1.0"); err == nil {
			cmd = exec.Command("gst-play-1.0", "--no-interactive", path)
		} else if _, err := exec.LookPath("ffplay"); err == nil {
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", path)
		} else {
			cmd = exec.Command("xdg-open", path)
		}

		audioPlayingMu.Lock()
		currentAudioCmd = cmd
		audioPlayingMu.Unlock()

		_ = cmd.Run()

		audioPlayingMu.Lock()
		if currentAudioCmd == cmd {
			currentAudioCmd = nil
			if currentAudioStop != nil {
				currentAudioStop()
				currentAudioStop = nil
			}
			currentAudioPath = ""
		}
		audioPlayingMu.Unlock()
	}()
}

// loadMediaPreview sets the paintable on picture, loading synchronously if cached
// or asynchronously downloading the media in the background.
func loadMediaPreview(msg StoredMessage, picture *gtk.Picture, onLoaded func(tex *gdk.Texture)) {
	if tex := getMediaThumbnailTexture(msg); tex != nil {
		picture.SetPaintable(tex)
		if onLoaded != nil {
			onLoaded(tex)
		}
		return
	}

	waMsg := parseE2EMessage(msg.Raw)
	if waMsg == nil || (waMsg.GetImageMessage() == nil && waMsg.GetVideoMessage() == nil && waMsg.GetDocumentMessage() == nil) {
		return
	}

	key := msg.ChatJID + "_" + msg.ID
	go func() {
		path, err := downloadMedia(context.Background(), msg)
		if err != nil {
			return
		}
		var tex *gdk.Texture
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".webp") {
			tex, _ = gdk.NewTextureFromFilename(path)
		} else if strings.HasSuffix(lower, ".pdf") {
			tex = getPDFThumbnailTexture(path)
		} else if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".mkv") || strings.HasSuffix(lower, ".webm") {
			tex = getVideoThumbnailTexture(path)
		}

		if tex != nil {
			mediaTextureMu.Lock()
			mediaTextureCache[key] = tex
			mediaTextureMu.Unlock()
			glib.IdleAdd(func() {
				picture.SetPaintable(tex)
				if onLoaded != nil {
					onLoaded(tex)
				}
			})
		}
	}()
}

// getLocalMediaPath computes the anticipated local file path for a message.
func getLocalMediaPath(msg StoredMessage) string {
	waMsg := parseE2EMessage(msg.Raw)
	ext := ".bin"
	if waMsg != nil {
		switch {
		case waMsg.GetImageMessage() != nil:
			ext = ".jpg"
			if mt := waMsg.GetImageMessage().GetMimetype(); strings.Contains(mt, "png") {
				ext = ".png"
			}
		case waMsg.GetVideoMessage() != nil:
			ext = ".mp4"
		case waMsg.GetAudioMessage() != nil:
			ext = ".ogg"
			if mt := waMsg.GetAudioMessage().GetMimetype(); strings.Contains(mt, "mp4") || strings.Contains(mt, "m4a") {
				ext = ".m4a"
			}
		case waMsg.GetDocumentMessage() != nil:
			doc := waMsg.GetDocumentMessage()
			if doc.GetFileName() != "" {
				ext = filepath.Ext(doc.GetFileName())
				if ext == "" {
					ext = ".pdf"
				}
			}
		case waMsg.GetStickerMessage() != nil:
			ext = ".webp"
		}
	}

	filename := fmt.Sprintf("%s_%s%s", sanitizeJIDFilename(msg.ChatJID), sanitizeJIDFilename(msg.ID), ext)
	return filepath.Join(mediaCacheDir(), filename)
}

// downloadMedia ensures the media file is downloaded and returns the local file path.
func downloadMedia(ctx context.Context, msg StoredMessage) (string, error) {
	targetPath := getLocalMediaPath(msg)
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return targetPath, nil
	}

	if waClient == nil {
		return "", errors.New("WhatsApp client is not connected")
	}

	waMsg := parseE2EMessage(msg.Raw)
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

	// Recheck if downloaded while waiting for lock
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return targetPath, nil
	}

	data, err := waClient.Download(ctx, downloadable)
	if err != nil {
		return "", fmt.Errorf("downloading media: %w", err)
	}

	if err := os.WriteFile(targetPath, data, 0o600); err != nil {
		return "", fmt.Errorf("saving media to disk: %w", err)
	}

	return targetPath, nil
}

// openMedia opens the message's attachment with the system viewer or web browser.
func openMedia(msg StoredMessage, onStatus func(string)) {
	waMsg := parseE2EMessage(msg.Raw)
	if waMsg != nil && waMsg.GetLocationMessage() != nil {
		loc := waMsg.GetLocationMessage()
		lat := loc.GetDegreesLatitude()
		long := loc.GetDegreesLongitude()
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%f,%f", lat, long)
		_ = openURL(mapURL)
		return
	}

	if onStatus != nil {
		onStatus("Downloading…")
	}

	go func() {
		path, err := downloadMedia(context.Background(), msg)
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
			_ = openFile(path)
		})
	}()
}

// openFile launches the given file path with the default system application.
func openFile(path string) error {
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	if err := gio.AppInfoLaunchDefaultForURI(uri, nil); err == nil {
		return nil
	}
	cmd := exec.Command("xdg-open", path)
	return cmd.Start()
}

// openURL launches the given URL in the default browser.
func openURL(rawURL string) error {
	if err := gio.AppInfoLaunchDefaultForURI(rawURL, nil); err == nil {
		return nil
	}
	return exec.Command("xdg-open", rawURL).Start()
}

// formatFileSize formats bytes to human-readable size.
func formatFileSize(bytes uint64) string {
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

// copyMediaToClipboard copies the media image or file path to the system clipboard.
func copyMediaToClipboard(msg StoredMessage) {
	go func() {
		path, err := downloadMedia(context.Background(), msg)
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

// showMediaInFileManager reveals the media file in the user's desktop file manager.
func showMediaInFileManager(msg StoredMessage) {
	go func() {
		path, err := downloadMedia(context.Background(), msg)
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

