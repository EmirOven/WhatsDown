package emoji

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed emojis.json
var rawJSON []byte

var (
	catalog  []Item
	initOnce sync.Once
)

// All returns the full catalog of unicode emojis.
func All() []Item {
	initOnce.Do(func() {
		_ = json.Unmarshal(rawJSON, &catalog)
	})
	return catalog
}
