package logging

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// statusRecorder allows us to capture the response status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rec := &statusRecorder{
			ResponseWriter: w,
			status:         200,
		}

		next.ServeHTTP(rec, r)

		duration := time.Since(start)

		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		// Use chi's helper to get request id
		reqID := middleware.GetReqID(r.Context())
		if reqID == "" {
			reqID = "-"
		}

		log.Printf(
			"[%s] %d %s %s  %v  client=%s",
			reqID,
			rec.status,
			r.Method,
			r.URL.Path,
			duration,
			ip,
		)
	})
}
