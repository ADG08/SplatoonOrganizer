package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// serveHealth answers the container healthcheck on /health until ctx is done.
func serveHealth(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, // a client sending headers slowly can't hold a connection
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Debug("health server listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// checkHealth asks the running bot's /health, as the container's healthcheck: it returns
// the exit code, 0 when the bot answers.
func checkHealth() int {
	port := os.Getenv("HEALTH_PORT")
	if port == "" {
		port = defaultHealthPort
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
