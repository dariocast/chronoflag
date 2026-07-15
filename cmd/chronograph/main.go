package main

import (
	"context"
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
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
