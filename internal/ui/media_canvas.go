package ui

import (
	"fmt"
	"math"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/graphene"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type mediaCanvas struct {
	Scroll   *gtk.ScrolledWindow
	Picture  *gtk.Picture
	Controls *gtk.Box
	texture  *gdk.Texture
	Zoom     float64
	Rotation int
	label    *gtk.Label
	panning  bool
}

func (c *MainController) newMediaCanvas() *mediaCanvas {
	v := &mediaCanvas{Scroll: gtk.NewScrolledWindow(), Picture: gtk.NewPicture(), Zoom: 1}
	v.Scroll.SetHExpand(true)
	v.Scroll.SetVExpand(true)
	v.Scroll.SetKineticScrolling(false)
	v.Picture.SetKeepAspectRatio(true)
	v.Picture.SetCanShrink(true)
	v.Picture.SetHExpand(true)
	v.Picture.SetVExpand(true)
	v.Scroll.SetChild(v.Picture)
	bar := gtk.NewBox(gtk.OrientationHorizontal, 8)
	bar.SetHAlign(gtk.AlignCenter)
	bar.SetMarginTop(8)
	bar.SetMarginBottom(12)
	bar.SetMarginStart(12)
	bar.SetMarginEnd(12)
	v.Controls = bar
	linked := gtk.NewBox(gtk.OrientationHorizontal, 0)
	linked.AddCSSClass("linked")
	add := func(box *gtk.Box, icon, tip string, action func()) {
		b := gtk.NewButtonFromIconName(icon)
		b.SetTooltipText(tip)
		c.trackWidgetSignal(v.Scroll, b, b.ConnectClicked(action))
		box.Append(b)
	}
	add(linked, "zoom-out-symbolic", "Zoom out", func() { v.SetZoom(v.Zoom / 1.25) })
	add(linked, "zoom-in-symbolic", "Zoom in", func() { v.SetZoom(v.Zoom * 1.25) })
	bar.Append(linked)
	v.label = gtk.NewLabel("Fit")
	v.label.SetWidthChars(6)
	bar.Append(v.label)
	add(bar, "zoom-fit-best-symbolic", "Fit to window", func() { v.SetZoom(1) })
	add(bar, "object-rotate-right-symbolic", "Rotate clockwise", func() { v.Rotate() })
	wheel := gtk.NewEventControllerScroll(gtk.EventControllerScrollVertical)
	wheel.SetPropagationPhase(gtk.PhaseCapture)
	c.trackWidgetSignal(v.Scroll, wheel, wheel.ConnectScroll(func(_, dy float64) bool {

		v.ZoomByScroll(dy)
		return true
	}))
	v.Scroll.AddController(wheel)
	drag := gtk.NewGestureDrag()
	drag.SetButton(1)
	drag.SetPropagationPhase(gtk.PhaseCapture)
	var x, y float64
	c.trackWidgetSignal(v.Scroll, drag, drag.ConnectDragBegin(func(px, py float64) {
		v.panning = false
		for target := v.Scroll.Pick(px, py, gtk.PickDefault); target != nil; target = gtk.BaseWidget(target).Parent() {
			if _, ok := target.(*gtk.Scrollbar); ok {
				drag.SetState(gtk.EventSequenceDenied)
				return
			}
		}
		horizontal, vertical := v.Scroll.HAdjustment(), v.Scroll.VAdjustment()
		if v.Zoom <= 1 || (horizontal.Upper() <= horizontal.PageSize() && vertical.Upper() <= vertical.PageSize()) {
			drag.SetState(gtk.EventSequenceDenied)
			return
		}
		x, y = horizontal.Value(), vertical.Value()
		v.panning = true
		drag.SetState(gtk.EventSequenceClaimed)
		v.Picture.SetCursorFromName("grabbing")
	}))
	c.trackWidgetSignal(v.Scroll, drag, drag.ConnectDragUpdate(func(dx, dy float64) {
		if !v.panning {
			return
		}
		setPanAdjustment(v.Scroll.HAdjustment(), x-dx)
		setPanAdjustment(v.Scroll.VAdjustment(), y-dy)
	}))
	c.trackWidgetSignal(v.Scroll, drag, drag.ConnectDragEnd(func(_, _ float64) {
		v.panning = false
		if v.Zoom > 1 {
			v.Picture.SetCursorFromName("grab")
		}
	}))
	// Gesture offsets must stay in viewport coordinates: the picture moves as we pan.
	v.Scroll.AddController(drag)
	return v
}
func (v *mediaCanvas) SetTexture(texture *gdk.Texture) {
	v.texture = texture
	v.updateSize()
}
func (v *mediaCanvas) Rotate() {
	v.panning = false
	v.Rotation = (v.Rotation + 90) % 360
	v.updateSize()
}
func (v *mediaCanvas) SetZoom(zoom float64) {
	v.panning = false
	v.Zoom = math.Max(.25, math.Min(8, zoom))
	v.updateSize()
}
func (v *mediaCanvas) dimensions() (float64, float64) {
	if v.texture == nil {
		return 1, 1
	}
	w, h := float64(v.texture.Width()), float64(v.texture.Height())
	if v.Rotation%180 != 0 {
		return h, w
	}
	return w, h
}
func (v *mediaCanvas) updatePaintable(scale float64) {
	if v.texture == nil {
		return
	}
	if v.Rotation == 0 && scale == 1 {
		v.Picture.SetPaintable(v.texture)
		return
	}
	w, h := float32(v.texture.Width()), float32(v.texture.Height())
	rw, rh := v.dimensions()
	snapshot := gtk.NewSnapshot()
	point := graphene.NewPointAlloc()
	snapshot.Translate(point.Init(float32(rw*scale/2), float32(rh*scale/2)))
	snapshot.Scale(float32(scale), float32(scale))
	snapshot.Rotate(float32(v.Rotation))
	snapshot.Translate(point.Init(-w/2, -h/2))
	rect := graphene.RectAlloc()
	snapshot.AppendTexture(v.texture, rect.Init(0, 0, w, h))
	size := graphene.NewSizeAlloc()
	v.Picture.SetPaintable(snapshot.ToPaintable(size.Init(float32(rw*scale), float32(rh*scale))))
}
func (v *mediaCanvas) updateSize() {
	if v.texture == nil {
		return
	}
	if math.Abs(v.Zoom-1) < .001 {
		v.Zoom = 1
		v.updatePaintable(1)
		v.Picture.SetHAlign(gtk.AlignFill)
		v.Picture.SetVAlign(gtk.AlignFill)
		v.Picture.SetHExpand(true)
		v.Picture.SetVExpand(true)
		v.Picture.SetCanShrink(true)
		v.Picture.SetSizeRequest(-1, -1)
		v.label.SetText("Fit")
		v.Picture.SetCursorFromName("default")
		return
	}
	w, h := v.dimensions()
	fit := math.Min(float64(max(1, v.Scroll.Width()-16))/w, float64(max(1, v.Scroll.Height()-16))/h)
	scale := fit * v.Zoom
	v.updatePaintable(scale)
	v.Picture.SetHAlign(gtk.AlignCenter)
	v.Picture.SetVAlign(gtk.AlignCenter)
	v.Picture.SetHExpand(false)
	v.Picture.SetVExpand(false)
	v.Picture.SetCanShrink(false)
	v.Picture.SetSizeRequest(max(1, int(w*scale)), max(1, int(h*scale)))
	v.label.SetText(fmt.Sprintf("%.0f%%", scale*100))
	if v.Zoom > 1 {
		v.Picture.SetCursorFromName("grab")
	} else {
		v.Picture.SetCursorFromName("default")
	}
}

func (v *mediaCanvas) ZoomByScroll(dy float64) {
	if dy != 0 {
		v.SetZoom(v.Zoom * math.Pow(1.15, -dy))
	}
}

func setPanAdjustment(adjustment *gtk.Adjustment, value float64) {
	adjustment.SetValue(math.Max(adjustment.Lower(), math.Min(math.Max(adjustment.Lower(), adjustment.Upper()-adjustment.PageSize()), value)))
}
