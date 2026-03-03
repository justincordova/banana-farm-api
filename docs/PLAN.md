# Banana Farm API - Project Plan

A Go CRUD REST API for managing a banana farm. Built as a learning project with production-quality patterns.

---

## Tech Stack

| Concern | Package | Notes |
|---|---|---|
| Router | `github.com/go-chi/chi/v5` | Lightweight, idiomatic, stdlib-compatible |
| CORS | `github.com/go-chi/cors` | Chi CORS middleware |
| Rate Limiting | `github.com/go-chi/httprate` | IP-based rate limiting |
| ORM | `gorm.io/gorm` | Most popular Go ORM |
| Database | `gorm.io/driver/sqlite` | SQLite, zero setup |
| Env Config | `github.com/caarlos0/env/v11` | Struct-based env loading |
| .env File | `github.com/joho/godotenv` | Load .env file |
| Validation | `github.com/go-playground/validator/v10` | Struct validation |
| Logging | `log/slog` (stdlib) | Structured logging |
| Log Colors | `github.com/lmittmann/tint` | Colored slog handler for dev |
| Log Rotation | `gopkg.in/lumberjack.v2` | File rotation for prod |
| Testing | `github.com/stretchr/testify` | Assertions and test helpers |
| Hot Reload | `github.com/air-verse/air` | Dev tool, not a dependency |

---

## Domain Model

```
Farm
├── name          string
├── location      string
├── size_acres    float64
├── established   time.Time
├── Workers[]
├── Tools[]
└── BananaTrees[]
     ├── variety       string (cavendish, plantain, lady_finger, red, blue_java)
     ├── planted_at    time.Time
     ├── status        string (planted, growing, flowering, fruiting, harvested, dead)
     ├── health        string (healthy, diseased, pest_damaged)
     └── Bunches[]
          ├── harvested_at  *time.Time
          ├── weight_kg     float64
          └── Bananas[]
               ├── hand_number   int
               ├── size          string (small, medium, large)
               ├── ripeness      string (green, turning, ripe, overripe)
               └── weight_grams  float64

Worker
├── name    string
├── role    string (farmer, harvester, irrigator, supervisor)
├── farm_id uint

Tool
├── name      string
├── type      string (machete, pruning_shears, irrigation_pump, fertilizer_sprayer, harvesting_knife, wheelbarrow, bunch_cover)
├── condition string (new, good, worn, broken)
├── farm_id   uint
```

### Relationships
- Farm has many BananaTrees, Workers, Tools
- BananaTree belongs to Farm, has many Bunches
- Bunch belongs to BananaTree, has many Bananas
- Banana belongs to Bunch
- Worker belongs to Farm
- Tool belongs to Farm

---

## Project Structure

```
banana-farm-api/
├── main.go                     # Entry point: server setup, graceful shutdown
├── .env                        # Environment variables
├── .env.example                # Template for .env
├── go.mod
├── .air.toml                   # Hot reload config
│
├── config/
│   ├── config.go               # Config struct, env loading, APP_ENV
│   └── logger.go               # slog setup (tint for dev, JSON+lumberjack for prod)
│
├── database/
│   └── database.go             # GORM connection, auto-migrate, close
│
├── models/
│   ├── farm.go
│   ├── banana_tree.go
│   ├── bunch.go
│   ├── banana.go
│   ├── tool.go
│   └── worker.go
│
├── handlers/
│   ├── farm.go
│   ├── farm_test.go
│   ├── banana_tree.go
│   ├── banana_tree_test.go
│   ├── bunch.go
│   ├── bunch_test.go
│   ├── banana.go
│   ├── banana_test.go
│   ├── tool.go
│   ├── tool_test.go
│   ├── worker.go
│   ├── worker_test.go
│   └── health.go
│
├── routes/
│   └── routes.go               # Chi router, route registration, middleware stack
│
├── middleware/
│   ├── logging.go              # slog request logger (method, path, status, duration)
│   ├── logging_test.go
│   └── error.go                # NotFound, MethodNotAllowed handlers
│
└── helpers/
    ├── response.go             # respondJSON(), respondError() helpers
    ├── pagination.go           # ParsePagination from query params
    └── pagination_test.go
```

---

## API Routes

### Health
```
GET    /health                          # DB status, app env, uptime
```

### Farms
```
GET    /farms                           # List (paginated, filterable by location)
POST   /farms                           # Create
GET    /farms/{id}                      # Get by ID
PUT    /farms/{id}                      # Update
DELETE /farms/{id}                      # Delete
GET    /farms/{id}/trees                # List trees for farm
GET    /farms/{id}/workers              # List workers for farm
GET    /farms/{id}/tools                # List tools for farm
GET    /farms/{id}/stats                # Tree count, banana count, worker count, etc.
```

### Banana Trees
```
GET    /trees                           # List (paginated, filterable by status, variety)
POST   /trees                           # Create
GET    /trees/{id}                      # Get by ID
PUT    /trees/{id}                      # Update
DELETE /trees/{id}                      # Delete
GET    /trees/{id}/bunches              # List bunches for tree
```

### Bunches
```
GET    /bunches                         # List (paginated)
POST   /bunches                         # Create
GET    /bunches/{id}                    # Get by ID
PUT    /bunches/{id}                    # Update
DELETE /bunches/{id}                    # Delete
GET    /bunches/{id}/bananas            # List bananas for bunch
```

### Bananas
```
GET    /bananas                         # List (paginated, filterable by ripeness, size)
POST   /bananas                         # Create
GET    /bananas/{id}                    # Get by ID
PUT    /bananas/{id}                    # Update
DELETE /bananas/{id}                    # Delete
```

### Tools
```
GET    /tools                           # List (paginated, filterable by type, condition)
POST   /tools                           # Create
GET    /tools/{id}                      # Get by ID
PUT    /tools/{id}                      # Update
DELETE /tools/{id}                      # Delete
```

### Workers
```
GET    /workers                         # List (paginated, filterable by role)
POST   /workers                         # Create
GET    /workers/{id}                    # Get by ID
PUT    /workers/{id}                    # Update
DELETE /workers/{id}                    # Delete
```

---

## Middleware Stack (order matters)

```go
r.Use(middleware.RequestID)              // Unique ID per request
r.Use(middleware.Recoverer)              // Catch panics
r.Use(httprate.LimitByIP(100, 1*time.Minute))  // Rate limit
r.Use(cors.Handler(corsOptions))         // CORS
r.Use(customSlogMiddleware)              // Request logging with slog

r.NotFound(notFoundHandler)              // JSON 404
r.MethodNotAllowed(methodNotAllowedHandler)  // JSON 405
```

---

## Configuration

### .env file
```env
APP_ENV=development          # development | production | test
APP_PORT=8080
LOG_LEVEL=debug              # debug | info | warn | error
DB_PATH=./banana_farm.db
CORS_ALLOWED_ORIGINS=http://localhost:3000
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m
```

### Config struct (loaded via caarlos0/env)
```go
type Config struct {
    AppEnv             string        `env:"APP_ENV" envDefault:"development"`
    Port               int           `env:"APP_PORT" envDefault:"8080"`
    LogLevel           string        `env:"LOG_LEVEL" envDefault:"debug"`
    DBPath             string        `env:"DB_PATH" envDefault:"./banana_farm.db"`
    CORSAllowedOrigins []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"*"`
    RateLimitMax       int           `env:"RATE_LIMIT_MAX" envDefault:"100"`
    RateLimitWindow    time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
}
```

---

## Logging (slog)

### Development (APP_ENV=development)
- Handler: `tint.NewHandler` (colored text output to stdout)
- Level: from `LOG_LEVEL` env var (default: debug)
- Format: `2024-01-15 10:30:00 INFO server started port=8080 env=development`

### Production (APP_ENV=production)
- Handler: `slog.JSONHandler` writing to `lumberjack` (file rotation) + stdout
- Level: from `LOG_LEVEL` env var (default: info)
- Rotation: 10MB max, 10 files, 30 day retention
- Format: `{"time":"...","level":"INFO","msg":"server started","port":8080}`

### Test (APP_ENV=test)
- Handler: `slog.NewTextHandler(io.Discard, nil)` (silent)

### Request logging middleware output
```
INFO  request completed method=GET path=/farms status=200 duration=1.2ms request_id=abc123
WARN  request completed method=POST path=/farms status=400 duration=0.8ms request_id=def456
ERROR request completed method=GET path=/trees/999 status=500 duration=2.1ms request_id=ghi789
```

---

## Server Lifecycle (main.go)

1. Load `.env` file via `godotenv`
2. Parse config struct via `caarlos0/env`
3. Initialize slog based on `APP_ENV`
4. Connect to SQLite via GORM, run auto-migrations
5. Build Chi router with middleware + routes
6. Start HTTP server in a goroutine
7. Log startup: port, env, PID
8. Wait for `SIGINT` or `SIGTERM` via `signal.Notify`
9. Graceful shutdown with 30s timeout via `srv.Shutdown(ctx)`
10. Close DB connection
11. Log shutdown complete

---

## Testing Strategy

### Approach
- Use real SQLite in-memory DB (`file::memory:`) - no mocking the DB
- Use `httptest.NewRequest` + `httptest.NewRecorder` for handler tests
- Use `testify/assert` for clean assertions
- Test files live next to source: `farm.go` → `farm_test.go`

### Test helper (shared setup)
```go
// test_helpers.go or helpers_test.go
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
    require.NoError(t, err)
    db.AutoMigrate(&Farm{}, &BananaTree{}, &Bunch{}, &Banana{}, &Tool{}, &Worker{})
    return db
}

func setupTestRouter(db *gorm.DB) chi.Router {
    // build router with all routes, passing test db
    return routes.Setup(db)
}
```

### What to test
1. **Handler tests** - each CRUD operation per entity
   - Create: valid input returns 201 + body, invalid returns 400
   - Read: existing returns 200, missing returns 404
   - Update: valid returns 200, missing returns 404
   - Delete: existing returns 204, missing returns 404
   - List: pagination, filtering, empty results

2. **Model tests** - GORM relationships, cascading deletes

3. **Middleware tests** - not-found returns JSON 404, rate limiter triggers 429

4. **Helper tests** - pagination parsing, response formatting

### Run tests
```bash
go test ./...              # run all tests
go test ./handlers/...     # run handler tests only
go test -v ./...           # verbose output
go test -cover ./...       # with coverage
```

---

## Build Order (step by step)

### Phase 1: Foundation
1. `go mod init github.com/justincordova/banana-farm-api`
2. Install all dependencies
3. Create `.env` + `.env.example`
4. `config/config.go` - config struct + env loading
5. `config/logger.go` - slog setup with tint (dev) / JSON (prod)
6. `database/database.go` - GORM + SQLite connection + migrations
7. `main.go` - server startup + graceful shutdown

### Phase 2: First Entity (Farm)
8. `models/farm.go` - Farm struct with GORM tags
9. `helpers/response.go` - respondJSON + respondError helpers
10. `helpers/pagination.go` - pagination parsing
11. `handlers/farm.go` - full CRUD handlers
12. `routes/routes.go` - Chi router + middleware stack
13. `middleware/logging.go` - slog request logger
14. `middleware/error.go` - NotFound + MethodNotAllowed
15. Wire everything in main.go and test manually

### Phase 3: Remaining Entities
16. `models/banana_tree.go` + `handlers/banana_tree.go` (with lifecycle status)
17. `models/bunch.go` + `handlers/bunch.go`
18. `models/banana.go` + `handlers/banana.go` (with hand_number)
19. `models/tool.go` + `handlers/tool.go`
20. `models/worker.go` + `handlers/worker.go`
21. Nested routes (`/farms/{id}/trees`, etc.)

### Phase 4: Features
22. `handlers/health.go` - health check endpoint
23. Filtering on list endpoints (query params)
24. `GET /farms/{id}/stats` - stats endpoint

### Phase 5: Tests
25. Test helpers (setupTestDB, setupTestRouter)
26. Handler tests for each entity
27. Middleware tests
28. Pagination helper tests
