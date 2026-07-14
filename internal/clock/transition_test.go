package clock_test

import (
	"testing"
	"time"

	"chronograph/internal/clock"
)

func TestStopwatchTransitionsAndLaps(t *testing.T) {
	t0 := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	c := clock.NewStopwatch("clock-1")

	running, start, err := clock.Apply(c, clock.Command{ID: "start-1", Type: clock.Start}, t0)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if running.State != clock.Running || start.Type != clock.Started {
		t.Fatalf("unexpected start result: state=%s event=%s", running.State, start.Type)
	}
	if got := clock.ElapsedAt(running, t0.Add(1234*time.Millisecond)); got != 1234*time.Millisecond {
		t.Fatalf("elapsed=%s, want 1.234s", got)
	}

	withLap, lapEvent, err := clock.Apply(running, clock.Command{ID: "lap-1", Type: clock.Lap}, t0.Add(2*time.Second))
	if err != nil {
		t.Fatalf("lap: %v", err)
	}
	if lapEvent.Type != clock.Lapped || len(withLap.Laps) != 1 {
		t.Fatalf("lap not recorded: %#v", withLap.Laps)
	}
	if withLap.Laps[0].Number != 1 || withLap.Laps[0].Elapsed != 2*time.Second || withLap.Laps[0].Split != 2*time.Second {
		t.Fatalf("unexpected first lap: %#v", withLap.Laps[0])
	}

	withLap, _, err = clock.Apply(withLap, clock.Command{ID: "lap-2", Type: clock.Lap}, t0.Add(2750*time.Millisecond))
	if err != nil {
		t.Fatalf("second lap: %v", err)
	}
	if withLap.Laps[1].Split != 750*time.Millisecond {
		t.Fatalf("second split=%s, want 750ms", withLap.Laps[1].Split)
	}
}

func TestPauseResumeAndReset(t *testing.T) {
	t0 := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	c := clock.NewStopwatch("clock-1")
	c, _, _ = clock.Apply(c, clock.Command{ID: "1", Type: clock.Start}, t0)
	c, _, _ = clock.Apply(c, clock.Command{ID: "2", Type: clock.Pause}, t0.Add(3*time.Second))
	if c.State != clock.Paused || clock.ElapsedAt(c, t0.Add(10*time.Second)) != 3*time.Second {
		t.Fatalf("pause did not freeze stopwatch: %#v", c)
	}
	c, _, _ = clock.Apply(c, clock.Command{ID: "3", Type: clock.Resume}, t0.Add(5*time.Second))
	c, _, _ = clock.Apply(c, clock.Command{ID: "4", Type: clock.Pause}, t0.Add(6*time.Second))
	if got := clock.ElapsedAt(c, t0.Add(20*time.Second)); got != 4*time.Second {
		t.Fatalf("elapsed=%s, want 4s", got)
	}
	c.Laps = []clock.LapRecord{{Number: 1, Elapsed: time.Second, Split: time.Second}}
	c, event, err := clock.Apply(c, clock.Command{ID: "5", Type: clock.Reset}, t0.Add(7*time.Second))
	if err != nil || event.Type != clock.ResetDone || c.State != clock.Idle || c.Accumulated != 0 || len(c.Laps) != 0 {
		t.Fatalf("unexpected reset: clock=%#v event=%#v err=%v", c, event, err)
	}
}

func TestTimerStopsAtZeroAndRejectsLap(t *testing.T) {
	t0 := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	c := clock.NewTimer("timer-1", 3*time.Second)
	c, _, err := clock.Apply(c, clock.Command{ID: "1", Type: clock.Start}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := clock.Apply(c, clock.Command{ID: "2", Type: clock.Lap}, t0.Add(time.Second)); err != clock.ErrLapUnsupported {
		t.Fatalf("lap error=%v, want ErrLapUnsupported", err)
	}
	if got := clock.RemainingAt(c, t0.Add(2500*time.Millisecond)); got != 500*time.Millisecond {
		t.Fatalf("remaining=%s, want 500ms", got)
	}
	c, event, err := clock.Apply(c, clock.Command{ID: "3", Type: clock.Expire}, t0.Add(3*time.Second))
	if err != nil || c.State != clock.Expired || event.Type != clock.ExpiredEvent || clock.RemainingAt(c, t0.Add(4*time.Second)) != 0 {
		t.Fatalf("unexpected expiry: clock=%#v event=%#v err=%v", c, event, err)
	}
	c, _, err = clock.Apply(c, clock.Command{ID: "4", Type: clock.Reset}, t0.Add(5*time.Second))
	if err != nil || c.State != clock.Idle || clock.RemainingAt(c, t0.Add(9*time.Second)) != 3*time.Second {
		t.Fatalf("timer reset failed: %#v err=%v", c, err)
	}
}

func TestInvalidAndUndoTransitions(t *testing.T) {
	t0 := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	c := clock.NewStopwatch("clock-1")
	if _, _, err := clock.Apply(c, clock.Command{ID: "pause", Type: clock.Pause}, t0); err != clock.ErrInvalidTransition {
		t.Fatalf("pause idle error=%v", err)
	}
	running, event, _ := clock.Apply(c, clock.Command{ID: "start", Type: clock.Start}, t0)
	undone, undoEvent, err := clock.Compensate(running, event, t0.Add(9*time.Second), "undo-1")
	if err != nil || undone.State != clock.Idle || undone.Version != 2 || undoEvent.Type != clock.Undone {
		t.Fatalf("undo failed: clock=%#v event=%#v err=%v", undone, undoEvent, err)
	}
	if _, _, err := clock.Compensate(running, event, t0.Add(11*time.Second), "undo-late"); err != clock.ErrUndoExpired {
		t.Fatalf("late undo error=%v", err)
	}
}

func TestDisplayPrecision(t *testing.T) {
	if got := clock.FormatStopwatch(12*time.Second + 347*time.Millisecond); got != "00:12.34" {
		t.Fatalf("live stopwatch=%q", got)
	}
	if got := clock.FormatStopwatchDetail(12*time.Second + 347*time.Millisecond); got != "00:12.347" {
		t.Fatalf("stopped stopwatch=%q", got)
	}
	if got := clock.FormatTimer(12*time.Second + time.Millisecond); got != "00:13" {
		t.Fatalf("timer=%q", got)
	}
}
