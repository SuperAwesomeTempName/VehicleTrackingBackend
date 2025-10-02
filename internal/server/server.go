package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/config"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/handlers"
)

type Server struct {
	config *config.Config
	logger *zap.Logger
	router *gin.Engine
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Initialize logger
	var logger *zap.Logger
	var err error

	if cfg.Logger.Level == "debug" {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Set Gin mode based on log level
	if cfg.Logger.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	server := &Server{
		config: cfg,
		logger: logger,
		router: router,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all the routes
func (s *Server) setupRoutes() {
	// Initialize handlers
	healthHandler := handlers.NewHealthHandler()
	apiHandler := handlers.NewAPIHandler(s.logger)

	// Health check routes
	s.router.GET("/health", healthHandler.Health)
	s.router.GET("/health/ready", healthHandler.Ready)

	// API routes
	api := s.router.Group("/api/v1")
	{
		api.GET("/ping", apiHandler.Ping)
		api.GET("/version", apiHandler.Version)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	address := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)

	srv := &http.Server{
		Addr:    address,
		Handler: s.router,
	}

	// Start server in a goroutine
	go func() {
		s.logger.Info("Starting server", zap.String("address", address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	// Give the server 30 seconds to finish handling requests
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	s.logger.Info("Server exited")
	return nil
}
