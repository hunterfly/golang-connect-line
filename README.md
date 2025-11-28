# Go Todo API with LINE Chatbot

REST API service with LINE Messaging API integration, built with Go, Fiber, and PostgreSQL using Hexagonal Architecture (Ports & Adapters).

## Features

- **Todo CRUD API** - Create, read, update, delete todo items
- **LINE Chatbot** - Interactive chatbot with webhook integration
- **Hexagonal Architecture** - Clean separation between domain, ports, and adapters
- **PostgreSQL Database** - Data persistence with GORM
- **Swagger Documentation** - Auto-generated API docs
- **Docker Support** - Containerized development environment
- **Hot Reload** - Development with Air

## Tech Stack

- **Go 1.24** - Programming language
- **Fiber v2** - Web framework
- **GORM** - PostgreSQL ORM
- **LINE Bot SDK v8** - LINE Messaging API integration
- **Viper** - Configuration management
- **Swagger** - API documentation
- **Docker** - Containerization

## Architecture

This project follows **Hexagonal Architecture** (Ports & Adapters):

```
internal/
├── domain/                    # Core domain (entities, DTOs)
│   ├── todo.go               # Todo entity
│   ├── line.go               # LINE webhook entities
│   └── dto.go                # Domain transfer objects
├── ports/                    # Port interfaces
│   ├── input/               # Input ports (use cases)
│   │   ├── todo_service.go
│   │   └── line_webhook_service.go
│   └── output/              # Output ports (data access)
│       ├── todo_repository.go
│       └── line_client.go
├── application/             # Use case implementations
│   ├── todo_service.go
│   └── line_webhook_service.go
└── adapters/               # Adapter implementations
    ├── input/http/         # HTTP handlers
    │   ├── handler.go
    │   └── line_webhook_handler.go
    └── output/             # External services
        ├── postgres/       # Database adapter
        └── line/           # LINE API adapter
```

**Dependency Flow:** HTTP → Application Service → Repository/LINE Client → External Services

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- (Optional) Air for hot reload
- LINE Developer Account for chatbot features

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd golang-connect-line
```

2. **Configure environment variables**
```bash
# Edit .env file
LINE_CHANNEL_SECRET=your_channel_secret_here
LINE_CHANNEL_TOKEN=your_channel_access_token_here
```

3. **Run with Docker**
```bash
docker-compose up -d
docker-compose logs --follow
```

4. **Or run with Air (hot reload)**
```bash
air
```

The API will be available at `http://localhost:9089`

## API Endpoints

### Todo API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/v1/api/todo` | Create todo |
| `PUT` | `/v1/api/todo` | Update todo |
| `DELETE` | `/v1/api/todo/:id` | Delete todo |
| `GET` | `/v1/api/todo/:id` | Get specific todo |
| `GET` | `/v1/api/todo` | List all todos |

### LINE Webhook

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/webhook/line` | LINE webhook endpoint |

### Documentation

Swagger UI: `http://localhost:9089/swagger/index.html`

## LINE Chatbot Setup

### 1. Create LINE Official Account

1. Go to [LINE Developers Console](https://developers.line.biz/console/)
2. Create a new provider or use existing
3. Create a new Messaging API channel
4. Get your **Channel Secret** and **Channel Access Token**

### 2. Configure Environment

Update `.env` with your LINE credentials:
```bash
LINE_CHANNEL_SECRET=your_actual_channel_secret
LINE_CHANNEL_TOKEN=your_actual_channel_access_token
```

### 3. Setup Webhook URL

For local development, use [ngrok](https://ngrok.com/):
```bash
ngrok http 9089
```

Then set webhook URL in LINE Console:
```
https://your-ngrok-url.ngrok.io/webhook/line
```

Enable **Use webhook** and disable **Auto-reply messages** in LINE Console.

### 4. Test the Bot

Add your LINE Official Account as a friend and try:
- Send any message → Bot echoes back
- `/help` → Show available commands
- `/about` → Bot information
- `/echo hello world` → Bot replies "hello world"

## Development

### Project Structure

```
.
├── cmd/api/main.go          # Application entry point
├── configs/                 # Configuration files
│   ├── config.yml          # Config mapping
│   └── config.go           # Config structs
├── internal/               # Application code
├── pkg/                    # Reusable packages
│   ├── database_driver/    # Database connection
│   └── validator/          # Validation utilities
├── protocal/               # Server setup & routing
│   └── http.go
├── docs/                   # Swagger documentation
├── docker-compose.yml      # Docker services
└── .air.toml              # Hot reload config
```

### Generate Swagger Docs

After updating API annotations:
```bash
swag init -g cmd/api/main.go -o docs
```

### Build

```bash
go build -o ./tmp/app/engine ./cmd/api/main.go
./tmp/app/engine http
```

### Testing

```bash
go test ./...
```

## Adding New Features

### Adding a New Endpoint (Following Hexagonal Architecture)

1. **Define domain DTOs** in `internal/domain/dto.go`
2. **Update output port** interface in `internal/ports/output/`
3. **Implement repository** in `internal/adapters/output/postgres/`
4. **Update input port** interface in `internal/ports/input/`
5. **Implement application service** in `internal/application/`
6. **Define HTTP DTOs** in `internal/adapters/input/http/request.go` and `response.go`
7. **Add HTTP handler** in `internal/adapters/input/http/handler.go`
8. **Register route** in `protocal/http.go`
9. **Regenerate Swagger:** `swag init -g cmd/api/main.go -o docs`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment (local/dev/prod) | local |
| `APP_DEBUG` | Debug mode | true |
| `APP_PORT` | Server port | 9089 |
| `POSTGRES_HOST` | Database host | database |
| `POSTGRES_PORT` | Database port | 5432 |
| `POSTGRES_USERNAME` | Database username | postgres |
| `POSTGRES_PASSWORD` | Database password | - |
| `POSTGRES_DATABASE` | Database name | postgres |
| `POSTGRES_SSLMODE` | SSL mode | false |
| `LINE_CHANNEL_SECRET` | LINE channel secret | - |
| `LINE_CHANNEL_TOKEN` | LINE channel access token | - |

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
