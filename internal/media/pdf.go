package media

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

// GetPDFPageCount returns the total number of pages in the given PDF document.
func GetPDFPageCount(path string) int {
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

// GetPDFPageTexture extracts or retrieves a cached preview texture for a specific PDF page.
func GetPDFPageTexture(path string, pageNum int) *gdk.Texture {
	if path == "" {
		return nil
	}
	if pageNum < 1 {
		pageNum = 1
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", path, pageNum)))
	thumbName := fmt.Sprintf("pdf_page_%x.png", h[:8])
	thumbPath := filepath.Join(CacheDir(), thumbName)
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
	return nil
}

// GetPDFThumbnailTexture extracts or retrieves a cached thumbnail for a PDF document.
func GetPDFThumbnailTexture(path string) *gdk.Texture {
	return GetPDFPageTexture(path, 1)
}
