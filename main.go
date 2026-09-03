package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"portfolio/internal/adminauth"
	"portfolio/pkg/api"
)

type spaHandler struct {
	staticPath  string
	indexPath   string
	apiServer   *api.Server
	authService *adminauth.Service
	authMux     *http.ServeMux
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// If the path starts with /api/admin/auth/, delegate to auth handler if configured
	if h.authMux != nil && strings.HasPrefix(r.URL.Path, "/api/admin/auth/") {
		h.authMux.ServeHTTP(w, r)
		return
	}

	// If the path starts with /api/, delegate directly to API server
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h.apiServer.ServeHTTP(w, r)
		return
	}

	// Build full path for static file
	path := filepath.Join(h.staticPath, filepath.Clean(r.URL.Path))

	// Check if file exists and is not a directory
	fi, err := os.Stat(path)
	if err == nil && !fi.IsDir() {
		http.ServeFile(w, r, path)
		return
	}

	// SPA Fallback: if not found, serve index.html
	indexPath := filepath.Join(h.staticPath, h.indexPath)
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	// If no frontend build found, fall back to API server root info
	h.apiServer.ServeHTTP(w, r)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	portFlag := flag.String("port", port, "Port to listen on")
	frontendDist := flag.String("dist", "./frontend/dist", "Path to frontend dist directory")
	flag.Parse()

	apiServer := api.NewServer()

	// Configure admin authentication from environment
	authCfg := adminauth.LoadConfigFromEnv()
	var authService *adminauth.Service
	var authMux *http.ServeMux

	// In production, require admin auth configuration to fail closed
	if strings.EqualFold(authCfg.Environment, "production") {
		var err error
		authService, err = adminauth.NewService(authCfg)
		if err != nil {
			log.Fatalf("Fatal: Admin authentication configuration failed in production: %v", err)
		}
		authMux = http.NewServeMux()
		authService.RegisterRoutes(authMux)
		log.Printf("  Admin Auth:    Enabled [Production] (Owner: %s)", authCfg.AllowedLogin)
	} else if authCfg.ClientID != "" && authCfg.ClientSecret != "" && authCfg.SessionSecret != "" {
		var err error
		authService, err = adminauth.NewService(authCfg)
		if err != nil {
			log.Printf("Warning: Admin auth initialization failed: %v", err)
		} else {
			authMux = http.NewServeMux()
			authService.RegisterRoutes(authMux)
			log.Printf("  Admin Auth:    Enabled [Dev] (Owner: %s)", authCfg.AllowedLogin)
		}
	} else {
		log.Printf("  Admin Auth:    Disabled (missing credentials in %s environment)", authCfg.Environment)
	}

	handler := &spaHandler{
		staticPath:  *frontendDist,
		indexPath:   "index.html",
		apiServer:   apiServer,
		authService: authService,
		authMux:     authMux,
	}

	addr := fmt.Sprintf("0.0.0.0:%s", *portFlag)
	log.Printf("=====================================================")
	log.Printf("  Regio Dani Pangestu Portfolio Server (Go + Vite)")
	log.Printf("  Listening on http://localhost:%s", *portFlag)
	log.Printf("  API Base:      http://localhost:%s/api/health", *portFlag)
	log.Printf("  Serving Dist:  %s", *frontendDist)
	log.Printf("=====================================================")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
