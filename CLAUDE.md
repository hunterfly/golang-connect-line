# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go REST API application using the Fiber web framework with a hexagonal/clean architecture. The project implements a Todo API with PostgreSQL as the database, using GORM as the ORM.

**Key Technologies:**
- Go 1.24
- Fiber v2 (web framework)
- GORM (PostgreSQL ORM)
- Viper (configuration management)
- Swagger/OpenAPI documentation
- Air (hot reload for development)

## Development Commands

### Running the Application

**With Docker (recommended for local development):**
```bash
docker-compose up
```

**With Air (hot reload):**
```bash
air
# Builds to: ./tmp/app/engine
# Runs with: ./tmp/app/engine http
```

**Direct build and run:**
```bash
go build -o ./tmp/app/engine -tags musl ./cmd/api/main.go
./tmp/app/engine http
```

### Testing

```bash
go test ./...
```

### Swagger Documentation

Regenerate Swagger docs after updating API annotations:
```bash
swag init -g cmd/api/main.go -o docs
```

View documentation at: `http://localhost:{APP_PORT}/swagger/index.html`

### Code Quality

```bash
go fmt ./...
go vet ./...
```

## Architecture

### Hexagonal Architecture (Ports and Adapters)

The codebase follows strict hexagonal architecture with clear separation between domain, ports, and adapters:

```
internal/
├── domain/                    # Core domain layer (business entities and DTOs)
│   ├── todo.go               # Todo entity with GORM model
│   ├── dto.go                # Domain DTOs (request/response)
│   └── time_utils.go         # Domain utility functions
├── ports/                    # Port interfaces (contracts)
│   ├── input/               # Input ports (use cases)
│   │   └── todo_service.go  # TodoService interface
│   └── output/              # Output ports (data access)
│       └── todo_repository.go # TodoRepository interface
├── application/             # Use case implementations
│   └── todo_service.go      # Business logic for todo operations
└── adapters/               # Adapter implementations
    ├── input/              # Primary/Driving adapters
    │   └── http/          # HTTP adapter
    │       ├── handler.go     # Fiber HTTP handlers
    │       ├── request.go     # HTTP request DTOs
    │       └── response.go    # HTTP response DTOs
    └── output/            # Secondary/Driven adapters
        └── postgres/      # PostgreSQL adapter
            └── todo_repository.go # Repository implementation
```

**Dependency Flow:** HTTP Adapter → Application Service → Repository Adapter → Database

### Key Architectural Patterns

1. **Hexagonal Architecture Layers:**
   - **Domain (Core)**: Pure business logic and entities with no external dependencies
     - `internal/domain/todo.go` - Todo entity
     - `internal/domain/dto.go` - Domain transfer objects
   - **Ports (Interfaces)**: Define contracts for communication
     - Input ports: `internal/ports/input/todo_service.go` - What the app can do
     - Output ports: `internal/ports/output/todo_repository.go` - What the app needs
   - **Application**: Use case implementations
     - `internal/application/todo_service.go` - Business logic
   - **Adapters**: Concrete implementations
     - Input: `internal/adapters/input/http/` - HTTP handlers
     - Output: `internal/adapters/output/postgres/` - Database access

2. **Dependency Injection (Hexagonal Wiring):**
   - Wire dependencies in `protocal/http.go:61-67`
   - Flow: PostgreSQL Adapter → Application Service → HTTP Adapter
   - Example:
     ```go
     postgresRepo := postgres.NewTodoRepository(db)      // Output adapter
     srv := application.NewTodoService(postgresRepo)     // Application service
     hdl := httpAdapter.New(srv, db)                     // Input adapter
     ```

3. **DTO Conversion Pattern:**
   - HTTP layer has its own DTOs (`internal/adapters/input/http/request.go`, `response.go`)
   - Domain layer has domain DTOs (`internal/domain/dto.go`)
   - HTTP adapter converts between HTTP DTOs and domain DTOs
   - This maintains independence between HTTP concerns and domain logic

### Configuration Management

Configuration is managed via Viper with environment variable support:

- **Config files:** `configs/config.yml`
- **Environment mapping:** Config keys map to env vars (e.g., `app.port` → `APP_PORT`)
- **Initialization:** `configs.InitViper("./configs", env)` in protocal layer
- **Access:** `configs.GetViper()` returns typed config struct

Environment variables are defined in `.env` file (not committed).

### Entry Point and HTTP Server

- **Main entry:** `cmd/api/main.go` calls `protocal.ServeHTTP()`
- **Server setup:** `protocal/http.go:25` initializes Fiber, DB, and routes
- **Routes:** Defined at `protocal/http.go:64-74`
  - Health check: `GET /health`
  - Todo CRUD: `/v1/api/todo` endpoints
  - Swagger: `GET /swagger/*`
- **Graceful shutdown:** Handled via signal interrupt at `protocal/http.go:48-59`

### Database and Migrations

- **Driver:** `pkg/database_driver/gorm/postgres.go` handles connection
- **Auto-migration:** Triggered in `postgres.NewTodoRepository()` via `domain.MigrateDatabase()` at `internal/adapters/output/postgres/todo_repository.go:26`
- **Models:** Defined in `internal/domain/todo.go` with GORM hooks (BeforeCreate)

## Module Name Inconsistency

**Important:** The go.mod declares `module golang-template` but the project directory is `golang-connect-line`. All import paths use `golang-template`. When adding new packages, import as:
```go
import "golang-template/internal/..."
```

## Common Patterns

### Adding a New Endpoint (Hexagonal Approach)

1. **Define domain DTOs** in `internal/domain/dto.go`
2. **Update output port** interface in `internal/ports/output/todo_repository.go`
3. **Implement repository** in `internal/adapters/output/postgres/todo_repository.go`
4. **Update input port** interface in `internal/ports/input/todo_service.go`
5. **Implement application service** in `internal/application/todo_service.go`
6. **Define HTTP DTOs** in `internal/adapters/input/http/request.go` and `response.go`
7. **Add HTTP handler** in `internal/adapters/input/http/handler.go` with:
   - HTTP DTO → Domain DTO conversion
   - Swagger annotations
8. **Register route** in `protocal/http.go`
9. **Regenerate Swagger docs:** `swag init -g cmd/api/main.go -o docs`

**Key principle:** Dependencies point inward (adapters depend on ports, ports depend on domain)

### Image Handling

Images are base64 encoded/decoded in the PostgreSQL adapter:
- Encoding: `internal/adapters/output/postgres/todo_repository.go:40` (CreateTodo)
- Decoding: Throughout GetTodo, UpdateTodo, DeleteTodo methods

### Date Formatting

RFC3339 format is used: `2006-01-02T15:04:05Z`
- Defined in: `internal/domain/time_utils.go:11`
- Used in: PostgreSQL adapter for date conversions

### Pagination and Sorting

Implemented in application service layer (`internal/application/todo_service.go:43-86`):
- Default pagination: page=1, limit=100
- Default ordering: by ID ascending
- Offset calculation: `(page - 1) * perPage`

### DTO Conversion

The HTTP adapter converts between HTTP DTOs and domain DTOs:
- **HTTP → Domain:** Before calling service methods (see `handler.go:74-81`, `121-128`)
- **Domain → HTTP:** When returning responses (see `handler.go:249-266`)
- This keeps HTTP concerns separate from business logic
