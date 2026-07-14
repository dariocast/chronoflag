package clock

import (
	"errors"
	"fmt"
	"time"
)

type ClockType string

const (
	Stopwatch ClockType = "stopwatch"
	Timer     ClockType = "timer"
)

type State string

const (
	Idle    State = "idle"
	Running State = "running"
	Paused  State = "paused"
	Expired State = "expired"
)

type CommandType string

const (
	Start  CommandType = "start"
	Pause  CommandType = "pause"
	Resume CommandType = "resume"
	Lap    CommandType = "lap"
	Reset  CommandType = "reset"
	Expire CommandType = "expire"
)

type EventType string

const (
	Started      EventType = "started"
	PausedEvent  EventType = "paused"
	Resumed      EventType = "resumed"
	Lapped       EventType = "lapped"
	ResetDone    EventType = "reset"
	ExpiredEvent EventType = "expired"
	Undone       EventType = "undone"
)

var (
	ErrInvalidTransition = errors.New("invalid clock transition")
	ErrLapUnsupported    = errors.New("lap is only supported by stopwatches")
	ErrUndoExpired       = errors.New("undo window expired")
	ErrUndoSuperseded    = errors.New("operation is no longer the latest")
)

type LapRecord struct {
	Number  int           `json:"number"`
	Elapsed time.Duration `json:"elapsed"`
	Split   time.Duration `json:"split"`
	Label   string        `json:"label,omitempty"`
}

type Clock struct {
	ID          string        `json:"id"`
	Type        ClockType     `json:"type"`
	State       State         `json:"state"`
	Label       string        `json:"label,omitempty"`
	Order       int           `json:"order"`
	Duration    time.Duration `json:"duration,omitempty"`
	Accumulated time.Duration `json:"accumulated"`
	Anchor      time.Time     `json:"anchor,omitempty"`
	Version     int64         `json:"version"`
	Laps        []LapRecord   `json:"laps,omitempty"`
}

type Command struct {
	ID       string      `json:"id"`
	Type     CommandType `json:"type"`
	DeviceID string      `json:"device_id,omitempty"`
}

type Snapshot struct {
	State       State         `json:"state"`
	Accumulated time.Duration `json:"accumulated"`
	Anchor      time.Time     `json:"anchor,omitempty"`
	Laps        []LapRecord   `json:"laps,omitempty"`
}

type Event struct {
	Sequence   int64     `json:"sequence"`
	ClockID    string    `json:"clock_id"`
	CommandID  string    `json:"command_id"`
	DeviceID   string    `json:"device_id,omitempty"`
	Type       EventType `json:"type"`
	OccurredAt time.Time `json:"occurred_at"`
	Before     Snapshot  `json:"before"`
}

func NewStopwatch(id string) Clock {
	return Clock{ID: id, Type: Stopwatch, State: Idle}
}

func NewTimer(id string, duration time.Duration) Clock {
	if duration < 0 {
		duration = 0
	}
	return Clock{ID: id, Type: Timer, State: Idle, Duration: duration}
}

func ElapsedAt(c Clock, now time.Time) time.Duration {
	elapsed := c.Accumulated
	if c.State == Running && !c.Anchor.IsZero() && now.After(c.Anchor) {
		elapsed += now.Sub(c.Anchor)
	}
	if elapsed < 0 {
		return 0
	}
	if c.Type == Timer && elapsed > c.Duration {
		return c.Duration
	}
	return elapsed
}

func RemainingAt(c Clock, now time.Time) time.Duration {
	if c.Type != Timer {
		return 0
	}
	remaining := c.Duration - ElapsedAt(c, now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func FormatStopwatch(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	centiseconds := d / (10 * time.Millisecond)
	minutes := centiseconds / 6000
	seconds := (centiseconds / 100) % 60
	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, centiseconds%100)
}

func FormatStopwatchDetail(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	milliseconds := d / time.Millisecond
	minutes := milliseconds / 60000
	seconds := (milliseconds / 1000) % 60
	return fmt.Sprintf("%02d:%02d.%03d", minutes, seconds, milliseconds%1000)
}

func FormatTimer(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secondsTotal := int64(0)
	if d > 0 {
		secondsTotal = int64((d + time.Second - 1) / time.Second)
	}
	return fmt.Sprintf("%02d:%02d", secondsTotal/60, secondsTotal%60)
}

func snapshot(c Clock) Snapshot {
	laps := append([]LapRecord(nil), c.Laps...)
	return Snapshot{State: c.State, Accumulated: c.Accumulated, Anchor: c.Anchor, Laps: laps}
}
