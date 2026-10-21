package app

import (
	"context"
	"log"
	"path/filepath"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var liveReloadMonitor gio.FileMonitorrer

// startLiveReload watches the ui/ directory and applies changes in-process.
func startLiveReload(app *adw.Application, css *gtk.CSSProvider) {
	monitor, err := gio.NewFileForPath(uiDir).MonitorDirectory(context.Background(), gio.FileMonitorWatchMoves)
	if err != nil {
		log.Printf("[live-reload] unavailable: %v", err)
		return
	}
	liveReloadMonitor = monitor

	pending := map[string]bool{}
	generation := 0

	flush := func() {
		if pending[".blp"] {
			go compileBlueprints()
		}
		if pending[".ui"] {
			reloadWindow()
		}
		if pending[".css"] {
			log.Printf("[live-reload] reloading stylesheets")
			loadCSS(css)
		}
		clear(pending)
	}

	gio.BaseFileMonitor(monitor).ConnectChanged(func(file, other gio.Filer, event gio.FileMonitorEvent) {
		name := file.Basename()
		switch event {
		case gio.FileMonitorEventChangesDoneHint, gio.FileMonitorEventCreated, gio.FileMonitorEventMovedIn:
		case gio.FileMonitorEventRenamed:
			if other == nil {
				return
			}
			name = other.Basename()
		default:
			return
		}

		ext := filepath.Ext(name)
		if ext != ".blp" && ext != ".ui" && ext != ".css" {
			return
		}
		pending[ext] = true

		generation++
		mine := generation
		glib.TimeoutAdd(150, func() {
			if mine == generation {
				flush()
			}
		})
	})

	log.Printf("[live-reload] watching %s/ for .blp, .ui and .css changes", uiDir)
}
