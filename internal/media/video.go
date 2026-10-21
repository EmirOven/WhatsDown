package media

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

// GetVideoThumbnailTexture extracts or retrieves a cached frame thumbnail for a video.
func GetVideoThumbnailTexture(path string) *gdk.Texture {
	if path == "" {
		return nil
	}
	h := sha256.Sum256([]byte(path))
	thumbName := fmt.Sprintf("vid_thumb_%x.jpg", h[:8])
	thumbPath := filepath.Join(CacheDir(), thumbName)
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
