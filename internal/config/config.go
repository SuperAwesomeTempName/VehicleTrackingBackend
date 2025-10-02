package config

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

// Config holds all configuration for our application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type LoggerConfig struct {
	Level string
}

// Load reads configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.name", "postgres")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("logger.level", "info")

	// Read from environment variables
	viper.AutomaticEnv()

	// Try to read config file (optional)
	viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", viper.GetString("server.port")),
			Host: getEnvOrDefault("SERVER_HOST", viper.GetString("server.host")),
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DATABASE_HOST", viper.GetString("database.host")),
			Port:     getEnvIntOrDefault("DATABASE_PORT", viper.GetInt("database.port")),
			User:     getEnvOrDefault("DATABASE_USER", viper.GetString("database.user")),
			Password: getEnvOrDefault("DATABASE_PASSWORD", viper.GetString("database.password")),
			Name:     getEnvOrDefault("DATABASE_NAME", viper.GetString("database.name")),
			SSLMode:  getEnvOrDefault("DATABASE_SSLMODE", viper.GetString("database.sslmode")),
		},
		Logger: LoggerConfig{
			Level: getEnvOrDefault("LOG_LEVEL", viper.GetString("logger.level")),
		},
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
