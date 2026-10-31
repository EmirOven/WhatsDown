package wasync

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
)

type AppStateClient interface {
	SendAppState(context.Context, appstate.PatchInfo) error
}
type stateFetcher interface {
	FetchAppState(context.Context, appstate.WAPatchName, bool, bool) error
}

// App-state writes share versions and hashes. Serialize writers across UI actions.
var writeMu sync.Mutex

func SendAppState(ctx context.Context, client AppStateClient, patch appstate.PatchInfo) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	err := client.SendAppState(ctx, patch)
	if err == nil {
		return nil
	}
	if !errors.Is(err, appstate.ErrMismatchingLTHash) && !errors.Is(err, appstate.ErrMismatchingPatchMAC) && !errors.Is(err, whatsmeow.ErrAppStateUpdate) {
		return err
	}
	fetcher, ok := client.(stateFetcher)
	if !ok {
		return err
	}
	if errors.Is(err, whatsmeow.ErrAppStateUpdate) {
		if fetchErr := fetcher.FetchAppState(ctx, patch.Type, false, false); fetchErr == nil {
			return client.SendAppState(ctx, patch)
		}
	}
	// Rebuild only the invalid collection from WhatsApp's authenticated snapshot.
	if recoveryErr := fetcher.FetchAppState(ctx, patch.Type, true, false); recoveryErr != nil {
		return fmt.Errorf("recovering WhatsApp sync state: %w", recoveryErr)
	}
	return client.SendAppState(ctx, patch)
}
