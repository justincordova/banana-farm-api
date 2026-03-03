# Banana Farm API

A Go CRUD REST API for managing a banana farm. Built with production-quality patterns.

## Tech Stack

- **Router**: Chi (`github.com/go-chi/chi/v5`)
- **ORM**: GORM (`gorm.io/gorm`)
- **Database**: SQLite (`gorm.io/driver/sqlite`)
- **Validation**: go-playground/validator
- **Logging**: slog (stdlib) with tint (dev) / JSON (prod)
- **Testing**: testify

## Getting Started

### Prerequisites

- Go 1.21+

### Installation

```bash
# Install dependencies
go mod download

# Copy environment template
cp .env.example .env

# Run with hot reload (dev)
air

# Or run directly
go run main.go
```

### Configuration

Create a `.env` file:

```env
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug
DB_PATH=./banana_farm.db
CORS_ALLOWED_ORIGINS=http://localhost:3000
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m
```

## API Endpoints

### Health
- `GET /health` - DB status, app env, uptime

### Farms
- `GET /farms` - List (paginated, filterable)
- `POST /farms` - Create
- `GET /farms/{id}` - Get by ID
- `PUT /farms/{id}` - Update
- `DELETE /farms/{id}` - Delete
- `GET /farms/{id}/trees` - List trees for farm
- `GET /farms/{id}/workers` - List workers for farm
- `GET /farms/{id}/tools` - List tools for farm
- `GET /farms/{id}/stats` - Farm statistics

### Banana Trees
- `GET /trees` - List (paginated, filterable)
- `POST /trees` - Create
- `GET /trees/{id}` - Get by ID
- `PUT /trees/{id}` - Update
- `DELETE /trees/{id}` - Delete
- `GET /trees/{id}/bunches` - List bunches for tree

### Bunches
- `GET /bunches` - List (paginated)
- `POST /bunches` - Create
- `GET /bunches/{id}` - Get by ID
- `PUT /bunches/{id}` - Update
- `DELETE /bunches/{id}` - Delete
- `GET /bunches/{id}/bananas` - List bananas for bunch

### Bananas
- `GET /bananas` - List (paginated, filterable)
- `POST /bananas` - Create
- `GET /bananas/{id}` - Get by ID
- `PUT /bananas/{id}` - Update
- `DELETE /bananas/{id}` - Delete

### Tools
- `GET /tools` - List (paginated, filterable)
- `POST /tools` - Create
- `GET /tools/{id}` - Get by ID
- `PUT /tools/{id}` - Update
- `DELETE /tools/{id}` - Delete

### Workers
- `GET /workers` - List (paginated, filterable)
- `POST /workers` - Create
- `GET /workers/{id}` - Get by ID
- `PUT /workers/{id}` - Update
- `DELETE /workers/{id}` - Delete

## Testing

```bash
go test ./...              # Run all tests
go test -v ./...           # Verbose output
go test -cover ./...       # With coverage
```

## Project Structure

```
banana-farm-api/
├── main.go              # Entry point
├── config/              # Config & logging
├── database/            # GORM connection
├── models/              # Data models
├── handlers/            # HTTP handlers
├── routes/              # Chi router setup
├── middleware/          # Logging, error handling
└── helpers/             # Response, pagination helpers
```
