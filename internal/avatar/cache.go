package avatar

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/cache"
)

var (
	mu            sync.RWMutex
	memoryCache   = cache.NewLRU[*gdk.Texture](16<<20, 256)
	negativeCache = make(map[string]time.Time)
	inFlight      = make(map[string][]func(*gdk.Texture))
	fetchSem      = make(chan struct{}, 2)
)

func CacheDir() string {
	dir := filepath.Join(glib.GetUserDataDir(), "whatsdown", "avatars")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func SanitizeJIDFilename(jid string) string {
	clean := strings.ReplaceAll(jid, "/", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	return clean
}

func DiskPath(jid string) string {
	return filepath.Join(CacheDir(), SanitizeJIDFilename(jid)+".jpg")
}

// Invalidate removes the cached avatar for the given JID.
func Invalidate(jid string) {
	mu.Lock()
	memoryCache.Delete(jid)
	delete(negativeCache, jid)
	mu.Unlock()
	_ = os.Remove(DiskPath(jid))
}

// StoreBytes saves raw avatar bytes to disk and clears the in-memory/negative cache for the JID.
func StoreBytes(jid string, data []byte) {
	if len(data) == 0 {
		return
	}
	_ = os.WriteFile(DiskPath(jid), data, 0o600)
	mu.Lock()
	memoryCache.Delete(jid)
	delete(negativeCache, jid)
	mu.Unlock()
}

// ClearCache removes disk cached avatars and empties the in-memory texture cache.
func ClearCache() {
	cacheDir := CacheDir()
	_ = os.RemoveAll(cacheDir)
	_ = os.MkdirAll(cacheDir, 0o700)

	mu.Lock()
	memoryCache.Clear()
	negativeCache = make(map[string]time.Time)
	mu.Unlock()
}
