# Distributed WebSockets

A horizontally scalable WebSocket server built with Go. Multiple server instances coordinate through Redis (connection registry) and RabbitMQ (message routing via topic exchange) to deliver messages between users regardless of which server they are connected to.

## How It Works

- Each server instance maintains its own local WebSocket connections.
- When a user connects, their `userID` and the `serverID` they connected to are stored in Redis.
- When a user sends a message to another user:
  1. The server checks if the target user is connected locally. If so, it delivers directly.
  2. Otherwise, it looks up the target user's server in Redis and publishes the message to RabbitMQ with a routing key of `server.{targetServerID}`.
- Each server subscribes to its own routing key and delivers incoming messages to the appropriate local WebSocket connection.

## Prerequisites

- Go 1.26+
- Docker (for Redis and RabbitMQ)
- [Air](https://github.com/air-verse/air) (optional, for hot-reloading during development)

## Getting Started

### 1. Start Redis and RabbitMQ

```bash
docker compose up redis rabbitmq -d
```

### 2. Configure Environment

Copy the example env file and fill in the values:

```bash
cp .env.example .env
```

### 3. Run a Single Instance

```bash
go run cmd/main.go
```

Or with Air for hot-reloading:

```bash
air
```

### 4. Run Multiple Instances (Testing Distribution)

Terminal 1:

```bash
SERVER_ID=server-1 PORT=:8080 go run cmd/main.go
```

Terminal 2:

```bash
SERVER_ID=server-2 PORT=:8081 go run cmd/main.go
```

### 5. Run with Docker Compose (All Services)

```bash
docker compose up --build
```

## WebSocket Usage

Connect to the WebSocket endpoint with a user ID:

```
ws://localhost:8080/ws?user_id=alice
```

Send a message to another user:

```json
{ "to": "bob", "body": "hello" }
```

The recipient receives:

```json
{ "from": "alice", "to": "bob", "body": "hello" }
```

Every frame also says where it came from: messages include `from_server`, `to_server` and `route` (`local` or `rabbitmq`), and the sender gets an `ack` (or an `error` if the recipient is offline). The demo frontend uses these fields.

## Live Demo Frontend (`web/`)

A Vite + React chat for two devices. The person who starts a chat connects to `server-1`. The person who joins (by QR code or link) connects to `server-2`. Each message animates its path through Redis and RabbitMQ, and a wire log shows the raw frames.

```bash
# run two servers locally (see step 4 above), then:
cd web
cp .env.example .env   # VITE_SERVERS lists the server WebSocket URLs
npm install
npm run dev
```

## Deploying the Demo

**Backend** (a VPS with Docker; ports 80/443 open; DNS A record for your API domain pointing at it):

```bash
cp deploy/.env.example deploy/.env   # set API_DOMAIN and passwords
docker compose -f deploy/compose.yaml --env-file deploy/.env up -d --build
```

This runs `server-1`, `server-2`, Redis, RabbitMQ and Caddy (automatic HTTPS). The two servers are exposed at `wss://<API_DOMAIN>/s1/ws` and `wss://<API_DOMAIN>/s2/ws`.

**Frontend** (Vercel, Netlify, Cloudflare Pages, etc.): set the root directory to `web`, the build command to `npm run build` and the output directory to `dist`, then set:

```
VITE_SERVERS=server-1=wss://<API_DOMAIN>/s1/ws,server-2=wss://<API_DOMAIN>/s2/ws
```

Point a subdomain (e.g. `sockets.yourdomain.com`) at it.

## Project Structure

```
cmd/
  main.go          - Application entry point
  api/             - HTTP routes
  ws/              - WebSocket handler and message routing
config/
  cache/           - Redis client initialization
  file/            - File path utilities
internal/
  adapter/         - Interface implementations (Redis, RabbitMQ)
  domain/          - Interfaces and domain types
  service/         - Business logic layer
pkg/
  env/             - Environment variable loading
  errors/          - Custom error types
  response/        - HTTP response helpers
  util/            - Utilities (server ID, panic helpers)
```

## Environment Variables

See `.env.example` for the full list. Key variables for the distributed socket functionality:

| Variable            | Description                                | Default    |
| ------------------- | ------------------------------------------ | ---------- |
| `PORT`              | HTTP server port                           | `:5000`    |
| `SERVER_ID`         | Unique identifier for this server instance | hostname   |
| `REDIS_ADDR`        | Redis address                              | (required) |
| `REDIS_PASSWORD`    | Redis password                             | (required) |
| `RABBITMQ_USER`     | RabbitMQ username                          | `guest`    |
| `RABBITMQ_ADDR`     | RabbitMQ address                           | (required) |
| `RABBITMQ_PASSWORD` | RabbitMQ password                          | (required) |


## Related Articles

- [**Distributed Sockets**](https://www.umohsg.com/blog/distributed-sockets-b54b91e0-adc1-4333-9190-66e28f7b7b19)
