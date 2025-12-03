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
)

// defaultUpstream is a temporary upstream used for the smoke test.
// Later we'll make this configurable via CLI / config.
const defaultUpstream = "http://httpbin.org"

func newReverseProxy(target string) *httputil.ReverseProxy {
	up, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid upstream %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(up)

	// Optional: tweak director to preserve Host or add headers
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		// Example: add header so upstream knows request passed via sidekick
		r.Header.Set("X-Forwarded-By", "sidekick")
	}

	// Optional: set timeouts via Transport if needed (left default here)
	return proxy
}

func main() {
	// create chi router
	r := chi.NewRouter()

	// useful middlewares from chi
	r.Use(middleware.RequestID)                 // creates X-Request-ID
	r.Use(middleware.RealIP)                    // respect X-Forwarded-For
	r.Use(logging.Middleware)                   // basic logging to stdout (for dev)
	r.Use(middleware.Recoverer)                 // recover from panics and return 500
	r.Use(middleware.Timeout(30 * time.Second)) // timeout per request
	
	// health endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// metrics endpoint (placeholder). We'll add Prometheus handler later.
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# metrics will appear here when enabled\n"))
	})

	// create proxy and mount as catch-all
	proxy := newReverseProxy(defaultUpstream)

	// We mount proxy as the last route. chi's `Handle` accepts patterns,
	// so "/*" will forward everything not matched above to upstream.
	// If you later add admin endpoints, they should come before this.
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// example response header we add at the edge
		w.Header().Set("X-Sidekick", "v0.1")
		proxy.ServeHTTP(w, r)
	}))

	addr := ":8081"
	log.Printf("sidekick (chi) listening on %s -> %s", addr, defaultUpstream)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
