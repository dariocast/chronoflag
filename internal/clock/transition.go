package clock

import "time"

const UndoWindow = 10 * time.Second

func Apply(current Clock, command Command, now time.Time) (Clock, Event, error) {
	if current.Type == Timer && command.Type == Lap {
		return current, Event{}, ErrLapUnsupported
	}

	before := snapshot(current)
	next := current
	var eventType EventType

	switch command.Type {
	case Start:
		if current.State != Idle {
			return current, Event{}, ErrInvalidTransition
		}
		next.State = Running
		next.Anchor = now
		eventType = Started
	case Pause:
		if current.State != Running {
			return current, Event{}, ErrInvalidTransition
		}
		next.Accumulated = ElapsedAt(current, now)
		next.Anchor = time.Time{}
		next.State = Paused
		eventType = PausedEvent
	case Resume:
		if current.State != Paused {
			return current, Event{}, ErrInvalidTransition
		}
		next.Anchor = now
		next.State = Running
		eventType = Resumed
	case Lap:
		if current.State != Running {
			return current, Event{}, ErrInvalidTransition
		}
		elapsed := ElapsedAt(current, now)
		previous := time.Duration(0)
		if len(current.Laps) > 0 {
			previous = current.Laps[len(current.Laps)-1].Elapsed
		}
		next.Laps = append(append([]LapRecord(nil), current.Laps...), LapRecord{
			Number:  len(current.Laps) + 1,
			Elapsed: elapsed,
			Split:   elapsed - previous,
		})
		eventType = Lapped
	case Reset:
		next.State = Idle
		next.Accumulated = 0
		next.Anchor = time.Time{}
		next.Laps = nil
		eventType = ResetDone
	case Expire:
		if current.Type != Timer || current.State != Running || RemainingAt(current, now) > 0 {
			return current, Event{}, ErrInvalidTransition
		}
		next.State = Expired
		next.Accumulated = current.Duration
		next.Anchor = time.Time{}
		eventType = ExpiredEvent
	default:
		return current, Event{}, ErrInvalidTransition
	}

	next.Version++
	event := Event{
		Sequence: next.Version, ClockID: current.ID, CommandID: command.ID,
		DeviceID: command.DeviceID, Type: eventType, OccurredAt: now, Before: before,
	}
	return next, event, nil
}

func Compensate(current Clock, latest Event, now time.Time, commandID string) (Clock, Event, error) {
	if current.Version != latest.Sequence {
		return current, Event{}, ErrUndoSuperseded
	}
	if now.Before(latest.OccurredAt) || now.Sub(latest.OccurredAt) > UndoWindow {
		return current, Event{}, ErrUndoExpired
	}
	beforeUndo := snapshot(current)
	next := current
	next.State = latest.Before.State
	next.Accumulated = latest.Before.Accumulated
	next.Anchor = latest.Before.Anchor
	next.Laps = append([]LapRecord(nil), latest.Before.Laps...)
	next.Version++
	return next, Event{
		Sequence: next.Version, ClockID: current.ID, CommandID: commandID,
		Type: Undone, OccurredAt: now, Before: beforeUndo,
	}, nil
}
