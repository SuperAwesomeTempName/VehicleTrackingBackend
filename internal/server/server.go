package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/auth"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/config"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/db"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/handlers"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/middleware"
	redisclient "github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/redis"
	"github.com/SuperAwesomeTempName/VehicleTrackingBackend/internal/ws"
)

type Server struct {
	config *config.Config
	logger *zap.Logger
	router *gin.Engine

	redis  *redisclient.Client
	broker *ws.Broker
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

	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // TODO: change for your frontend
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	s := &Server{
		config: cfg,
		logger: logger,
		router: router,
	}

	// Initialize database connection early so readiness checks can succeed
	{
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		dsn := os.Getenv("DATABASE_DSN")
		if dsn == "" {
			dsn = fmt.Sprintf(
				"postgres://%s:%s@%s:%d/%s?sslmode=%s",
				cfg.Database.User,
				cfg.Database.Password,
				cfg.Database.Host,
				cfg.Database.Port,
				cfg.Database.Name,
				cfg.Database.SSLMode,
			)
		}

		if err := db.Connect(ctx, dsn); err != nil {
			logger.Warn("database connect failed", zap.Error(err))
		}
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all the routes
func (s *Server) setupRoutes() {
	// Initialize handlers
	healthHandler := handlers.NewHealthHandler("VehicleTrackingBackend")
	apiHandler := handlers.NewAPIHandler(s.logger)

	// Redis client for locations
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		// Default to localhost for local development outside Docker
		redisAddr = "localhost:6379"
	}
	r := redisclient.New(redisAddr)
	s.redis = r

	// WebSocket broker
	broker := ws.NewBroker(r)
	s.broker = broker

	// Locations handler uses the redis client
	locations := handlers.NewLocationsGinHandler(r)

	// --- Health Check Routes ---
	s.router.GET("/health/live", func(c *gin.Context) {
		healthHandler.Health(c)
	})

	s.router.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// Check DB connectivity
		if err := db.Ping(ctx); err != nil {
			s.logger.Warn("readiness: db ping failed", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_unavailable"})
			return
		}

		// Check Redis connectivity
		if err := r.RDB().Ping(ctx).Err(); err != nil {
			s.logger.Warn("readiness: redis ping failed", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis_unavailable"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Prometheus metrics endpoint
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// --- Auth Routes ---
	authGroup := s.router.Group("/auth")
	{
		authGroup.POST("/register", auth.RegisterHandler)

		// Initialize JWT manager when both key paths are set
		priv := os.Getenv("JWT_PRIVATE_KEY_PATH")
		pub := os.Getenv("JWT_PUBLIC_KEY_PATH")
		if priv != "" && pub != "" {
			jwtMgr, err := auth.NewJWTManagerFromFiles(priv, pub, "vehicletracking", 15*time.Minute)
			if err != nil {
				s.logger.Warn("failed to initialize JWT manager", zap.Error(err))
				authGroup.POST("/login", func(c *gin.Context) { c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()}) })
				authGroup.POST("/refresh", func(c *gin.Context) { c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()}) })
				authGroup.POST("/logout", func(c *gin.Context) { c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()}) })
				return
			}
			authGroup.POST("/login", auth.LoginHandler(jwtMgr))
			authGroup.POST("/refresh", auth.RefreshHandler(jwtMgr))
			authGroup.POST("/logout", auth.LogoutHandler())
		} else {
			// Keys not provided
			authGroup.POST("/login", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JWT not configured: set JWT_PRIVATE_KEY_PATH and JWT_PUBLIC_KEY_PATH"})
			})
			authGroup.POST("/refresh", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JWT not configured: set JWT_PRIVATE_KEY_PATH and JWT_PUBLIC_KEY_PATH"})
			})
			authGroup.POST("/logout", func(c *gin.Context) {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "JWT not configured: set JWT_PRIVATE_KEY_PATH and JWT_PUBLIC_KEY_PATH"})
			})
		}
	}

	limiter := middleware.RateLimiterMiddleware(r.RDB(), 60, time.Minute)

	// --- API Routes ---
	api := s.router.Group("/api/v1")
	api.Use(limiter)
	{
		api.GET("/ping", apiHandler.Ping)
		api.GET("/version", apiHandler.Version)
		api.POST("/locations", locations.Post)
	}

	// WebSocket endpoint
	s.router.GET("/ws", func(c *gin.Context) {
		broker.ServeWS(c.Writer, c.Request)
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	address := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)

	srv := &http.Server{
		Addr:    address,
		Handler: s.router,
	}

	// Start server asynchronously
	go func() {
		s.logger.Info("Starting server", zap.String("address", address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful shutdown on signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown", zap.Error(err))
	}

	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			s.logger.Warn("Failed to close Redis client", zap.Error(err))
		}
	}

	db.Close()

	_ = s.logger.Sync()
	s.logger.Info("Server exited cleanly")
	return nil
}
