// hello-api: the smallest HTTP service that still behaves well on Kubernetes.
//
//	GET /          -> "hello from <pod> (version <v>)"
//	GET /healthz   -> liveness: the process is alive
//	GET /readyz    -> readiness: 503 once SIGTERM arrives (drain from LB first)
//	GET /burn?ms=N -> spin one CPU for N ms (drives the HPA lab)
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

var version = "dev" // set at build time: -ldflags "-X main.version=<sha>"

func main() {
	host, _ := os.Hostname()
	var ready atomic.Bool
	ready.Store(true)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "this is %s (version %s)\n", host, version)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "draining", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/burn", func(w http.ResponseWriter, r *http.Request) {
		ms, err := strconv.Atoi(r.URL.Query().Get("ms"))
		if err != nil || ms <= 0 || ms > 5000 {
			ms = 100
		}
		deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
		n := 0
		for time.Now().Before(deadline) {
			n++
		}
		fmt.Fprintf(w, "burned %dms on %s (%d loops)\n", ms, host, n)
	})

	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	log.Printf("listening on :8080 version=%s", version)

	// Graceful shutdown: SIGTERM -> fail readiness -> wait for endpoints and
	// load balancers to stop sending traffic -> finish in-flight requests.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	ready.Store(false)
	drain := 10 * time.Second
	if v, err := time.ParseDuration(os.Getenv("DRAIN_DELAY")); err == nil {
		drain = v
	}
	log.Printf("SIGTERM: draining for %s", drain)
	time.Sleep(drain)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Print("bye")
}
