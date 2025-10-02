# VehicleTrackingBackend

A simple, clean Go boilerplate for building REST APIs with Gin framework.

## Features

- 🚀 **Simple & Clean**: Minimal boilerplate code
- 🏗️ **Well Structured**: Clean architecture with separation of concerns
- ⚡ **Fast**: Built with Gin framework
- 📝 **Configurable**: Environment-based configuration with Viper
- 🔍 **Logging**: Structured logging with Zap
- 🏥 **Health Checks**: Built-in health and readiness endpoints
- 🐳 **Docker Ready**: Dockerfile and docker-compose included

## Project Structure

```
.
├── main.go                 # Application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── server/            # HTTP server setup
│   └── handlers/          # HTTP request handlers
├── docker-compose.yaml    # Docker services configuration
├── Dockerfile            # Docker build configuration
├── Makefile             # Build and development commands
└── README.md           # This file
```

## Quick Start

### Using Docker (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/SuperAwesomeTempName/VehicleTrackingBackend.git
cd VehicleTrackingBackend
```

2. Start the application:
```bash
docker-compose up --build
```

The server will be available at `http://localhost:8080`

### Local Development

1. Install Go 1.21 or later

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run main.go
```

## API Endpoints

### Health Checks
- `GET /health` - Basic health check
- `GET /health/ready` - Readiness probe

### API
- `GET /api/v1/ping` - Simple ping endpoint
- `GET /api/v1/version` - Get API version

## Configuration

The application can be configured using environment variables or a config file.

### Environment Variables
- `SERVER_HOST` - Server host (default: "0.0.0.0")
- `SERVER_PORT` - Server port (default: "8080")
- `LOG_LEVEL` - Log level (default: "info")

## Development

### Available Make Commands

```bash
make build          # Build the application
make run             # Run the application locally
make test            # Run tests
make clean           # Clean build artifacts
make up              # Start with docker-compose
make down            # Stop docker-compose services
```

### Adding New Features

1. **Add new handlers** in `internal/handlers/`
2. **Register routes** in `internal/server/server.go`
3. **Add configuration** in `internal/config/config.go`
4. **Update dependencies** in `go.mod`

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.
