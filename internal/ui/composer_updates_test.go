package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func TestComposerBurstCoalescesPreviewAndHeightWork(t *testing.T) {
	c := newFolderSidebar(t)
	builder := gtk.NewBuilder()
	if err := builder.AddFromString(`<interface>
 <object class="GtkScrolledWindow" id="message_entry_scrolled"><property name="child"><object class="GtkTextView" id="entry"/></property></object>
 <object class="GtkButton" id="send"/>
 <object class="GtkBox" id="markdown_preview_bar"/>
 <object class="GtkLabel" id="markdown_preview_label"/>
 </interface>`); err != nil {
		t.Fatal(err)
	}
	entry, _ := castObject[*gtk.TextView](builder, "entry")
	send, _ := castObject[*gtk.Button](builder, "send")
	c.bindInputWidgets(builder, entry, send)
	updates := 0
	handle := c.markdownPreviewLabel.NotifyProperty("label", func() { updates++ })
	defer c.markdownPreviewLabel.HandlerDisconnect(handle)
	buffer := entry.Buffer()
	for i := 0; i < 500; i++ {
		buffer.SetText(fmt.Sprintf("*message%d*", i))
	}
	if updates != 0 {
		t.Fatal("formatting ran synchronously for each keystroke")
	}
	if !c.inputHeightPending {
		t.Fatal("height work was not queued")
	}
	pumpSidebarEvents(130 * time.Millisecond)
	if updates != 1 || c.markdownPreviewLabel.Text() != "message499" {
		t.Fatalf("preview updates=%d text=%q", updates, c.markdownPreviewLabel.Text())
	}
	if c.inputHeightPending {
		t.Fatal("height work did not finish")
	}
}
