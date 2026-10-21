package app

import (
	"log"
	"os"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func mergeComponents(builder *gtk.Builder) {
	assets := ui.GetAssets()
	components := []struct {
		path     string
		embedded string
	}{
		{"ui/components/ChatsMenu.ui", assets.ChatsMenuUI},
		{"ui/components/CallsMenu.ui", assets.CallsMenuUI},
		{"ui/components/ArchivedMenu.ui", assets.ArchivedMenuUI},
		{"ui/components/SettingsMenu.ui", assets.SettingsMenuUI},
		{"ui/components/RightChatView.ui", assets.RightChatViewUI},
	}
	for _, c := range components {
		xml := c.embedded
		if devMode {
			if data, err := os.ReadFile(c.path); err == nil {
				xml = string(data)
			}
		}
		if err := builder.AddFromString(xml); err != nil {
			log.Printf("merging component %s: %v", c.path, err)
		}
	}
}

func destroyBuilderWindows(builder *gtk.Builder) {
	for _, id := range []string{"initial_window", "syncing_window", "main_window"} {
		if obj := builder.GetObject(id); obj != nil {
			if w, ok := obj.Cast().(interface{ Destroy() }); ok {
				w.Destroy()
			}
		}
	}
}

func castObject[T any](builder *gtk.Builder, id string) (T, bool) {
	var zero T
	obj := builder.GetObject(id)
	if obj == nil {
		return zero, false
	}
	v, ok := obj.Cast().(T)
	return v, ok
}
