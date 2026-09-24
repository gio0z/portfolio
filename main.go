package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"portfolio/internal/app"
)

func main() {
	cfg := app.LoadConfigFromEnv()

	portFlag := flag.String("port", cfg.Port, "Port to listen on (overrides PORT)")
	distFlag := flag.String("dist", cfg.FrontendDist, "Path to frontend dist directory (overrides FRONTEND_DIST)")
	flag.Parse()

	cfg.Port = *portFlag
	cfg.FrontendDist = *distFlag

	a, err := app.New(cfg)
	if err != nil {
		// cfg.String redacts secrets; the error names env vars, not values.
		log.Fatalf("Fatal: application composition failed in %s environment: %v\n%s", cfg.Env, err, cfg.String())
	}
	defer func() {
		_ = a.Close()
	}()

	srv := &http.Server{
		Addr:         "0.0.0.0:" + cfg.Port,
		Handler:      a,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("=====================================================")
		log.Printf("  Regio Dani Pangestu Portfolio Server (Go + Astro)")
		log.Printf("  Listening on http://localhost:%s", cfg.Port)
		log.Printf("  API Base:      http://localhost:%s/api/health", cfg.Port)
		log.Printf("  Serving Dist:  %s", cfg.FrontendDist)
		log.Printf("=====================================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Printf("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}
}
