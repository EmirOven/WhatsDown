package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

type mediaViewer struct {
	widget  *adw.ToolbarView
	actions map[string]*gio.SimpleAction
	Close   func()
	Canvas  *mediaCanvas
}

func (c *MainController) openMediaViewer(msg store.StoredMessage) *mediaViewer {
	return c.buildMediaViewer(msg, func(ctx context.Context) (string, error) { return media.DownloadMedia(ctx, c.client, msg) })
}

// Overlay the existing root content, retaining the conversation and its scroll position.
func (c *MainController) buildMediaViewer(msg store.StoredMessage, load func(context.Context) (string, error)) *mediaViewer {
	if c.mediaViewer != nil {
		c.mediaViewer.Close()
	}
	if c.rootOverlay == nil {
		base := c.window.Content()
		c.window.SetContent(nil)
		c.rootOverlay = gtk.NewOverlay()
		if base != nil {
			c.rootOverlay.SetChild(base)
		}
		c.window.SetContent(c.rootOverlay)
	}
	previousFocus := c.window.Focus()
	toolbar := adw.NewToolbarView()
	toolbar.SetHExpand(true)
	toolbar.SetVExpand(true)
	toolbar.AddCSSClass("media-viewer")
	header := adw.NewHeaderBar()
	title := "Photo"
	if msg.Kind == store.MessageKindVideo {
		title = "Video"
	}
	if msg.Kind == store.MessageKindDocument {
		title = "PDF document"
		if raw := media.ParseE2EMessage(msg.Raw); raw != nil && raw.GetDocumentMessage().GetFileName() != "" {
			title = raw.GetDocumentMessage().GetFileName()
		}
	}
	header.SetTitleWidget(adw.NewWindowTitle(title, ""))
	back := gtk.NewButtonFromIconName("go-previous-symbolic")
	back.SetTooltipText("Back to conversation")
	header.PackStart(back)
	for _, button := range []struct{ icon, tip, action string }{
		{"mail-reply-sender-symbolic", "Reply", "media.reply"},
		{"mail-forward-symbolic", "Forward", "media.forward"},
	} {
		b := gtk.NewButtonFromIconName(button.icon)
		b.SetTooltipText(button.tip)
		b.SetActionName(button.action)
		header.PackStart(b)
	}
	menuButton := gtk.NewMenuButton()
	menuButton.SetIconName("view-more-symbolic")
	menuButton.SetTooltipText("Media actions")
	header.PackEnd(menuButton)
	toolbar.AddTopBar(header)
	content := gtk.NewBox(gtk.OrientationVertical, 8)
	status := gtk.NewLabel("Loading media…")
	status.SetWrap(true)
	status.SetMarginTop(12)
	content.Append(status)
	body := gtk.NewBox(gtk.OrientationVertical, 8)
	body.SetHExpand(true)
	body.SetVExpand(true)
	content.Append(body)
	toolbar.SetContent(content)
	view := &mediaViewer{widget: toolbar, actions: make(map[string]*gio.SimpleAction)}
	ctx, cancel := context.WithCancel(context.Background())
	alive := true
	var stream *gtk.MediaFile
	var timer glib.SourceHandle
	view.Close = func() {
		if !alive {
			return
		}
		alive = false
		cancel()
		if stream != nil {
			stream.Pause()
			stream.Clear()
		}
		if timer != 0 {
			glib.SourceRemove(timer)
			timer = 0
		}
		c.releaseMessageWidget(toolbar)
		c.rootOverlay.RemoveOverlay(toolbar)
		if c.mediaViewer == view {
			c.mediaViewer = nil
		}
		if previousFocus != nil && gtk.BaseWidget(previousFocus).Root() != nil {
			gtk.BaseWidget(previousFocus).GrabFocus()
		}
	}
	c.trackWidgetSignal(toolbar, back, back.ConnectClicked(view.Close))
	c.trackWidgetSignal(toolbar, c.window, c.window.ConnectDestroy(view.Close))
	var path string
	group := gio.NewSimpleActionGroup()
	toolbar.InsertActionGroup("media", group)
	menu := gio.NewMenu()
	messageMenu, fileMenu := gio.NewMenu(), gio.NewMenu()
	add := func(section *gio.Menu, name, label string, callback func()) *gio.SimpleAction {
		action := gio.NewSimpleAction(name, nil)
		c.trackWidgetSignal(toolbar, action, action.ConnectActivate(func(*glib.Variant) { callback() }))
		group.AddAction(action)
		if section != nil {
			section.Append(label, "media."+name)
		}
		view.actions[name] = action
		return action
	}
	add(nil, "reply", "_Reply", func() { view.Close(); c.jumpToMessage(msg.ChatJID, msg.ID); c.startReply(msg) })
	add(nil, "forward", "_Forward…", func() { view.Close(); c.showForwardMessages([]store.StoredMessage{msg}) })
	add(messageMenu, "go-to-message", "_Go to Message", func() { view.Close(); c.jumpToMessage(msg.ChatJID, msg.ID) })
	external := add(fileMenu, "open-external", "Open in _Default App", func() {
		if err := media.OpenFile(path); err != nil {
			status.SetText(err.Error())
			status.SetVisible(true)
		}
	})
	external.SetEnabled(false)
	files := add(fileMenu, "show-in-files", "_Show in Files", func() { media.ShowMediaInFileManager(c.client, msg) })
	files.SetEnabled(false)
	menu.AppendSection("", messageMenu)
	menu.AppendSection("", fileMenu)
	menuButton.SetMenuModel(menu)
	keys := gtk.NewEventControllerKey()
	keys.SetPropagationPhase(gtk.PhaseCapture)
	c.trackWidgetSignal(toolbar, keys, keys.ConnectKeyPressed(func(key, _ uint, _ gdk.ModifierType) bool {
		if c.window.VisibleDialog() != nil {
			return false
		}
		if key == gdk.KEY_Escape {
			view.Close()
			return true
		}
		return false
	}))
	c.window.AddController(keys)
	c.trackWidgetCleanup(toolbar, func() { c.window.RemoveController(keys) })
	c.mediaViewer = view
	c.rootOverlay.AddOverlay(toolbar)
	c.window.Present()
	back.GrabFocus()
	go func() {
		downloaded, err := load(ctx)
		if err != nil {
			glib.IdleAdd(func() {
				if alive {
					status.SetText("Could not load media: " + err.Error())
				}
			})
			return
		}
		pages := 1
		var texture *gdk.Texture
		if msg.Kind == store.MessageKindDocument {
			pages = media.GetPDFPageCount(downloaded)
			texture = media.GetPDFPageTexture(downloaded, 1)
		} else if msg.Kind == store.MessageKindImage {
			texture, err = gdk.NewTextureFromFilename(downloaded)
		}
		glib.IdleAdd(func() {
			if !alive {
				return
			}
			path = downloaded
			external.SetEnabled(true)
			files.SetEnabled(true)
			status.SetVisible(false)
			if err != nil {
				status.SetText(err.Error())
				status.SetVisible(true)
				return
			}
			if msg.Kind == store.MessageKindVideo {
				stream = gtk.NewMediaFileForFilename(path)
				video := gtk.NewVideoForMediaStream(stream)
				video.SetAutoplay(false)
				video.SetHExpand(true)
				video.SetVExpand(true)
				body.Append(video)
				stream.Play()
				timer = glib.TimeoutAdd(250, func() bool {
					if !alive {
						return false
					}
					if err := stream.Error(); err != nil {
						status.SetText("Could not play video: " + err.Error())
						status.SetVisible(true)
						timer = 0
						return false
					}
					return true
				})
				return
			}
			if texture == nil {
				status.SetText("Could not render this file. Use Open in Default App from the media menu.")
				status.SetVisible(true)
				return
			}
			canvas := c.newMediaCanvas()
			view.Canvas = canvas
			canvas.SetTexture(texture)
			body.Append(canvas.Scroll)
			toolbar.AddBottomBar(canvas.Controls)
			if msg.Kind == store.MessageKindDocument {
				page := 1
				switcher := buildPDFSwitcher(path, func(tex *gdk.Texture) {
					if alive {
						canvas.SetTexture(tex)
					}
				}, pages, &page)
				switcher.SetMarginBottom(12)
				toolbar.AddBottomBar(switcher)
			}
		})
	}()
	return view
}
