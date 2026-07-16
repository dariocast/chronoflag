package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"chronograph/internal/clock"
	"chronograph/internal/store"
)

func TestMemoryStoreCapabilitiesAndCommands(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	s := store.NewMemory()
	created, err := s.CreateInstance(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if created.ControlToken == created.ViewToken || created.ControlToken == "" {
		t.Fatal("capabilities are not distinct")
	}
	control, _ := s.ResolveCapability(ctx, created.ControlToken)
	view, _ := s.ResolveCapability(ctx, created.ViewToken)
	if control.Scope != store.Control || view.Scope != store.View {
		t.Fatalf("wrong scopes: %#v %#v", control, view)
	}
	snap, _ := s.Snapshot(ctx, created.InstanceID)
	if len(snap.Clocks) != 1 || snap.Clocks[0].Type != clock.Stopwatch {
		t.Fatalf("unexpected initial snapshot: %#v", snap)
	}
	clockID := snap.Clocks[0].ID
	result, _, err := s.ApplyCommand(ctx, control, clockID, clock.Command{ID: "start", Type: clock.Start}, now)
	if err != nil || result.Clocks[0].State != clock.Running {
		t.Fatalf("start: %#v %v", result, err)
	}
	duplicate, _, err := s.ApplyCommand(ctx, control, clockID, clock.Command{ID: "start", Type: clock.Start}, now.Add(time.Second))
	if err != nil || duplicate.Clocks[0].Version != 1 {
		t.Fatalf("idempotency failed: %#v %v", duplicate, err)
	}
	if _, _, err := s.ApplyCommand(ctx, view, clockID, clock.Command{ID: "pause", Type: clock.Pause}, now); err != store.ErrForbidden {
		t.Fatalf("view mutation error=%v", err)
	}
}

func TestMemoryStoreSerializesConcurrentCommands(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := store.NewMemory()
	created, _ := s.CreateInstance(ctx, now)
	cap, _ := s.ResolveCapability(ctx, created.ControlToken)
	snap, _ := s.Snapshot(ctx, created.InstanceID)
	id := snap.Clocks[0].ID
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = s.ApplyCommand(ctx, cap, id, clock.Command{ID: time.Now().String(), Type: clock.Start}, now)
		}()
	}
	wg.Wait()
	snap, _ = s.Snapshot(ctx, created.InstanceID)
	if snap.Clocks[0].Version != 1 {
		t.Fatalf("version=%d, want 1", snap.Clocks[0].Version)
	}
}

func TestMemoryStoreAddsIndependentTimerAndUpdatesBoard(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := store.NewMemory()
	created, _ := s.CreateInstance(ctx, now)
	cap, _ := s.ResolveCapability(ctx, created.ControlToken)
	snap, err := s.AddClock(ctx, cap, clock.Timer, 5*time.Minute, now)
	if err != nil || len(snap.Clocks) != 2 || snap.Clocks[1].Duration != 5*time.Minute {
		t.Fatalf("add timer: %#v %v", snap, err)
	}
	snap, err = s.UpdateClock(ctx, cap, snap.Clocks[1].ID, store.ClockPatch{Label: ptr("HEAT 2"), Highlighted: ptr(true)}, now)
	if err != nil || snap.HighlightedClockID != snap.Clocks[1].ID || snap.Clocks[1].Label != "HEAT 2" {
		t.Fatalf("update: %#v %v", snap, err)
	}
	movedID := snap.Clocks[1].ID
	snap, err = s.UpdateClock(ctx, cap, movedID, store.ClockPatch{Order: ptr(0)}, now)
	if err != nil || snap.Clocks[0].ID != movedID {
		t.Fatalf("reorder: %#v %v", snap, err)
	}
}

func TestMemoryStoreExpiresTimersWithoutAConnectedController(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := store.NewMemory()
	created, _ := s.CreateInstance(ctx, now)
	cap, _ := s.ResolveCapability(ctx, created.ControlToken)
	snap, _ := s.AddClock(ctx, cap, clock.Timer, time.Second, now)
	timer := snap.Clocks[1]
	snap, _, _ = s.ApplyCommand(ctx, cap, timer.ID, clock.Command{ID: "start-timer", Type: clock.Start}, now)
	changed, err := s.ExpireDue(ctx, now.Add(time.Second))
	if err != nil || len(changed) != 1 {
		t.Fatalf("changed=%d err=%v", len(changed), err)
	}
	if changed[0].Clocks[1].State != clock.Expired {
		t.Fatalf("state=%s", changed[0].Clocks[1].State)
	}
}

func TestMemoryStoreUpdatesTitleAndRotatesCapabilities(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemory()
	created, _ := s.CreateInstance(ctx, time.Now())
	control, _ := s.ResolveCapability(ctx, created.ControlToken)
	updated, err := s.UpdateInstance(ctx, control, store.InstancePatch{Title: ptr("City relay")}, time.Now())
	if err != nil || updated.Title != "City relay" {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	replacement, err := s.RotateCapability(ctx, control, store.Control)
	if err != nil || replacement.ControlToken == "" || replacement.ControlToken == created.ControlToken {
		t.Fatalf("replacement=%#v err=%v", replacement, err)
	}
	if _, err := s.ResolveCapability(ctx, created.ControlToken); err != store.ErrNotFound {
		t.Fatalf("old token error=%v", err)
	}
	if _, err := s.ResolveCapability(ctx, replacement.ControlToken); err != nil {
		t.Fatal(err)
	}
}

func ptr[T any](v T) *T { return &v }
