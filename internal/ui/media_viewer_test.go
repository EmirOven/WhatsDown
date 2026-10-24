package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestMediaViewerPhotoAndActions(t *testing.T) {
	c := dialogTestController(t)
	path := filepath.Join(t.TempDir(), "photo.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 100, 80))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	msg := store.StoredMessage{ChatJID: "123@g.us", ID: "viewer-photo", Kind: store.MessageKindImage}
	base := gtk.NewBox(gtk.OrientationVertical, 0)
	c.rootOverlay.SetChild(base)
	toplevels := len(gtk.WindowListToplevels())
	win := c.buildMediaViewer(msg, func(context.Context) (string, error) { return path, nil })
	t.Cleanup(win.Close)
	if c.mediaViewer != win || win.widget.Parent() == nil {
		t.Fatal("media viewer blocks the main window")
	}
	if coreglib.BaseObject(c.rootOverlay.Child()).Native() != coreglib.BaseObject(base).Native() {
		t.Fatal("opening media replaced the conversation")
	}
	var picture *gtk.Picture
	deadline := time.Now().Add(2 * time.Second)
	for picture == nil && time.Now().Before(deadline) {
		pumpSidebarEvents(10 * time.Millisecond)
		walkDialog(win.widget, func(w gtk.Widgetter) {

			if pic, ok := w.(*gtk.Picture); ok && pic.Paintable() != nil {
				picture = pic
			}
		})
	}
	if picture == nil {
		t.Fatal("image did not load inside the viewer")
	}
	if len(gtk.WindowListToplevels()) != toplevels {
		t.Fatal("media created another window")
	}
	for _, name := range []string{"reply", "forward", "go-to-message", "open-external", "show-in-files"} {
		if win.actions[name] == nil || !win.actions[name].Enabled() {
			t.Fatalf("missing enabled action %s", name)
		}
	}

	if win.Canvas == nil {
		t.Fatal("viewer lacks zoom/rotation canvas")
	}
	win.Canvas.Rotate()
	paintable := win.Canvas.Picture.Paintable()
	if paintable.IntrinsicWidth() != 80 || paintable.IntrinsicHeight() != 100 {
		t.Fatal("rotation did not swap image dimensions")
	}
	win.Canvas.SetZoom(2)
	pumpSidebarEvents(40 * time.Millisecond)
	if win.Canvas.Zoom != 2 || win.Canvas.Picture.CanShrink() {
		t.Fatal("zoom did not enlarge the image canvas")
	}
	var wheel *gtk.EventControllerScroll
	controllers := win.Canvas.Scroll.ObserveControllers()
	for i := uint(0); i < controllers.NItems(); i++ {
		if candidate, ok := controllers.Item(i).Cast().(*gtk.EventControllerScroll); ok {
			wheel = candidate
		}
	}
	if wheel == nil {
		t.Fatal("viewer has no scrollwheel zoom")
	}
	win.Canvas.ZoomByScroll(-1)
	if win.Canvas.Zoom <= 2 {
		t.Fatal("scrollwheel did not zoom in")
	}
	pumpSidebarEvents(30 * time.Millisecond)
	var drag *gtk.GestureDrag
	controllers = win.Canvas.Picture.ObserveControllers()
	for i := uint(0); i < controllers.NItems(); i++ {
		if candidate, ok := controllers.Item(i).Cast().(*gtk.GestureDrag); ok {
			drag = candidate
		}
	}
	if drag == nil {
		t.Fatal("viewer has no drag panning")
	}
	adjustment := win.Canvas.Scroll.VAdjustment()
	if adjustment.Upper() > adjustment.PageSize() {
		adjustment.SetValue(0)
		drag.Emit("drag-begin", float64(0), float64(0))
		drag.Emit("drag-update", float64(0), float64(-20))
		if adjustment.Value() <= 0 {
			t.Fatal("dragging did not pan zoomed media")
		}
	}
	win.Canvas.SetZoom(1)
	if !win.Canvas.Picture.CanShrink() {
		t.Fatal("Fit did not restore responsive sizing")
	}

	// Clicking Reply targets the original message, and leaves the main app usable.
	c.currentChat = &store.Chat{JID: msg.ChatJID}
	c.messagesList = gtk.NewListBox()
	c.messagesScrolledWindow = gtk.NewScrolledWindow()
	c.messagesScrolledWindow.SetChild(c.messagesList)
	row := gtk.NewListBoxRow()
	row.SetChild(gtk.NewLabel("photo message"))
	c.messagesList.Append(row)
	c.messageRowMap = map[string]*gtk.ListBoxRow{msg.ID: row}
	win.actions["reply"].Activate(nil)
	if c.mediaViewer != nil || coreglib.BaseObject(c.rootOverlay.Child()).Native() != coreglib.BaseObject(base).Native() {
		t.Fatal("closing viewer did not preserve root content")
	}
	if c.replyingTo == nil || c.replyingTo.ID != msg.ID {
		t.Fatal("viewer Reply did not target original message")
	}
	pumpSidebarEvents(300 * time.Millisecond)
}

func TestMediaViewerCloseCancelsDownload(t *testing.T) {
	c := dialogTestController(t)
	started, canceled := make(chan struct{}), make(chan struct{})
	win := c.buildMediaViewer(store.StoredMessage{Kind: store.MessageKindVideo}, func(ctx context.Context) (string, error) {
		close(started)
		<-ctx.Done()
		close(canceled)
		return "", ctx.Err()
	})
	t.Cleanup(win.Close)
	<-started
	win.Close()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("closing viewer did not cancel download")
	}
	pumpSidebarEvents(20 * time.Millisecond)
}

func TestVoiceSpeedCycleAndSeekControl(t *testing.T) {
	c := newFolderSidebar(t)
	box := c.buildAudioWidget(store.StoredMessage{Kind: store.MessageKindVoice})
	t.Cleanup(func() { c.releaseMessageWidget(box) })
	var speed *gtk.Button
	var scale *gtk.Scale
	walkDialog(box, func(w gtk.Widgetter) {
		if b, ok := w.(*gtk.Button); ok && b.Label() == "1x" {
			speed = b
		}
		if s, ok := w.(*gtk.Scale); ok {
			scale = s
		}
	})
	if speed == nil || scale == nil {
		t.Fatal("voice message lacks speed control or scrubber")
	}
	for _, want := range []string{"1.5x", "2x", "0.5x", "1x"} {
		speed.Emit("clicked")
		if speed.Label() != want {
			t.Fatalf("speed=%q,want %q", speed.Label(), want)
		}
	}
}

func TestPhotoPreviewHasNoViewButton(t *testing.T) {
	c := newFolderSidebar(t)
	preview := c.buildImageVideoWidget(store.StoredMessage{Kind: store.MessageKindImage})
	t.Cleanup(func() { c.releaseMessageWidget(preview) })
	walkDialog(preview, func(w gtk.Widgetter) {
		if _, ok := w.(*gtk.Button); ok {
			t.Fatal("photo preview still contains an overlay button")
		}
	})
	if !preview.Focusable() {
		t.Fatal("clickable photo cannot be reached with keyboard")
	}
}

func TestPDFAndVideoUseInternalViewers(t *testing.T) {
	for _, kind := range []string{store.MessageKindDocument, store.MessageKindVideo} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sample.pdf")
			if kind == store.MessageKindDocument {
				if _, err := exec.LookPath("pdftoppm"); err != nil {
					t.Skip("poppler required")
				}
				var pdf bytes.Buffer
				pdf.WriteString("%PDF-1.4\n")
				offsets := []int{0}
				for i, object := range []string{
					"<< /Type /Catalog /Pages 2 0 R >>",
					"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
					"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 200] /Resources << >> /Contents 4 0 R >>",
					"<< /Length 0 >>\nstream\n\nendstream",
				} {
					offsets = append(offsets, pdf.Len())
					fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, object)
				}
				start := pdf.Len()
				pdf.WriteString("xref\n0 5\n0000000000 65535 f \n")
				for _, offset := range offsets[1:] {
					fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
				}
				fmt.Fprintf(&pdf, "trailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", start)
				if err := os.WriteFile(path, pdf.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := exec.LookPath("ffmpeg"); err != nil {
					t.Skip("ffmpeg required to generate video fixture")
				}
				path = filepath.Join(t.TempDir(), "sample.mp4")
				if out, err := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "color=black:s=64x64:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path).CombinedOutput(); err != nil {
					t.Fatalf("video fixture: %v: %s", err, out)
				}
			}
			c := dialogTestController(t)
			win := c.buildMediaViewer(store.StoredMessage{Kind: kind}, func(context.Context) (string, error) { return path, nil })
			t.Cleanup(win.Close)
			if c.mediaViewer != win || win.widget.Parent() == nil {
				t.Fatal("viewer is modal")
			}
			deadline := time.Now().Add(4 * time.Second)
			ready := false
			for !ready && time.Now().Before(deadline) {
				pumpSidebarEvents(20 * time.Millisecond)
				walkDialog(win.widget, func(w gtk.Widgetter) {
					if kind == store.MessageKindDocument {
						if p, ok := w.(*gtk.Picture); ok && p.Paintable() != nil {
							ready = true
						}
					}
					if kind == store.MessageKindVideo {
						if v, ok := w.(*gtk.Video); ok {
							if stream := v.MediaStream(); stream != nil {
								if err := gtk.BaseMediaStream(stream).Error(); err != nil {
									t.Fatal(err)
								}
								ready = gtk.BaseMediaStream(stream).IsPrepared()
							}
						}
					}
				})
			}
			if !ready {
				t.Fatalf("%s did not render inside the app", kind)
			}
			win.Close()
			pumpSidebarEvents(20 * time.Millisecond)
		})
	}
}

func TestEscapeClosesMediaViewer(t *testing.T) {
	c := dialogTestController(t)
	view := c.buildMediaViewer(store.StoredMessage{Kind: store.MessageKindImage}, func(ctx context.Context) (string, error) { <-ctx.Done(); return "", ctx.Err() })
	t.Cleanup(view.Close)
	if !c.handleChatViewKeyPress(gdk.KEY_Escape, 0, 0) || c.mediaViewer != nil {
		t.Fatal("Escape did not close media overlay")
	}
	pumpSidebarEvents(10 * time.Millisecond)
}
