package ui

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

func castObject[T any](builder *gtk.Builder, id string) (T, bool) {
	obj := builder.GetObject(id)
	if obj == nil {
		var zero T
		return zero, false
	}
	val, ok := obj.Cast().(T)
	return val, ok
}
