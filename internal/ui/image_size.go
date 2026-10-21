package ui

import "math"

func (c *MainController) getMaxImageWidth() int {
	listWidth := 0
	if c.messagesList != nil {
		listWidth = c.messagesList.AllocatedWidth()
	}
	if listWidth <= 0 && c.window != nil {
		listWidth = c.window.AllocatedWidth() - 388
	}
	if listWidth <= 0 {
		listWidth = 560
	}
	normalWidth := listWidth - 100
	if normalWidth < 280 {
		normalWidth = 280
	}
	maxImageWidth := normalWidth / 2
	if maxImageWidth < 180 {
		maxImageWidth = 180
	}
	return maxImageWidth
}

func computeImageDisplaySize(w, h, maxW int) (int, int) {
	if maxW <= 0 {
		maxW = 240
	}
	if w <= 0 || h <= 0 {
		return maxW, int(float64(maxW) * 0.75)
	}
	scale := float64(maxW) / float64(w)
	targetW := maxW
	targetH := int(math.Round(float64(h) * scale))

	maxH := int(float64(maxW) * 1.6)
	if targetH > maxH {
		targetH = maxH
		targetW = int(math.Round(float64(w) * (float64(maxH) / float64(h))))
	}
	if targetW < 100 {
		targetW = 100
	}
	if targetH < 100 {
		targetH = 100
	}
	return targetW, targetH
}
