package main

import (
	"context"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Kept at package level so the monitor isn't garbage-collected.
var liveReloadMonitor gio.FileMonitorrer

// startLiveReload watches the ui/ directory and applies changes in-process:
//
//	*.blp -> recompiled with blueprint-compiler (which rewrites the .ui file)
//	*.ui  -> the window is rebuilt
//	*.css -> the stylesheet is reloaded
//
// Go source changes are handled by run.sh, which rebuilds and restarts the app.
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
			// Atomic saves (write temp file, rename over target): the
			// target name is in otherFile.
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

		// Debounce: editors and compilers emit bursts of events per save.
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

func compileBlueprints() {
	files, _ := filepath.Glob(filepath.Join(uiDir, "*.blp"))
	if len(files) == 0 {
		return
	}
	log.Printf("[live-reload] compiling blueprints")
	args := append([]string{"batch-compile", uiDir, uiDir}, files...)
	out, err := exec.Command("blueprint-compiler", args...).CombinedOutput()
	if err != nil {
		log.Printf("[live-reload] blueprint-compiler failed: %v\n%s", err, out)
	}
	// On success the .ui file changes, which triggers reloadWindow.
}

func reloadWindow() {
	if err := showPage(currentPage); err != nil {
		log.Printf("[live-reload] UI error, keeping previous window: %v", err)
		return
	}
	log.Printf("[live-reload] window rebuilt from %s", currentPage.uiPath())
}
