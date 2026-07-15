package store

import (
	"context"
	"errors"
	"time"

	"chronograph/internal/clock"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrArchived  = errors.New("instance is archived")
	ErrLimit     = errors.New("instance limit exceeded")
)

type Scope string

const (
	Control Scope = "control"
	View    Scope = "view"
)

type Tier string

const (
	Free    Tier = "free"
	Premium Tier = "premium"
)

type Lifecycle string

const (
	Active   Lifecycle = "active"
	Archived Lifecycle = "archived"
)

type Capability struct {
	InstanceID string `json:"instance_id"`
	Scope      Scope  `json:"scope"`
}
type CreatedInstance struct{ InstanceID, ControlToken, ViewToken string }
type InstanceSnapshot struct {
	ID                 string        `json:"id"`
	Tier               Tier          `json:"tier"`
	Lifecycle          Lifecycle     `json:"lifecycle"`
	Clocks             []clock.Clock `json:"clocks"`
	HighlightedClockID string        `json:"highlighted_clock_id,omitempty"`
	Version            int64         `json:"version"`
	LastControlAt      time.Time     `json:"last_control_at"`
	ArchivedAt         *time.Time    `json:"archived_at,omitempty"`
}
type ClockPatch struct {
	Label       *string `json:"label,omitempty"`
	Order       *int    `json:"order,omitempty"`
	Highlighted *bool   `json:"highlighted,omitempty"`
}
type ExportData struct {
	SchemaVersion int              `json:"schema_version"`
	Instance      InstanceSnapshot `json:"instance"`
	Events        []clock.Event    `json:"events"`
}

type Store interface {
	CreateInstance(context.Context, time.Time) (CreatedInstance, error)
	ResolveCapability(context.Context, string) (Capability, error)
	Snapshot(context.Context, string) (InstanceSnapshot, error)
	ApplyCommand(context.Context, Capability, string, clock.Command, time.Time) (InstanceSnapshot, clock.Event, error)
	Undo(context.Context, Capability, string, string, time.Time) (InstanceSnapshot, clock.Event, error)
	AddClock(context.Context, Capability, clock.ClockType, time.Duration, time.Time) (InstanceSnapshot, error)
	UpdateClock(context.Context, Capability, string, ClockPatch, time.Time) (InstanceSnapshot, error)
	RemoveClock(context.Context, Capability, string, time.Time) (InstanceSnapshot, error)
	EventsAfter(context.Context, string, int64) ([]clock.Event, error)
	Export(context.Context, Capability) (ExportData, error)
	DeleteInstance(context.Context, Capability) error
	ArchiveAndPurge(context.Context, time.Time) (int, int, error)
	ExpireDue(context.Context, time.Time) ([]InstanceSnapshot, error)
}

func reorderClock(s *InstanceSnapshot, id string, target int) bool {
	from := findClock(s, id)
	if from < 0 {
		return false
	}
	if target < 0 {
		target = 0
	}
	if target >= len(s.Clocks) {
		target = len(s.Clocks) - 1
	}
	item := s.Clocks[from]
	s.Clocks = append(s.Clocks[:from], s.Clocks[from+1:]...)
	s.Clocks = append(s.Clocks, clock.Clock{})
	copy(s.Clocks[target+1:], s.Clocks[target:])
	s.Clocks[target] = item
	for i := range s.Clocks {
		s.Clocks[i].Order = i
	}
	return true
}
