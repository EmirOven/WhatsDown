package ui

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

func addCSSClasses(w gtk.Widgetter, classes ...string) {
	if w == nil {
		return
	}
	bw := gtk.BaseWidget(w)
	for _, c := range classes {
		bw.AddCSSClass(c)
	}
}
