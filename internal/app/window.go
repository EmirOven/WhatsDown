package app

import (
	"errors"
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func showPage(p page) error {
	xml := p.embeddedUI()
	if devMode {
		if data, err := os.ReadFile(p.uiPath()); err == nil {
			xml = string(data)
		}
	}
	if xml == "" {
		return errors.New("no UI XML available for " + p.uiPath())
	}
	return buildWindow(p, xml)
}

func buildWindow(p page, uiXML string) error {
	builder := gtk.NewBuilder()
	if err := builder.AddFromString(uiXML); err != nil {
		destroyBuilderWindows(builder)
		return err
	}
	if p == pageMain {
		mergeComponents(builder)
	}

	var window *adw.ApplicationWindow
	switch p {
	case pageInitial:
		w, ok1 := castObject[*adw.ApplicationWindow](builder, "initial_window")
		image, ok2 := castObject[*gtk.Image](builder, "qr_image")
		if !ok1 || !ok2 {
			destroyBuilderWindows(builder)
			return errors.New("UI must define initial_window and qr_image")
		}
		if qrTexture != nil {
			image.SetFromPaintable(qrTexture)
		}
		window, qrImage = w, image
		syncProgressBar, syncStatusLabel = nil, nil

	case pageSyncing:
		w, ok1 := castObject[*adw.ApplicationWindow](builder, "syncing_window")
		bar, ok2 := castObject[*gtk.ProgressBar](builder, "sync_progress")
		status, ok3 := castObject[*gtk.Label](builder, "sync_status")
		if !ok1 || !ok2 || !ok3 {
			destroyBuilderWindows(builder)
			return errors.New("UI must define syncing_window, sync_progress, and sync_status")
		}
		window, qrImage = w, nil
		syncProgressBar, syncStatusLabel = bar, status
		renderSync()

	case pageMain:
		w, err := ui.SetupMainWindow(builder, waClient, msgStore)
		if err != nil {
			destroyBuilderWindows(builder)
			return err
		}
		window, qrImage = w, nil
		syncProgressBar, syncStatusLabel = nil, nil
	}

	window.SetApplication(&application.Application)
	window.Present()

	old := currentWindow
	currentWindow, currentPage = window, p
	if old != nil {
		old.Destroy()
	}
	return nil
}
