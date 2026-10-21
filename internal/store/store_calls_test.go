package store_test

import (
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestCallsStore(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	call := store.CallRecord{
		ID:           "call-1",
		ChatJID:      "123@s.whatsapp.net",
		CallerJID:    "123@s.whatsapp.net",
		CallerName:   "Caller",
		Timestamp:    time.Now(),
		DurationSecs: 45,
		IsVideo:      true,
		IsGroup:      false,
		Outcome:      store.CallOutcomeConnected,
		FromMe:       false,
	}

	if err := s.SaveCall(ctx, call); err != nil {
		t.Fatalf("saveCall: %v", err)
	}

	calls, err := s.ListCalls(ctx, "")
	if err != nil || len(calls) != 1 || calls[0].DurationSecs != 45 {
		t.Fatalf("unexpected calls: %+v (err: %v)", calls, err)
	}

	if err := s.DeleteCall(ctx, "call-1"); err != nil {
		t.Fatalf("deleteCall: %v", err)
	}
	calls, _ = s.ListCalls(ctx, "")
	if len(calls) != 0 {
		t.Errorf("expected 0 calls after delete")
	}
}
