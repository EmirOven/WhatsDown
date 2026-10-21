package ui

import (
	"hash/fnv"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var senderColorPalette = []string{
	"#E542A3", "#E55A5A", "#F0821E", "#D97706",
	"#16A34A", "#0D9488", "#0284C7", "#4F46E5",
	"#7C3AED", "#C026D3", "#059669", "#2563EB",
	"#9333EA", "#EA580C", "#0891B2", "#475569",
}

var senderColorPaletteDark = []string{
	"#F472B6", "#F87171", "#FB923C", "#FBBF24",
	"#4ADE80", "#2DD4BF", "#38BDF8", "#818CF8",
	"#A78BFA", "#E879F9", "#34D399", "#60A5FA",
	"#C084FC", "#FB923C", "#22D3EE", "#94A3B8",
}

func isDarkMode() bool {
	if !gtk.IsInitialized() {
		return false
	}
	sm := adw.StyleManagerGetDefault()
	if sm != nil {
		return sm.Dark()
	}
	return false
}

func getSenderColor(seed string) string {
	if seed == "" {
		seed = "member"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	palette := senderColorPalette
	if isDarkMode() {
		palette = senderColorPaletteDark
	}
	idx := h.Sum32() % uint32(len(palette))
	return palette[idx]
}
