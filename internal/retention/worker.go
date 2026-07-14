package retention

import (
	"chronograph/internal/store"
	"context"
	"time"
)

type Worker struct {
	store    store.Store
	interval time.Duration
}

func New(s store.Store, interval time.Duration) *Worker {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Worker{store: s, interval: interval}
}
func (w *Worker) RunOnce(ctx context.Context, now time.Time) (int, int, error) {
	return w.store.ArchiveAndPurge(ctx, now)
}
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			_, _, _ = w.RunOnce(ctx, now.UTC())
		case <-ctx.Done():
			return
		}
	}
}
