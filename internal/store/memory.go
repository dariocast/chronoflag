package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"sync"
	"time"

	"chronograph/internal/clock"
)

type memoryRecord struct {
	snapshot InstanceSnapshot
	events   []clock.Event
	commands map[string]clock.Event
	purgeAt  *time.Time
}
type Memory struct {
	mu           sync.Mutex
	instances    map[string]*memoryRecord
	capabilities map[[32]byte]Capability
}

func NewMemory() *Memory {
	return &Memory{instances: map[string]*memoryRecord{}, capabilities: map[[32]byte]Capability{}}
}
func token() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func hash(s string) [32]byte { return sha256.Sum256([]byte(s)) }

func (m *Memory) CreateInstance(_ context.Context, now time.Time) (CreatedInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, control := token(), token()
	view := id
	clockID := token()
	s := InstanceSnapshot{ID: id, Tier: Free, Lifecycle: Active, LastControlAt: now, Clocks: []clock.Clock{clock.NewStopwatch(clockID)}}
	m.instances[id] = &memoryRecord{snapshot: s, commands: map[string]clock.Event{}}
	m.capabilities[hash(control)] = Capability{InstanceID: id, Scope: Control}
	m.capabilities[hash(view)] = Capability{InstanceID: id, Scope: View}
	return CreatedInstance{InstanceID: id, ControlToken: control, ViewToken: view}, nil
}
func (m *Memory) ResolveCapability(_ context.Context, raw string) (Capability, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.capabilities[hash(raw)]
	if !ok {
		return Capability{}, ErrNotFound
	}
	return c, nil
}
func clone(s InstanceSnapshot) InstanceSnapshot {
	s.Clocks = append([]clock.Clock(nil), s.Clocks...)
	for i := range s.Clocks {
		s.Clocks[i].Laps = append([]clock.LapRecord(nil), s.Clocks[i].Laps...)
	}
	return s
}
func (m *Memory) Snapshot(_ context.Context, id string) (InstanceSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.instances[id]
	if !ok {
		return InstanceSnapshot{}, ErrNotFound
	}
	return clone(r.snapshot), nil
}
func requireControl(c Capability) error {
	if c.Scope != Control {
		return ErrForbidden
	}
	return nil
}
func findClock(s *InstanceSnapshot, id string) int {
	for i := range s.Clocks {
		if s.Clocks[i].ID == id {
			return i
		}
	}
	return -1
}

func (m *Memory) ApplyCommand(_ context.Context, c Capability, id string, cmd clock.Command, now time.Time) (InstanceSnapshot, clock.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return InstanceSnapshot{}, clock.Event{}, err
	}
	r, ok := m.instances[c.InstanceID]
	if !ok {
		return InstanceSnapshot{}, clock.Event{}, ErrNotFound
	}
	if r.snapshot.Lifecycle != Active {
		return clone(r.snapshot), clock.Event{}, ErrArchived
	}
	if e, ok := r.commands[cmd.ID]; ok {
		return clone(r.snapshot), e, nil
	}
	i := findClock(&r.snapshot, id)
	if i < 0 {
		return InstanceSnapshot{}, clock.Event{}, ErrNotFound
	}
	next, e, err := clock.Apply(r.snapshot.Clocks[i], cmd, now)
	if err != nil {
		return clone(r.snapshot), clock.Event{}, err
	}
	r.snapshot.Clocks[i] = next
	r.snapshot.Version++
	e.Sequence = r.snapshot.Version
	r.snapshot.LastControlAt = now
	r.events = append(r.events, e)
	r.commands[cmd.ID] = e
	return clone(r.snapshot), e, nil
}
func (m *Memory) Undo(_ context.Context, c Capability, id, commandID string, now time.Time) (InstanceSnapshot, clock.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return InstanceSnapshot{}, clock.Event{}, err
	}
	r := m.instances[c.InstanceID]
	if r == nil {
		return InstanceSnapshot{}, clock.Event{}, ErrNotFound
	}
	i := findClock(&r.snapshot, id)
	if i < 0 || len(r.events) == 0 {
		return InstanceSnapshot{}, clock.Event{}, ErrNotFound
	}
	var latest clock.Event
	for j := len(r.events) - 1; j >= 0; j-- {
		if r.events[j].ClockID == id {
			latest = r.events[j]
			break
		}
	}
	latest.Sequence = r.snapshot.Clocks[i].Version
	next, e, err := clock.Compensate(r.snapshot.Clocks[i], latest, now, commandID)
	if err != nil {
		return clone(r.snapshot), clock.Event{}, err
	}
	r.snapshot.Clocks[i] = next
	r.snapshot.Version++
	e.Sequence = r.snapshot.Version
	r.events = append(r.events, e)
	r.snapshot.LastControlAt = now
	return clone(r.snapshot), e, nil
}
func (m *Memory) AddClock(_ context.Context, c Capability, t clock.ClockType, d time.Duration, now time.Time) (InstanceSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return InstanceSnapshot{}, err
	}
	r := m.instances[c.InstanceID]
	if r == nil {
		return InstanceSnapshot{}, ErrNotFound
	}
	if r.snapshot.Lifecycle != Active {
		return clone(r.snapshot), ErrArchived
	}
	if len(r.snapshot.Clocks) >= 100 {
		return clone(r.snapshot), ErrLimit
	}
	id := token()
	v := clock.NewStopwatch(id)
	if t == clock.Timer {
		if d <= 0 {
			return clone(r.snapshot), fmt.Errorf("duration must be positive")
		}
		v = clock.NewTimer(id, d)
	}
	v.Order = len(r.snapshot.Clocks)
	r.snapshot.Clocks = append(r.snapshot.Clocks, v)
	r.snapshot.Version++
	r.snapshot.LastControlAt = now
	return clone(r.snapshot), nil
}
func (m *Memory) UpdateClock(_ context.Context, c Capability, id string, p ClockPatch, now time.Time) (InstanceSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return InstanceSnapshot{}, err
	}
	r := m.instances[c.InstanceID]
	if r == nil {
		return InstanceSnapshot{}, ErrNotFound
	}
	i := findClock(&r.snapshot, id)
	if i < 0 {
		return InstanceSnapshot{}, ErrNotFound
	}
	if p.Label != nil {
		if len(*p.Label) > 80 {
			return clone(r.snapshot), ErrLimit
		}
		r.snapshot.Clocks[i].Label = *p.Label
	}
	if p.Order != nil {
		r.snapshot.Clocks[i].Order = *p.Order
		sort.SliceStable(r.snapshot.Clocks, func(i, j int) bool { return r.snapshot.Clocks[i].Order < r.snapshot.Clocks[j].Order })
	}
	if p.Highlighted != nil {
		if *p.Highlighted {
			r.snapshot.HighlightedClockID = id
		} else if r.snapshot.HighlightedClockID == id {
			r.snapshot.HighlightedClockID = ""
		}
	}
	r.snapshot.Version++
	r.snapshot.LastControlAt = now
	return clone(r.snapshot), nil
}
func (m *Memory) RemoveClock(_ context.Context, c Capability, id string, now time.Time) (InstanceSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return InstanceSnapshot{}, err
	}
	r := m.instances[c.InstanceID]
	if r == nil {
		return InstanceSnapshot{}, ErrNotFound
	}
	if len(r.snapshot.Clocks) <= 1 {
		return clone(r.snapshot), ErrLimit
	}
	i := findClock(&r.snapshot, id)
	if i < 0 {
		return InstanceSnapshot{}, ErrNotFound
	}
	r.snapshot.Clocks = append(r.snapshot.Clocks[:i], r.snapshot.Clocks[i+1:]...)
	if r.snapshot.HighlightedClockID == id {
		r.snapshot.HighlightedClockID = ""
	}
	r.snapshot.Version++
	r.snapshot.LastControlAt = now
	return clone(r.snapshot), nil
}
func (m *Memory) EventsAfter(_ context.Context, id string, seq int64) ([]clock.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.instances[id]
	if r == nil {
		return nil, ErrNotFound
	}
	out := []clock.Event{}
	for _, e := range r.events {
		if e.Sequence > seq {
			out = append(out, e)
		}
	}
	return out, nil
}
func (m *Memory) Export(_ context.Context, c Capability) (ExportData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return ExportData{}, err
	}
	r := m.instances[c.InstanceID]
	if r == nil {
		return ExportData{}, ErrNotFound
	}
	return ExportData{SchemaVersion: 1, Instance: clone(r.snapshot), Events: append([]clock.Event(nil), r.events...)}, nil
}
func (m *Memory) DeleteInstance(_ context.Context, c Capability) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := requireControl(c); err != nil {
		return err
	}
	if _, ok := m.instances[c.InstanceID]; !ok {
		return ErrNotFound
	}
	delete(m.instances, c.InstanceID)
	for h, v := range m.capabilities {
		if v.InstanceID == c.InstanceID {
			delete(m.capabilities, h)
		}
	}
	return nil
}
func (m *Memory) ArchiveAndPurge(_ context.Context, now time.Time) (int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, p := 0, 0
	for id, r := range m.instances {
		activeFor := 24 * time.Hour
		retain := 7 * 24 * time.Hour
		if r.snapshot.Tier == Premium {
			activeFor = 30 * 24 * time.Hour
			retain = 365 * 24 * time.Hour
		}
		if r.snapshot.Lifecycle == Active && !now.Before(r.snapshot.LastControlAt.Add(activeFor)) {
			r.snapshot.Lifecycle = Archived
			t := now
			r.snapshot.ArchivedAt = &t
			pt := now.Add(retain)
			r.purgeAt = &pt
			a++
		} else if r.purgeAt != nil && !now.Before(*r.purgeAt) {
			delete(m.instances, id)
			p++
		}
	}
	return a, p, nil
}
