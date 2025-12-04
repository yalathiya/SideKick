package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yalathiya/sidekick/internal/logging"
	"github.com/yalathiya/sidekick/internal/metrics"
	"github.com/yalathiya/sidekick/internal/ratelimit"
)

const defaultUpstream = "http://httpbin.org"

func newReverseProxy(target string) *httputil.ReverseProxy {
	up, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid upstream %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(up)

	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		r.Header.Set("X-Forwarded-By", "sidekick")
	}
	return proxy
}

func main() {
	r := chi.NewRouter()

	// register metrics
	metrics.Register()

	// Core middlewares
	r.Use(middleware.RequestID) // X-Request-ID
	r.Use(middleware.RealIP)    // X-Forwarded-For -> RemoteAddr

	// Your custom logging middleware (logs after response)
	r.Use(logging.Middleware)

	// Rate limiter: 5 requests per 10s per client
	lim := ratelimit.NewLimiter(5, 10*time.Second)
	// Wrap as chi middleware
	r.Use(func(next http.Handler) http.Handler {
		return lim.Middleware(next)
	})

	// health & metrics
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# metrics placeholder\n"))
	})

	// proxy catch-all
	proxy := newReverseProxy(defaultUpstream)
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Sidekick", "v0.1")
		proxy.ServeHTTP(w, r)
	}))

	addr := "0.0.0.0:8081"
	log.Printf("sidekick (chi) listening on %s -> %s", addr, defaultUpstream)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
