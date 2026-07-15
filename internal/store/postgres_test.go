package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"chronograph/internal/clock"
	"chronograph/internal/store"
)

func TestPostgresStoreRoundTrip(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := store.NewPostgres(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateInstance(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cap, err := s.ResolveCapability(ctx, created.ControlToken)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot(ctx, created.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	snap, _, err = s.ApplyCommand(ctx, cap, snap.Clocks[0].ID, clock.Command{ID: "pg-start", Type: clock.Start}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Clocks[0].State != clock.Running {
		t.Fatalf("state=%s", snap.Clocks[0].State)
	}
	raw, err := s.RawCapabilityCount(ctx, created.ControlToken)
	if err != nil {
		t.Fatal(err)
	}
	if raw != 0 {
		t.Fatal("raw capability token persisted")
	}
}
