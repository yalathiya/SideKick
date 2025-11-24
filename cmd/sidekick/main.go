package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	// temporary upstream for smoke tests
	target := "http://httpbin.org"
	upstream, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid upstream %q: %v", target, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstream)

	// simple handler that injects an identifying header and forwards to upstream
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Sidekick", "v0.1")
		proxy.ServeHTTP(w, r)
	})

	addr := ":8081"
	log.Printf("sidekick proxy listening on %s -> %s", addr, target)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
