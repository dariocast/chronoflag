package store

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"chronograph/internal/clock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	p, e := pgxpool.New(ctx, url)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return &Postgres{pool: p}, nil
}
func (p *Postgres) Close() { p.pool.Close() }
func (p *Postgres) Migrate(ctx context.Context) error {
	b, e := migrations.ReadFile("migrations/001_init.sql")
	if e != nil {
		return e
	}
	_, e = p.pool.Exec(ctx, string(b))
	return e
}
func pgerr(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return e
}
func (p *Postgres) CreateInstance(ctx context.Context, now time.Time) (CreatedInstance, error) {
	id, control := token(), token()
	view := id
	s := InstanceSnapshot{ID: id, Tier: Free, Lifecycle: Active, LastControlAt: now, Clocks: []clock.Clock{clock.NewStopwatch(token())}}
	b, _ := json.Marshal(s)
	tx, e := p.pool.Begin(ctx)
	if e != nil {
		return CreatedInstance{}, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "INSERT INTO instances(id,snapshot,last_control_at) VALUES($1,$2,$3)", id, b, now); e != nil {
		return CreatedInstance{}, e
	}
	for _, v := range []struct {
		raw   string
		scope Scope
	}{{control, Control}, {view, View}} {
		h := sha256.Sum256([]byte(v.raw))
		if _, e = tx.Exec(ctx, "INSERT INTO capabilities(token_hash,instance_id,scope) VALUES($1,$2,$3)", h[:], id, v.scope); e != nil {
			return CreatedInstance{}, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return CreatedInstance{}, e
	}
	return CreatedInstance{id, control, view}, nil
}
func (p *Postgres) ResolveCapability(ctx context.Context, raw string) (Capability, error) {
	h := sha256.Sum256([]byte(raw))
	var c Capability
	e := p.pool.QueryRow(ctx, "SELECT instance_id,scope FROM capabilities WHERE token_hash=$1", h[:]).Scan(&c.InstanceID, &c.Scope)
	return c, pgerr(e)
}
func scanSnapshot(row pgx.Row) (InstanceSnapshot, error) {
	var b []byte
	if e := row.Scan(&b); e != nil {
		return InstanceSnapshot{}, pgerr(e)
	}
	var s InstanceSnapshot
	e := json.Unmarshal(b, &s)
	return s, e
}
func (p *Postgres) Snapshot(ctx context.Context, id string) (InstanceSnapshot, error) {
	return scanSnapshot(p.pool.QueryRow(ctx, "SELECT snapshot FROM instances WHERE id=$1", id))
}
func saveSnapshot(ctx context.Context, tx pgx.Tx, s InstanceSnapshot) error {
	b, _ := json.Marshal(s)
	_, e := tx.Exec(ctx, "UPDATE instances SET snapshot=$2,tier=$3,lifecycle=$4,last_control_at=$5,archived_at=$6 WHERE id=$1", s.ID, b, s.Tier, s.Lifecycle, s.LastControlAt, s.ArchivedAt)
	return e
}
func locked(ctx context.Context, pool *pgxpool.Pool, id string, fn func(*InstanceSnapshot) error) (InstanceSnapshot, error) {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return InstanceSnapshot{}, e
	}
	defer tx.Rollback(ctx)
	s, e := scanSnapshot(tx.QueryRow(ctx, "SELECT snapshot FROM instances WHERE id=$1 FOR UPDATE", id))
	if e != nil {
		return InstanceSnapshot{}, e
	}
	if e = fn(&s); e != nil {
		return s, e
	}
	if e = saveSnapshot(ctx, tx, s); e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	return s, nil
}
func (p *Postgres) ApplyCommand(ctx context.Context, c Capability, id string, cmd clock.Command, now time.Time) (InstanceSnapshot, clock.Event, error) {
	if e := requireControl(c); e != nil {
		return InstanceSnapshot{}, clock.Event{}, e
	}
	tx, e := p.pool.Begin(ctx)
	if e != nil {
		return InstanceSnapshot{}, clock.Event{}, e
	}
	defer tx.Rollback(ctx)
	s, e := scanSnapshot(tx.QueryRow(ctx, "SELECT snapshot FROM instances WHERE id=$1 FOR UPDATE", c.InstanceID))
	if e != nil {
		return s, clock.Event{}, e
	}
	var prior []byte
	if e = tx.QueryRow(ctx, "SELECT event FROM clock_events WHERE instance_id=$1 AND command_id=$2", c.InstanceID, cmd.ID).Scan(&prior); e == nil {
		var ev clock.Event
		_ = json.Unmarshal(prior, &ev)
		return s, ev, nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return s, clock.Event{}, e
	}
	if s.Lifecycle != Active {
		return s, clock.Event{}, ErrArchived
	}
	i := findClock(&s, id)
	if i < 0 {
		return s, clock.Event{}, ErrNotFound
	}
	next, ev, e := clock.Apply(s.Clocks[i], cmd, now)
	if e != nil {
		return s, clock.Event{}, e
	}
	s.Clocks[i] = next
	s.Version++
	s.LastControlAt = now
	ev.Sequence = s.Version
	eb, _ := json.Marshal(ev)
	if _, e = tx.Exec(ctx, "INSERT INTO clock_events(instance_id,sequence,clock_id,command_id,event) VALUES($1,$2,$3,$4,$5)", s.ID, ev.Sequence, id, cmd.ID, eb); e != nil {
		return s, clock.Event{}, e
	}
	if e = saveSnapshot(ctx, tx, s); e != nil {
		return s, clock.Event{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, clock.Event{}, e
	}
	return s, ev, nil
}
func (p *Postgres) Undo(ctx context.Context, c Capability, id, commandID string, now time.Time) (InstanceSnapshot, clock.Event, error) {
	if e := requireControl(c); e != nil {
		return InstanceSnapshot{}, clock.Event{}, e
	}
	var out clock.Event
	s, e := locked(ctx, p.pool, c.InstanceID, func(s *InstanceSnapshot) error {
		var b []byte
		if e := p.pool.QueryRow(ctx, "SELECT event FROM clock_events WHERE instance_id=$1 AND clock_id=$2 ORDER BY sequence DESC LIMIT 1", s.ID, id).Scan(&b); e != nil {
			return pgerr(e)
		}
		var latest clock.Event
		if e := json.Unmarshal(b, &latest); e != nil {
			return e
		}
		i := findClock(s, id)
		if i < 0 {
			return ErrNotFound
		}
		latest.Sequence = s.Clocks[i].Version
		next, ev, e := clock.Compensate(s.Clocks[i], latest, now, commandID)
		if e != nil {
			return e
		}
		s.Clocks[i] = next
		s.Version++
		s.LastControlAt = now
		ev.Sequence = s.Version
		out = ev
		eb, _ := json.Marshal(ev)
		_, e = p.pool.Exec(ctx, "INSERT INTO clock_events(instance_id,sequence,clock_id,command_id,event) VALUES($1,$2,$3,$4,$5)", s.ID, ev.Sequence, id, commandID, eb)
		return e
	})
	return s, out, e
}
func (p *Postgres) AddClock(ctx context.Context, c Capability, t clock.ClockType, d time.Duration, now time.Time) (InstanceSnapshot, error) {
	if e := requireControl(c); e != nil {
		return InstanceSnapshot{}, e
	}
	return locked(ctx, p.pool, c.InstanceID, func(s *InstanceSnapshot) error {
		if s.Lifecycle != Active {
			return ErrArchived
		}
		if len(s.Clocks) >= 100 {
			return ErrLimit
		}
		id := token()
		v := clock.NewStopwatch(id)
		if t == clock.Timer {
			if d <= 0 {
				return fmt.Errorf("duration must be positive")
			}
			v = clock.NewTimer(id, d)
		}
		v.Order = len(s.Clocks)
		s.Clocks = append(s.Clocks, v)
		s.Version++
		s.LastControlAt = now
		return nil
	})
}
func (p *Postgres) UpdateClock(ctx context.Context, c Capability, id string, patch ClockPatch, now time.Time) (InstanceSnapshot, error) {
	if e := requireControl(c); e != nil {
		return InstanceSnapshot{}, e
	}
	return locked(ctx, p.pool, c.InstanceID, func(s *InstanceSnapshot) error {
		i := findClock(s, id)
		if i < 0 {
			return ErrNotFound
		}
		if patch.Label != nil {
			if len(*patch.Label) > 80 {
				return ErrLimit
			}
			s.Clocks[i].Label = *patch.Label
		}
		if patch.Order != nil {
			s.Clocks[i].Order = *patch.Order
		}
		if patch.Highlighted != nil {
			if *patch.Highlighted {
				s.HighlightedClockID = id
			} else if s.HighlightedClockID == id {
				s.HighlightedClockID = ""
			}
		}
		s.Version++
		s.LastControlAt = now
		return nil
	})
}
func (p *Postgres) RemoveClock(ctx context.Context, c Capability, id string, now time.Time) (InstanceSnapshot, error) {
	if e := requireControl(c); e != nil {
		return InstanceSnapshot{}, e
	}
	return locked(ctx, p.pool, c.InstanceID, func(s *InstanceSnapshot) error {
		if len(s.Clocks) <= 1 {
			return ErrLimit
		}
		i := findClock(s, id)
		if i < 0 {
			return ErrNotFound
		}
		s.Clocks = append(s.Clocks[:i], s.Clocks[i+1:]...)
		if s.HighlightedClockID == id {
			s.HighlightedClockID = ""
		}
		s.Version++
		s.LastControlAt = now
		return nil
	})
}
func (p *Postgres) EventsAfter(ctx context.Context, id string, seq int64) ([]clock.Event, error) {
	rows, e := p.pool.Query(ctx, "SELECT event FROM clock_events WHERE instance_id=$1 AND sequence>$2 ORDER BY sequence", id, seq)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []clock.Event{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		var ev clock.Event
		if e = json.Unmarshal(b, &ev); e != nil {
			return nil, e
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}
func (p *Postgres) Export(ctx context.Context, c Capability) (ExportData, error) {
	if e := requireControl(c); e != nil {
		return ExportData{}, e
	}
	s, e := p.Snapshot(ctx, c.InstanceID)
	if e != nil {
		return ExportData{}, e
	}
	events, e := p.EventsAfter(ctx, c.InstanceID, 0)
	return ExportData{SchemaVersion: 1, Instance: s, Events: events}, e
}
func (p *Postgres) DeleteInstance(ctx context.Context, c Capability) error {
	if e := requireControl(c); e != nil {
		return e
	}
	tag, e := p.pool.Exec(ctx, "DELETE FROM instances WHERE id=$1", c.InstanceID)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (p *Postgres) ArchiveAndPurge(ctx context.Context, now time.Time) (int, int, error) {
	tag, e := p.pool.Exec(ctx, `UPDATE instances SET lifecycle='archived', archived_at=$1, purge_at=CASE WHEN tier='premium' THEN $1 + interval '1 year' ELSE $1 + interval '7 days' END, snapshot=jsonb_set(jsonb_set(snapshot,'{lifecycle}','"archived"'),'{archived_at}',to_jsonb($1::timestamptz)) WHERE lifecycle='active' AND last_control_at <= $1 - CASE WHEN tier='premium' THEN interval '30 days' ELSE interval '24 hours' END`, now)
	if e != nil {
		return 0, 0, e
	}
	del, e := p.pool.Exec(ctx, "DELETE FROM instances WHERE lifecycle='archived' AND purge_at <= $1", now)
	if e != nil {
		return 0, 0, e
	}
	return int(tag.RowsAffected()), int(del.RowsAffected()), nil
}
func (p *Postgres) RawCapabilityCount(ctx context.Context, raw string) (int, error) {
	var n int
	e := p.pool.QueryRow(ctx, "SELECT count(*) FROM capabilities WHERE encode(token_hash,'escape')=$1", raw).Scan(&n)
	return n, e
}
