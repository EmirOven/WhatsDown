package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// CacheDir returns the directory path where downloaded media files are cached.
func CacheDir() string {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown", "media")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// SavedStickersDir returns the directory path where personal saved stickers are stored.
func SavedStickersDir() string {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown", "stickers")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// EnsureWebPSticker converts an image file into WebP suitable for stickers.
func EnsureWebPSticker(inputPath string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(inputPath))
	if ext == ".webp" {
		return os.ReadFile(inputPath)
	}

	out, err := exec.Command("magick", inputPath, "-resize", "512x512>", "webp:-").Output()
	if err == nil && len(out) > 0 {
		return out, nil
	}

	tmpOut := filepath.Join(os.TempDir(), fmt.Sprintf("stk_conv_%d.webp", time.Now().UnixNano()))
	defer os.Remove(tmpOut)
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-vf", "scale='min(512,iw)':min'(512,ih)':force_original_aspect_ratio=decrease", tmpOut)
	if err := cmd.Run(); err == nil {
		if b, err := os.ReadFile(tmpOut); err == nil && len(b) > 0 {
			return b, nil
		}
	}
	return os.ReadFile(inputPath)
}
