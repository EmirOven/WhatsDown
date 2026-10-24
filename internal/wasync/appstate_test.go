package wasync

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.mau.fi/whatsmeow/appstate"
)

type recoveryClient struct {
	calls                 int
	fetches               []appstate.WAPatchName
	sendErr, errorOnFetch error
}

func (c *recoveryClient) SendAppState(context.Context, appstate.PatchInfo) error {
	c.calls++
	if c.calls == 1 {
		return c.sendErr
	}
	return nil
}
func (c *recoveryClient) FetchAppState(_ context.Context, name appstate.WAPatchName, full, onlyIfMissing bool) error {
	if !full || onlyIfMissing {
		return errors.New("recovery did not request a fresh authenticated snapshot")
	}
	c.fetches = append(c.fetches, name)
	return c.errorOnFetch
}
func TestRecoverInvalidReadStateCollectionOnce(t *testing.T) {
	c := &recoveryClient{sendErr: fmt.Errorf("409 conflict: failed to verify patch: %w", appstate.ErrMismatchingLTHash)}
	patch := appstate.PatchInfo{Type: appstate.WAPatchRegularLow}
	if err := SendAppState(context.Background(), c, patch); err != nil {
		t.Fatal(err)
	}
	if c.calls != 2 || len(c.fetches) != 1 || c.fetches[0] != patch.Type {
		t.Fatalf("wrong recovery: %+v", c)
	}
}
func TestRecoveryDoesNotHideSnapshotFailure(t *testing.T) {
	failure := errors.New("phone unavailable")
	c := &recoveryClient{sendErr: appstate.ErrMismatchingPatchMAC, errorOnFetch: failure}
	if err := SendAppState(context.Background(), c, appstate.PatchInfo{Type: appstate.WAPatchRegularLow}); !errors.Is(err, failure) {
		t.Fatalf("wrong recovery error: %v", err)
	}
	if c.calls != 1 {
		t.Fatal("sent against an unverified state")
	}
}
func TestOrdinaryFailuresDoNotResetSyncState(t *testing.T) {
	failure := errors.New("network disconnected")
	c := &recoveryClient{sendErr: failure}
	if err := SendAppState(context.Background(), c, appstate.PatchInfo{}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if c.calls != 1 || len(c.fetches) != 0 {
		t.Fatal("ordinary failure reset app state")
	}
}
