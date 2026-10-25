package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestMediaPreviewKeepsDetailsTabAndScrollPosition(t *testing.T) {
	c := dialogTestController(t)
	c.window.SetDefaultSize(1000, 760)
	c.window.Present()
	details := adw.NewDialog()
	details.SetTitle("Details")
	details.SetContentWidth(460)
	details.SetContentHeight(500)
	t.Cleanup(func() { details.ForceClose() })
	toolbar := adw.NewToolbarView()
	toolbar.AddTopBar(adw.NewHeaderBar())
	stack := gtk.NewStack()
	stack.AddNamed(gtk.NewLabel("Media"), "media")
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	content := gtk.NewBox(gtk.OrientationVertical, 0)
	for i := 0; i < 100; i++ {
		label := gtk.NewLabel(fmt.Sprintf("Details row %d", i))
		label.SetSizeRequest(-1, 30)
		content.Append(label)
	}
	scroll.SetChild(content)
	stack.AddNamed(scroll, "docs")
	stack.SetVisibleChildName("docs")
	toolbar.SetContent(stack)
	details.SetChild(toolbar)
	closed := 0
	details.ConnectClosed(func() { closed++ })
	details.Present(c.window)
	pumpSidebarEvents(200 * time.Millisecond)
	scroll.VAdjustment().SetValue(350)
	original := scroll.VAdjustment().Value()
	if original < 300 {
		t.Fatal("details panel is not scrolled")
	}
	originalChild := coreglib.BaseObject(details.Child()).Native()
	for _, kind := range []string{store.MessageKindImage, store.MessageKindDocument} {
		path := filepath.Join(t.TempDir(), "media.png")
		if kind == store.MessageKindImage {
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 100, 80))); err != nil {
				t.Fatal(err)
			}
			f.Close()
		} else {
			path = filepath.Join(t.TempDir(), "document.pdf")
			var pdf bytes.Buffer
			pdf.WriteString("%PDF-1.4\n")
			offsets := []int{0}
			objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 200] /Resources << >> /Contents 4 0 R >>", "<< /Length 0 >>\nstream\n\nendstream"}
			for i, object := range objects {
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
		}
		view := c.buildMediaViewerWithParent(store.StoredMessage{Kind: kind}, func(context.Context) (string, error) { return path, nil }, details)
		t.Cleanup(view.Close)
		pumpSidebarEvents(150 * time.Millisecond)
		if view.host == nil {
			t.Fatal("details preview did not get an in-window dialog layer")
		}
		if closed != 0 {
			t.Fatal("opening preview closed Details")
		}
		if c.window.VisibleDialog() == nil || coreglib.BaseObject(c.window.VisibleDialog()).Native() != coreglib.BaseObject(view.host).Native() {
			t.Fatal("preview was not layered above Details")
		}
		// Back closes only the preview layer, leaving the same panel underneath.
		view.Close()
		pumpSidebarEvents(300 * time.Millisecond)
		if closed != 0 || c.window.VisibleDialog() == nil || coreglib.BaseObject(c.window.VisibleDialog()).Native() != coreglib.BaseObject(details).Native() {
			t.Fatal("Details did not persist after preview close")
		}
		if coreglib.BaseObject(details.Child()).Native() != originalChild || stack.VisibleChildName() != "docs" {
			t.Fatal("Details was rebuilt or changed tabs")
		}
		if math.Abs(scroll.VAdjustment().Value()-original) > .5 {
			t.Fatalf("Details scroll moved from %.1f to %.1f", original, scroll.VAdjustment().Value())
		}
	}
}
