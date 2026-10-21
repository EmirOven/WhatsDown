package ui

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

type pollOptWidget struct {
	btn         *gtk.Button
	indicator   *gtk.Image
	countLbl    *gtk.Label
	progressBar *gtk.ProgressBar
}
