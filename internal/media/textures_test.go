package media

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewTextureBounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1600, 800))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	tex, err := loadPreviewTexture(path)
	if err != nil {
		t.Fatal(err)
	}
	if tex.Width() != 640 || tex.Height() != 320 {
		t.Fatalf("preview size = %dx%d, want 640x320", tex.Width(), tex.Height())
	}
	// The on-disk attachment must remain at its original resolution.
	f, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	config, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 1600 || config.Height != 800 {
		t.Fatal("original image changed")
	}
}

func TestPreviewDoesNotEnlargeThumbnails(t *testing.T) {
	for _, size := range []image.Point{{X: 64, Y: 32}, {X: 32, Y: 64}, {X: 140, Y: 140}} {
		func() {
			path := filepath.Join(t.TempDir(), "thumb.png")
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, size.X, size.Y))); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			tex, err := loadPreviewTexture(path)
			if err != nil {
				t.Fatal(err)
			}
			if tex.Width() != size.X || tex.Height() != size.Y {
				t.Fatalf("%dx%d thumbnail enlarged to %dx%d", size.X, size.Y, tex.Width(), tex.Height())
			}
		}()
	}
}
