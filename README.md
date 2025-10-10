## VehicleTrackingBackend

A lightweight Go backend for vehicle tracking, built with `gin`, Redis, and Postgres. Features JWT auth with refresh tokens, WebSocket broadcast, rate limiting, health checks, and Prometheus metrics.

### Features
- Simple, structured Go service (Gin + Zap)
- JWT auth with refresh tokens (Argon2id password hashing)
- Redis-backed ingest and GEO storage, WebSocket notifications
- Health/readiness endpoints and `/metrics`
- Docker/Docker Compose ready

---

## Run the backend

### 1) Prerequisites
- Docker and Docker Compose
- Windows PowerShell or a POSIX shell

### 2) Environment
Create `.env` in project root or export env vars. Required variables for Compose are already referenced in `docker-compose.yaml`:

```env
DATABASE_USER=transport
DATABASE_PASSWORD=transport123
DATABASE_NAME=vehicletracking
```

JWT keys are read from files inside the container:
- `JWT_PRIVATE_KEY_PATH=/app/secrets/jwt_priv.pem`
- `JWT_PUBLIC_KEY_PATH=/app/secrets/jwt_pub.pem`

Generate keys locally into `./secrets`:

```powershell
make gen-jwt-keys
```

Compose will mount `./secrets` into the container at `/app/secrets`.

### 3) Start services

```powershell
docker compose up -d --build
```

App listens on `http://localhost:8080`.

### 4) Database migrations
If you haven’t run the refresh token migration yet, apply it like this (PowerShell):

```powershell
# Enable pgcrypto (for gen_random_uuid)
echo "CREATE EXTENSION IF NOT EXISTS pgcrypto;" | docker compose exec -T postgres psql -U transport -d vehicletracking -f -

# Apply migration 0005 (creates refresh_tokens)
type .\migrations\0005_refresh_tokens.up.sql | docker compose exec -T postgres psql -U transport -d vehicletracking -f -
```

### 5) Local (without Docker)
Run Postgres and Redis locally, then:

```powershell
$env:DATABASE_DSN = "postgres://transport:transport123@localhost:5432/vehicletracking?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
$env:JWT_PRIVATE_KEY_PATH = ".\secrets\jwt_priv.pem"
$env:JWT_PUBLIC_KEY_PATH  = ".\secrets\jwt_pub.pem"
go run .
```

---

## API Reference

Base URL: `http://localhost:8080`

### Health
- GET `/health/live`
  - 200: `{ "status": "ok", "service": "VehicleTrackingBackend" }`

- GET `/health/ready`
  - 200: `{ "status": "ready" }`
  - 503 when DB or Redis is unavailable

### Metrics
- GET `/metrics` (Prometheus exposition format)

### WebSocket
- GET `/ws`
  - Upgrades to a WS connection that receives light vehicle events published to Redis channels `vehicle:<busId>`.

### Auth
Routes: `/auth` (enabled when `JWT_PRIVATE_KEY_PATH` and `JWT_PUBLIC_KEY_PATH` are set)

- POST `/auth/register`
  - Request
    ```json
    { "name": "John Doe", "email": "john@example.com", "password": "P@ssw0rd!" }
    ```
  - Responses
    - 201: `{ "id": "<uuid>", "email": "john@example.com" }`
    - 400: `{ "error": "..." }`

- POST `/auth/login`
  - Request
    ```json
    { "email": "john@example.com", "password": "P@ssw0rd!" }
    ```
  - Responses
    - 200:
      ```json
      {
        "access_token": "<jwt>",
        "refresh_token": "<opaque-string>",
        "token_type": "Bearer",
        "expires_in": 900
      }
      ```
    - 401/503: `{ "error": "..." }`

- POST `/auth/refresh`
  - Request
    ```json
    { "refresh_token": "<opaque-string>" }
    ```
  - Responses
    - 200:
      ```json
      {
        "access_token": "<jwt>",
        "refresh_token": "<new-opaque-string>",
        "token_type": "Bearer",
        "expires_in": 900
      }
      ```
    - 401/500: `{ "error": "..." }`

- POST `/auth/logout`
  - Request
    ```json
    { "refresh_token": "<opaque-string>" }
    ```
  - Responses
    - 200: `{ "status": "ok" }`
    - 400/500: `{ "error": "..." }`

### API v1
Routes: `/api/v1` (rate limited)

- GET `/api/v1/ping`
  - 200: `{ "message": "pong" }`

- GET `/api/v1/version`
  - 200: `{ "version": "1.0.0", "service": "VehicleTrackingBackend" }`

- POST `/api/v1/locations`
  - Request
    ```json
    {
      "busId": "0b2b3646-1f6e-4fe1-b300-91a8b6a7f7d9",
      "latitude": 12.9716,
      "longitude": 77.5946,
      "timestamp": 1719930000,
      "speedKph": 32.5,
      "heading": 145
    }
    ```
    - Constraints: `latitude [-90,90]`, `longitude [-180,180]`, `timestamp` not more than 5 minutes in the future.
  - Responses
    - 204 No Content
    - 400/500: `{ "error": "..." }`

---

## cURL Quickstart

```bash
# Register
curl -sS http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"John","email":"john@example.com","password":"P@ssw0rd!"}'

# Login
curl -sS http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"john@example.com","password":"P@ssw0rd!"}'

# Refresh
curl -sS http://localhost:8080/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<opaque>"}'

# Post a location
curl -sS http://localhost:8080/api/v1/locations \
  -H 'Content-Type: application/json' \
  -d '{"busId":"<uuid>","latitude":12.9,"longitude":77.5,"timestamp":1719930000}' -i
```

---

## Test WebSocket + Redis end-to-end

### 1) Connect to WebSocket stream
- Using Node (no install):
```powershell
npx wscat -c ws://localhost:8080/ws
```

- Or using websocat (if installed):
```bash
websocat ws://localhost:8080/ws
```

### 2) Trigger an event via HTTP (recommended)
Posting to `/api/v1/locations` automatically publishes a minimal event to `vehicle:<busId>` which the WS will forward:

```powershell
curl -sS http://localhost:8080/api/v1/locations \
  -H 'Content-Type: application/json' \
  -d '{"busId":"BUS-123","latitude":12.9716,"longitude":77.5946,"timestamp":1719930000,"speedKph":32.5,"heading":145}' -i
```

You should see a JSON message on the WS client.

### 3) Trigger an event directly via Redis (manual publish)
You can publish a custom JSON message to the channel pattern subscribed by the broker (`vehicle:*`):

```powershell
docker compose exec -T redis redis-cli \
  PUBLISH vehicle:BUS-123 '{"msgId":"test-1","busId":"BUS-123","lat":12.9716,"lon":77.5946,"ts":1719930000}'
```

### 4) Optional: write raw data structures in Redis (bypassing HTTP)
The service also writes to these keys when you POST `/api/v1/locations`. You can emulate them manually for testing:

```powershell
# Append to positions stream
docker compose exec -T redis redis-cli XADD stream:positions * \
  msgId test-2 busId BUS-123 lat 12.9716 lon 77.5946 ts 1719930000 speed 30 heading 145

# Update GEO set and last-known hash
docker compose exec -T redis redis-cli GEOADD live:vehicles 77.5946 12.9716 BUS-123
docker compose exec -T redis redis-cli HSET vehicle:BUS-123:last lat 12.9716 lon 77.5946 ts 1719930000 speed 30

# Publish a lightweight WS event
docker compose exec -T redis redis-cli PUBLISH vehicle:BUS-123 '{"msgId":"test-3","busId":"BUS-123","lat":12.9716,"lon":77.5946,"ts":1719930000}'
```

---

## Configuration

Configuration is loaded from environment variables via Viper or `.env` under Docker:
- `SERVER_HOST` (default `0.0.0.0`)
- `SERVER_PORT` (default `8080`)
- `LOG_LEVEL` (default `info`)
- `REDIS_ADDR` (default `localhost:6379` when not in Docker)
- `DATABASE_DSN` (if not set, built from config struct)
- `JWT_PRIVATE_KEY_PATH`, `JWT_PUBLIC_KEY_PATH` (required for auth routes)

---

## Development

Common commands:
```bash
make run       # go run main.go
make build     # build into ./bin/api
make up        # docker compose up -d
make down      # docker compose down
```

Project entrypoint: `main.go`. Routes are registered in `internal/server/server.go`.

