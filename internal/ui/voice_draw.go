package ui

import (
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/audio"
)

// drawVoiceMeter renders the audio level meter bars into the recording bar drawing area.
func (c *MainController) drawVoiceMeter(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	barWidth := 3.0
	barGap := 2.0
	step := barWidth + barGap
	totalBars := int(float64(width) / step)
	if totalBars <= 0 {
		return
	}

	var levels []float64
	if c.voiceRecorder != nil {
		levels = c.voiceRecorder.Levels()
	}

	isRecording := c.voiceRecorder != nil && !c.voiceRecorder.Stopped()
	isPaused := c.voiceRecorder != nil && c.voiceRecorder.Paused()

	r, g, b, a := 0.0, 0.66, 0.52, 0.95
	if isPaused {
		a = 0.5
	} else if !isRecording {
		r, g, b, a = 0.2, 0.6, 0.86, 0.85
	}
	cr.SetSourceRGBA(r, g, b, a)

	if isRecording {
		startIdx := len(levels) - totalBars
		for i := 0; i < totalBars; i++ {
			idx := startIdx + i
			lvl := 0.08
			if idx >= 0 && idx < len(levels) {
				lvl = math.Max(0.08, math.Min(1.0, levels[idx]))
			}
			h := float64(height) * lvl
			x := float64(i) * step
			y := (float64(height) - h) / 2.0
			drawRoundedBar(cr, x, y, barWidth, h)
		}
	} else if len(levels) > 0 {
		waveformLevels := audio.ResampleLevels(levels, totalBars)
		peak := 0.0
		for _, v := range waveformLevels {
			peak = math.Max(peak, v)
		}
		for i := 0; i < totalBars; i++ {
			lvl := 0.08
			if peak > 0 && i < len(waveformLevels) {
				lvl = math.Max(0.08, math.Min(1.0, waveformLevels[i]/peak))
			}
			h := float64(height) * lvl
			x := float64(i) * step
			y := (float64(height) - h) / 2.0
			drawRoundedBar(cr, x, y, barWidth, h)
		}
	} else {
		for i := 0; i < totalBars; i++ {
			h := float64(height) * 0.08
			x := float64(i) * step
			y := (float64(height) - h) / 2.0
			drawRoundedBar(cr, x, y, barWidth, h)
		}
	}
}

func drawRoundedBar(cr *cairo.Context, x, y, w, h float64) {
	radius := w / 2.0
	if h < radius*2 {
		cr.Rectangle(x, y, w, h)
		cr.Fill()
		return
	}
	cr.NewPath()
	cr.Arc(x+radius, y+radius, radius, math.Pi, 0)
	cr.Arc(x+radius, y+h-radius, radius, 0, math.Pi)
	cr.ClosePath()
	cr.Fill()
}
