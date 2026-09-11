package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/karnesanthosh/Averon/internal/config"
	"github.com/karnesanthosh/Averon/internal/health"
	"github.com/karnesanthosh/Averon/internal/metrics"
	"github.com/karnesanthosh/Averon/internal/middleware"
	"github.com/karnesanthosh/Averon/internal/router"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {

	// CLI flags
	configPath := flag.String("config", "config/config.json", "path to gateway config file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("NebulaGate %s\n", Version)
		os.Exit(0)
	}

	// Load gateway configuration (routes + backends)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Build routes from config
	var routes []*router.Route
	for _, rc := range cfg.Routes {
		route := &router.Route{
			PathPrefix:  rc.Path,
			StripPrefix: rc.StripPrefix,
			Backends:    rc.Backends,
			Auth:        rc.Auth,
			Strategy: rc.Strategy,
		}
		if rc.RateLimit != nil {
			route.RateLimitRate = rc.RateLimit.RequestPerSecond
			route.RateLimitBurst = rc.RateLimit.Burst
		}
		routes = append(routes, route)
	}

	// Create the router (initializes per-route load balancers)
	rt, err := router.NewRouter(routes)
	if err != nil {
		log.Fatalf("failed to create router: %v", err)
	}

	// Collect all backend servers across all routes
	// (needed for health checking and metrics)
	allBackends := rt.AllBackends()

	// Initial health check
	for _, s := range allBackends {
		s.SetAlive(health.CheckServer(s))

		status := "DOWN"
		if s.IsAlive() {
			status = "UP"
		}

		log.Printf("→ %s (%s) [%s]", s.ID, s.URL, status)
	}

	log.Printf("Loaded %d route(s) with %d total backend(s)", len(cfg.Routes), len(allBackends))

	// Parse health check interval
	healthInterval, err := time.ParseDuration(cfg.HealthCheckInterval)
	if err != nil {
		healthInterval = 10 * time.Second
	}

	// Start background health monitoring
	go health.StartHealthChecker(allBackends, healthInterval)

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metrics.NewHandler(allBackends))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.Handle("/", rt) // All other requests go through the router

	// Create HTTP server
	addr := fmt.Sprintf(":%d", cfg.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: middleware.Recovery(mux),
	}

	// Start NebulaGate
	go func() {
		log.Printf("NebulaGate %s listening on http://localhost%s", Version, addr)

		if err := httpServer.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)

	signal.Notify(
		sigChan,
		os.Interrupt,
		syscall.SIGTERM,
	)

	sig := <-sigChan

	log.Printf("received signal: %v", sig)
	log.Println("starting graceful shutdown...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("NebulaGate stopped gracefully")
}
