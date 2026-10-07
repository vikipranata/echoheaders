package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var stdoutMu sync.Mutex

func logJSON(entry map[string]any) {
	entry["time"] = time.Now().UTC().Format(time.RFC3339Nano)
	b, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logJSON: %v\n", err)
		return
	}
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	_, _ = os.Stdout.Write(append(b, '\n'))
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xri != "" {
		return xri
	}
	// Strip the ephemeral port so the address can be used as a per-client key.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		entry := map[string]any{
			"level":       "info",
			"msg":         "request",
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"status":      rec.status,
			"bytes":       rec.bytes,
			"duration_ms": float64(time.Since(start).Nanoseconds()) / 1e6,
			"remoteAddress":  r.RemoteAddr,
			"realIp":      realIP(r),
		}
		if ua := r.Header.Get("User-Agent"); ua != "" {
			entry["user_agent"] = ua
		}
		logJSON(entry)
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/traceroute", tracerouteHandler)
	mux.HandleFunc("/dns", dnsHandler)
	mux.HandleFunc("/", headersHandler)
	return mux
}

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           loggingMiddleware(newRouter()),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logJSON(map[string]any{"level": "info", "msg": "listening", "port": port})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logJSON(map[string]any{"level": "error", "msg": "server error", "error": err.Error()})
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logJSON(map[string]any{"level": "error", "msg": "graceful shutdown failed", "error": err.Error()})
			os.Exit(1)
		}
		logJSON(map[string]any{"level": "info", "msg": "stopped"})
	}
}
