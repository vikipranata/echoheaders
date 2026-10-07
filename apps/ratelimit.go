package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type minuteLimiter struct {
	mu     sync.Mutex
	limit  int
	start  time.Time
	counts map[string]int
}

func newMinuteLimiter(limit int) *minuteLimiter {
	if limit <= 0 {
		limit = 10
	}
	return &minuteLimiter{limit: limit, start: time.Now(), counts: map[string]int{}}
}

func (l *minuteLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.start) >= time.Minute {
		l.start = now
		l.counts = map[string]int{}
	}
	if l.counts[key] >= l.limit {
		return false, time.Until(l.start.Add(time.Minute)).Round(time.Second)
	}
	l.counts[key]++
	return true, 0
}

func limitFromEnv(name string, def int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

var (
	dnsLimiter        = newMinuteLimiter(limitFromEnv("RATE_LIMIT_DNS", 10))
	pingLimiter       = newMinuteLimiter(limitFromEnv("RATE_LIMIT_PING", 10))
	tracerouteLimiter = newMinuteLimiter(limitFromEnv("RATE_LIMIT_TRACEROUTE", 10))
)

func checkRateLimit(l *minuteLimiter, w http.ResponseWriter, r *http.Request, renderHTML429 func(retryAfter time.Duration)) bool {
	ok, ra := l.allow(realIP(r))
	if ok {
		return true
	}
	secs := int(ra.Seconds())
	if secs < 1 {
		secs = 1
	}
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          false,
			"error":       "rate limit exceeded",
			"retry_after": secs,
		})
		return false
	}
	renderHTML429(time.Duration(secs) * time.Second)
	return false
}
