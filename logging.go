package main

import (
	"log"
	"net/http"
	"time"
)

type LogEntry struct {
	Timestamp time.Time     `json:"timestamp"`
	Method    string        `json:"method"`
	Path      string        `json:"path"`
	Status    int           `json:"status"`
	Duration  time.Duration `json:"duration"`
	UserAgent string        `json:"user_agent"`
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		entry := LogEntry{
			Timestamp: start,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    wrapped.statusCode,
			Duration:  duration,
			UserAgent: r.UserAgent(),
		}

		log.Printf("%s %s %d %v", entry.Method, entry.Path, entry.Status, entry.Duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
