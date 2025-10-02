package main

import (
	"log"

	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/config"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize and start server
	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
