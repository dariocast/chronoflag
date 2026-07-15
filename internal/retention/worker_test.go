package retention_test

import (
	"chronograph/internal/retention"
	"chronograph/internal/store"
	"context"
	"testing"
	"time"
)

func TestWorkerArchivesFreeInstanceThenPurgesIt(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	s := store.NewMemory()
	created, _ := s.CreateInstance(ctx, base)
	w := retention.New(s, time.Hour)
	a, p, err := w.RunOnce(ctx, base.Add(24*time.Hour))
	if err != nil || a != 1 || p != 0 {
		t.Fatalf("archive=%d purge=%d err=%v", a, p, err)
	}
	snap, _ := s.Snapshot(ctx, created.InstanceID)
	if snap.Lifecycle != store.Archived {
		t.Fatalf("lifecycle=%s", snap.Lifecycle)
	}
	a, p, err = w.RunOnce(ctx, base.Add(31*24*time.Hour))
	if err != nil || a != 0 || p != 1 {
		t.Fatalf("archive=%d purge=%d err=%v", a, p, err)
	}
	if _, err = s.Snapshot(ctx, created.InstanceID); err != store.ErrNotFound {
		t.Fatalf("purged err=%v", err)
	}
}
