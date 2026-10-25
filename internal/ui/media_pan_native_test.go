package ui

import (
	"context"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestNativeMediaPanUsesStableViewportCoordinates(t *testing.T) {
	if os.Getenv("WHATSDOWN_NATIVE_INPUT_TESTS") != "1" {
		t.Skip("set WHATSDOWN_NATIVE_INPUT_TESTS=1 on an isolated X11 display")
	}
	if os.Getenv("DISPLAY") == "" || (os.Getenv("WAYLAND_DISPLAY") != "" && os.Getenv("GDK_BACKEND") != "x11") {
		t.Skip("requires X11")
	}
	c := dialogTestController(t)
	const title = "WhatsDown media pan native test"
	c.window.SetTitle(title)
	c.window.SetDefaultSize(640, 480)
	path := filepath.Join(t.TempDir(), "photo.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 400, 300))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	view := c.buildMediaViewer(store.StoredMessage{Kind: store.MessageKindImage}, func(context.Context) (string, error) { return path, nil })
	t.Cleanup(view.Close)
	deadline := time.Now().Add(time.Second)
	for view.Canvas == nil && time.Now().Before(deadline) {
		pumpSidebarEvents(10 * time.Millisecond)
	}
	if view.Canvas == nil {
		t.Fatal("image did not load")
	}
	canvas := view.Canvas
	pumpSidebarEvents(200 * time.Millisecond)
	canvas.SetZoom(3)
	pumpSidebarEvents(150 * time.Millisecond)
	h, v := canvas.Scroll.HAdjustment(), canvas.Scroll.VAdjustment()
	h.SetValue(200)
	v.SetValue(200)
	startH, startV := h.Value(), v.Value()
	if startH < 100 || startV < 100 {
		t.Fatalf("zoomed canvas is not scrollable: viewport=%dx%d bounds=(%v/%v,%v/%v)", canvas.Scroll.Width(), canvas.Scroll.Height(), h.Upper(), h.PageSize(), v.Upper(), v.PageSize())
	}
	x, y, ok := canvas.Scroll.TranslateCoordinates(c.window, 220, 150)
	if !ok {
		t.Fatal("viewport has no window coordinates")
	}
	// Multiple physical moves while holding the button reproduce moving-image feedback.
	nativeWindowActions(t, title, []any{
		[]any{"move", x, y}, []any{"down"},
		[]any{"move", x + 20, y + 15}, []any{"move", x + 40, y + 30}, []any{"move", x + 60, y + 45}, []any{"move", x + 80, y + 60}, []any{"up"},
	})
	if math.Abs(h.Value()-(startH-80)) > 3 || math.Abs(v.Value()-(startV-60)) > 3 {
		t.Fatalf("drag drifted: got (%.1f,%.1f), want (%.1f,%.1f)", h.Value(), v.Value(), startH-80, startV-60)
	}
	nativeWindowActions(t, title, []any{
		[]any{"move", x + 80, y + 60}, []any{"down"},
		[]any{"move", x + 60, y + 45}, []any{"move", x + 40, y + 30}, []any{"move", x + 20, y + 15}, []any{"move", x, y}, []any{"up"},
	})
	if math.Abs(h.Value()-startH) > 3 || math.Abs(v.Value()-startV) > 3 {
		t.Fatalf("reverse drag did not restore position: %.1f,%.1f", h.Value(), v.Value())
	}
}
