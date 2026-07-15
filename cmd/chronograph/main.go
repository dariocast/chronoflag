package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chronograph/internal/httpapi"
	"chronograph/internal/realtime"
	"chronograph/internal/retention"
	"chronograph/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("chronograph stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var data store.Store
	var closeDB func()
	if url := os.Getenv("DATABASE_URL"); url != "" {
		pg, err := store.NewPostgres(ctx, url)
		if err != nil {
			return err
		}
		if err = pg.Migrate(ctx); err != nil {
			pg.Close()
			return err
		}
		data = pg
		closeDB = pg.Close
	} else {
		slog.Warn("DATABASE_URL not set; using volatile memory store")
		data = store.NewMemory()
		closeDB = func() {}
	}
	defer closeDB()
	hub := realtime.NewHub(64)
	worker := retention.New(data, time.Hour)
	go worker.Run(ctx)
	go runExpiry(ctx, data, hub)
	server := &http.Server{Addr: env("LISTEN_ADDR", ":8080"), Handler: httpapi.NewRouter(data, hub), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 75 * time.Second, MaxHeaderBytes: 1 << 20}
	errs := make(chan error, 1)
	go func() { slog.Info("chronograph listening", "address", server.Addr); errs <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func runExpiry(ctx context.Context, data store.Store, hub *realtime.Hub) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			changed, err := data.ExpireDue(ctx, now.UTC())
			if err != nil {
				slog.Error("timer expiry scan failed", "error", err)
				continue
			}
			for _, snapshot := range changed {
				payload, _ := json.Marshal(struct {
					store.InstanceSnapshot
					ServerTime time.Time `json:"server_time"`
				}{snapshot, now.UTC()})
				hub.Publish(snapshot.ID, realtime.Message{ID: snapshot.Version, Event: "update", Data: payload})
			}
		case <-ctx.Done():
			return
		}
	}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
