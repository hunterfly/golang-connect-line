# Go Todo API with LINE Chatbot + LM Studio AI

REST API service with LINE Messaging API integration and **LM Studio AI-powered conversations**, built with Go, Fiber, and PostgreSQL using Hexagonal Architecture (Ports & Adapters).

## Features

- **AI-Powered LINE Chatbot** - Intelligent conversations powered by LM Studio (local LLM)
- **Multi-Turn Conversations** - Session-based context with configurable history
- **Todo CRUD API** - Create, read, update, delete todo items
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
- **LM Studio** - Local LLM inference server (OpenAI-compatible API)
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
│   ├── session.go            # Conversation session entity
│   └── dto.go                # Domain transfer objects
├── ports/                    # Port interfaces
│   ├── input/               # Input ports (use cases)
│   │   ├── todo_service.go
│   │   └── line_webhook_service.go
│   └── output/              # Output ports (data access)
│       ├── todo_repository.go
│       ├── line_client.go
│       ├── lmstudio_client.go    # LM Studio AI client
│       └── session_store.go      # Session storage
├── application/             # Use case implementations
│   ├── todo_service.go
│   └── line_webhook_service.go
└── adapters/               # Adapter implementations
    ├── input/http/         # HTTP handlers
    │   ├── handler.go
    │   └── line_webhook_handler.go
    └── output/             # External services
        ├── postgres/       # Database adapter
        ├── line/           # LINE API adapter
        ├── lmstudio/       # LM Studio AI adapter
        └── memory/         # In-memory session storage
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
- Send any message → Bot replies with AI-generated response
- `/help` → Show available commands
- `/about` → Bot information
- `/echo hello world` → Bot replies "hello world"
- `/clear` → Clear conversation history

## LM Studio Setup

LM Studio provides a local LLM inference server with an OpenAI-compatible API. This allows the LINE bot to generate AI-powered responses.

### 1. Install LM Studio

1. Download LM Studio from [https://lmstudio.ai/](https://lmstudio.ai/)
2. Install and launch LM Studio
3. Download a model (recommended: Llama 3, Mistral, or Phi-3)

### 2. Start the Local Server

1. In LM Studio, go to the **Local Server** tab (left sidebar)
2. Select your downloaded model from the dropdown
3. Click **Start Server**
4. The server will start on `http://localhost:1234` by default

![LM Studio Server](https://lmstudio.ai/static/images/server-tab.png)

### 3. Configure Environment Variables

Add these to your `.env` file:

```bash
# LM Studio Configuration
LMSTUDIO_BASE_URL=http://localhost:1234    # LM Studio server URL
LMSTUDIO_MODEL=                             # Optional: specific model name (auto-detects if empty)
LMSTUDIO_TIMEOUT=120                        # Request timeout in seconds
LMSTUDIO_SYSTEM_PROMPT=You are a helpful assistant responding via LINE messaging.

# Session Configuration (Optional)
SESSION_TIMEOUT=30                          # Session timeout in minutes (default: 30)
SESSION_MAX_TURNS=10                        # Max conversation turns to keep (default: 10)
```

### 4. Verify Connection

Once both the Go application and LM Studio server are running:

1. Send a message to your LINE bot
2. The bot should respond with an AI-generated message
3. Check application logs for: `Session config: timeout=30m0s, maxTurns=10`

### LM Studio Configuration Options

| Variable | Description | Default |
|----------|-------------|---------|
| `LMSTUDIO_BASE_URL` | LM Studio server URL | `http://localhost:1234` |
| `LMSTUDIO_MODEL` | Model name (empty = auto-detect first available) | - |
| `LMSTUDIO_TIMEOUT` | Request timeout in seconds | 120 |
| `LMSTUDIO_SYSTEM_PROMPT` | System prompt for AI personality | "You are a helpful assistant..." |

### Session Configuration Options

| Variable | Description | Default |
|----------|-------------|---------|
| `SESSION_TIMEOUT` | Session timeout in minutes | 30 |
| `SESSION_MAX_TURNS` | Max conversation turns to keep in history | 10 |

### Troubleshooting LM Studio

**Bot not responding with AI messages:**
- Ensure LM Studio server is running (check the Local Server tab)
- Verify `LMSTUDIO_BASE_URL` matches your LM Studio server address
- Check application logs for connection errors

**Slow responses:**
- Try a smaller/faster model (e.g., Phi-3 mini instead of Llama 70B)
- Increase `LMSTUDIO_TIMEOUT` if responses are timing out
- Reduce `SESSION_MAX_TURNS` to send less context

**Out of memory errors:**
- Use a smaller model that fits your GPU/RAM
- Reduce `SESSION_MAX_TURNS` to limit context size

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

### Application

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment (local/dev/prod) | local |
| `APP_DEBUG` | Debug mode | true |
| `APP_PORT` | Server port | 9089 |

### Database

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_HOST` | Database host | database |
| `POSTGRES_PORT` | Database port | 5432 |
| `POSTGRES_USERNAME` | Database username | postgres |
| `POSTGRES_PASSWORD` | Database password | - |
| `POSTGRES_DATABASE` | Database name | postgres |
| `POSTGRES_SSLMODE` | SSL mode | false |

### LINE Messaging API

| Variable | Description | Default |
|----------|-------------|---------|
| `LINE_CHANNEL_SECRET` | LINE channel secret | - |
| `LINE_CHANNEL_TOKEN` | LINE channel access token | - |

### LM Studio (AI)

| Variable | Description | Default |
|----------|-------------|---------|
| `LMSTUDIO_BASE_URL` | LM Studio server URL | http://localhost:1234 |
| `LMSTUDIO_MODEL` | Model name (auto-detects if empty) | - |
| `LMSTUDIO_TIMEOUT` | Request timeout in seconds | 120 |
| `LMSTUDIO_SYSTEM_PROMPT` | System prompt for AI | "You are a helpful assistant..." |

### Session Management

| Variable | Description | Default |
|----------|-------------|---------|
| `SESSION_TIMEOUT` | Session timeout in minutes | 30 |
| `SESSION_MAX_TURNS` | Max conversation turns in history | 10 |

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
