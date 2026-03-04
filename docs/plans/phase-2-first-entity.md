# Phase 2: First Entity (Farm)

Build the full CRUD pipeline for the Farm entity end-to-end. This establishes every pattern you'll reuse for all other entities: model, handlers, response helpers, pagination, router, and middleware.

By the end of this phase you'll have a fully working Farm CRUD API with request logging, error handling, and pagination.

---

## Step 8: Create `internal/models/farm.go`

Create the `internal/models/` directory and the Farm model.

```go
package models

import "time"

// Farm represents a banana farm.
type Farm struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Location    string    `json:"location" gorm:"not null" validate:"required,min=1,max=200"`
	SizeAcres   float64   `json:"size_acres" gorm:"not null" validate:"required,gt=0"`
	Established time.Time `json:"established" gorm:"not null" validate:"required"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships — loaded via Preload, not by default
	BananaTrees []BananaTree `json:"banana_trees,omitempty" gorm:"foreignKey:FarmID"`
	Workers     []Worker     `json:"workers,omitempty" gorm:"foreignKey:FarmID"`
	Tools       []Tool       `json:"tools,omitempty" gorm:"foreignKey:FarmID"`
}
```

### What's happening here:
- **Struct tags** serve three purposes:
  - `json:"name"` — controls JSON serialization (snake_case for the API)
  - `gorm:"not null"` — database column constraints
  - `validate:"required,min=1"` — validation rules (used in handlers)
- **`ID`, `CreatedAt`, `UpdatedAt`** — GORM auto-manages these. `ID` auto-increments, timestamps auto-populate
- **Relationships** — `BananaTrees`, `Workers`, `Tools` won't be loaded by default. You explicitly load them with `.Preload("BananaTrees")` when needed
- **`omitempty`** — if relationships aren't loaded, they'll be `nil` and omitted from JSON (instead of showing `"banana_trees": null`)
- **`BananaTree`, `Worker`, `Tool`** don't exist yet — the compiler will complain. That's fine; we'll add stub types or comment these out until Phase 3

**Temporary fix** — until Phase 3, comment out the relationship fields or create stub models:

```go
// internal/models/stubs.go — temporary, delete in Phase 3
package models

type BananaTree struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	FarmID uint `json:"farm_id"`
}

type Bunch struct {
	ID           uint `json:"id" gorm:"primaryKey"`
	BananaTreeID uint `json:"banana_tree_id"`
}

type Banana struct {
	ID      uint `json:"id" gorm:"primaryKey"`
	BunchID uint `json:"bunch_id"`
}

type Worker struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	FarmID uint `json:"farm_id"`
}

type Tool struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	FarmID uint `json:"farm_id"`
}
```

### Update `internal/database/database.go` to include the Farm model:

Uncomment the models in the `Migrate` function:

```go
import "github.com/justincordova/banana-farm-api/internal/models"

func Migrate(db *gorm.DB) error {
	slog.Info("running database migrations")

	err := db.AutoMigrate(
		&models.Farm{},
		&models.BananaTree{},
		&models.Bunch{},
		&models.Banana{},
		&models.Worker{},
		&models.Tool{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations completed")
	return nil
}
```

---

## Step 9: Create `internal/helpers/response.go`

Create the `internal/helpers/` directory. These are reusable functions every handler will use.

```go
package helpers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// RespondJSON writes a JSON response with the given status code and payload.
// Every successful response goes through this function for consistency.
func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// ErrorResponse is the standard error response body.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// RespondError writes a JSON error response with the given status code and message.
// Every error response goes through this function for consistency.
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorResponse{Error: message})
}

// RespondErrorWithDetails writes a JSON error response with additional detail (e.g., validation errors).
func RespondErrorWithDetails(w http.ResponseWriter, status int, message string, details string) {
	RespondJSON(w, status, ErrorResponse{Error: message, Details: details})
}

// DecodeJSON reads the request body and decodes it into the given destination struct.
// Returns an error if the body is malformed or empty.
func DecodeJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}
```

### What's happening here:
- **`RespondJSON`** — every response goes through this. Sets `Content-Type`, writes status code, encodes payload. If encoding somehow fails, it logs the error (rare edge case)
- **`ErrorResponse`** — consistent error format: `{"error": "message", "details": "optional"}`
- **`DecodeJSON`** — wraps `json.NewDecoder` with `DisallowUnknownFields()` so if someone sends `{"nme": "test"}` (typo), they'll get an error instead of a silent ignore
- This is the Go equivalent of Express's `res.json()` and `res.status().json()`

---

## Step 10: Create `internal/helpers/pagination.go`

```go
package helpers

import (
	"net/http"
	"strconv"
)

// Pagination holds the parsed pagination parameters from query strings.
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// PaginatedResponse wraps a list response with pagination metadata.
type PaginatedResponse struct {
	Data       any   `json:"data"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// ParsePagination extracts page and limit from query parameters.
// Defaults: page=1, limit=20. Max limit=100.
//
// Usage:
//
//	GET /farms?page=2&limit=10
func ParsePagination(r *http.Request) Pagination {
	page := parseIntParam(r, "page", DefaultPage)
	limit := parseIntParam(r, "limit", DefaultLimit)

	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return Pagination{Page: page, Limit: limit}
}

// Offset calculates the SQL OFFSET value from page and limit.
// Example: page=2, limit=20 → offset=20 (skip first 20 rows)
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// NewPaginatedResponse creates a PaginatedResponse with calculated total pages.
func NewPaginatedResponse(data any, total int64, pagination Pagination) PaginatedResponse {
	totalPages := int(total) / pagination.Limit
	if int(total)%pagination.Limit != 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data:       data,
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// parseIntParam reads a query parameter and converts it to an int.
// Returns the default value if the parameter is missing or invalid.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}

	num, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return num
}
```

### What's happening here:
- **`ParsePagination`** reads `?page=X&limit=Y` from the URL. Clamps values to sane ranges
- **`Offset()`** converts page-based pagination to SQL OFFSET: `page 2, limit 20 → OFFSET 20`
- **`NewPaginatedResponse`** wraps your data with metadata. The API response looks like:
  ```json
  {
    "data": [...],
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
  ```
- **`MaxLimit = 100`** prevents clients from requesting `?limit=999999` and dumping your entire database

---

## Step 11: Create `internal/handlers/farm.go`

Create the `internal/handlers/` directory.

```go
package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/internal/helpers"
	"github.com/justincordova/banana-farm-api/internal/models"
)

// FarmHandler holds dependencies for farm-related HTTP handlers.
// This is the Go equivalent of a controller class — it groups handlers and their shared state.
type FarmHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

// NewFarmHandler creates a new FarmHandler with the given dependencies.
//
// Note: Each handler creates its own validator.Validate instance here for simplicity.
// In a larger application, you'd create one shared instance in main.go and pass it in,
// which allows registering custom validators in one place.
func NewFarmHandler(db *gorm.DB) *FarmHandler {
	return &FarmHandler{
		DB:       db,
		Validate: validator.New(),
	}
}

// CreateFarmRequest is the expected JSON body for creating a farm.
// Separate from the model so you control exactly what the client can send.
type CreateFarmRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Location    string  `json:"location" validate:"required,min=1,max=200"`
	SizeAcres   float64 `json:"size_acres" validate:"required,gt=0"`
	Established string  `json:"established" validate:"required"`
}

// UpdateFarmRequest is the expected JSON body for updating a farm.
// All fields are optional (pointers) — only non-nil fields are updated.
//
// Note: Established is intentionally omitted — farm founding dates don't change.
// If you need to support correcting a wrong date, add it back as *string here.
type UpdateFarmRequest struct {
	Name      *string  `json:"name" validate:"omitempty,min=1,max=100"`
	Location  *string  `json:"location" validate:"omitempty,min=1,max=200"`
	SizeAcres *float64 `json:"size_acres" validate:"omitempty,gt=0"`
}

// List handles GET /farms
// Supports pagination: ?page=1&limit=20
// Supports filtering: ?location=hawaii
func (h *FarmHandler) List(w http.ResponseWriter, r *http.Request) {
	pagination := helpers.ParsePagination(r)

	var farms []models.Farm
	var total int64

	query := h.DB.Model(&models.Farm{})

	// Filter by location if provided
	if location := r.URL.Query().Get("location"); location != "" {
		query = query.Where("location LIKE ?", "%"+location+"%")
	}

	// Get total count (before pagination)
	query.Count(&total)

	// Apply pagination and fetch
	result := query.Offset(pagination.Offset()).Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&farms)

	if result.Error != nil {
		slog.Error("failed to list farms", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to list farms")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, helpers.NewPaginatedResponse(farms, total, pagination))
}

// Create handles POST /farms
func (h *FarmHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateFarmRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Parse the established date string
	established, err := ParseDate(req.Established)
	if err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid date format for established, use YYYY-MM-DD")
		return
	}

	farm := models.Farm{
		Name:        req.Name,
		Location:    req.Location,
		SizeAcres:   req.SizeAcres,
		Established: established,
	}

	if result := h.DB.Create(&farm); result.Error != nil {
		slog.Error("failed to create farm", "error", result.Error)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to create farm")
		return
	}

	slog.Info("farm created", "id", farm.ID, "name", farm.Name)
	helpers.RespondJSON(w, http.StatusCreated, farm)
}

// Get handles GET /farms/{id}
func (h *FarmHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var farm models.Farm
	result := h.DB.First(&farm, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "farm not found")
			return
		}
		slog.Error("failed to get farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to get farm")
		return
	}

	helpers.RespondJSON(w, http.StatusOK, farm)
}

// Update handles PUT /farms/{id}
func (h *FarmHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Find existing farm
	var farm models.Farm
	if result := h.DB.First(&farm, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find farm")
		return
	}

	// Decode and validate request
	var req UpdateFarmRequest
	if err := helpers.DecodeJSON(r, &req); err != nil {
		helpers.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	// Apply only the fields that were sent (non-nil pointers)
	if req.Name != nil {
		farm.Name = *req.Name
	}
	if req.Location != nil {
		farm.Location = *req.Location
	}
	if req.SizeAcres != nil {
		farm.SizeAcres = *req.SizeAcres
	}

	if result := h.DB.Save(&farm); result.Error != nil {
		slog.Error("failed to update farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to update farm")
		return
	}

	slog.Info("farm updated", "id", farm.ID)
	helpers.RespondJSON(w, http.StatusOK, farm)
}

// Delete handles DELETE /farms/{id}
func (h *FarmHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if farm exists
	var farm models.Farm
	if result := h.DB.First(&farm, id); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			helpers.RespondError(w, http.StatusNotFound, "farm not found")
			return
		}
		slog.Error("failed to find farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to find farm")
		return
	}

	if result := h.DB.Delete(&farm); result.Error != nil {
		slog.Error("failed to delete farm", "error", result.Error, "id", id)
		helpers.RespondError(w, http.StatusInternalServerError, "failed to delete farm")
		return
	}

	slog.Info("farm deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
```

### Create `internal/handlers/helpers.go` for shared handler utilities:

```go
package handlers

import "time"

// ParseDate parses a date string in "YYYY-MM-DD" format.
// Exported so it can be tested from the handlers_test package.
func ParseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
```

### Key patterns to understand:

**Handler struct pattern:**
```go
type FarmHandler struct {
    DB       *gorm.DB
    Validate *validator.Validate
}
```
This is dependency injection in Go. Instead of global variables, the handler holds its dependencies. In Express terms, this is like passing `req.app.get('db')` — but type-safe and explicit.

**Request/Response separation:**
- `CreateFarmRequest` is what the client sends (input)
- `models.Farm` is what the database stores and the API returns (output)
- This separation means clients can't set `id`, `created_at`, etc.

**Pointer fields for partial updates:**
```go
type UpdateFarmRequest struct {
    Name *string `json:"name"`
}
```
If the client sends `{"name": "New Name"}`, then `req.Name` is `*string` pointing to `"New Name"`. If the client omits `name`, then `req.Name` is `nil`. This lets you distinguish between "not sent" and "sent as empty string".

**Error handling pattern:**
Every GORM operation checks for errors. If it's a `gorm.ErrRecordNotFound`, return 404. For everything else, log the actual error (for debugging) but return a generic message to the client (don't leak internals).

---

## Step 12: Create `internal/routes/routes.go`

Create the `internal/routes/` directory.

```go
package routes

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/handlers"
	"github.com/justincordova/banana-farm-api/internal/middleware"
)

// Setup creates and configures the Chi router with all middleware and routes.
// Returns the router ready to be used as an http.Handler.
func Setup(cfg *config.Config, db *gorm.DB) *chi.Mux {
	r := chi.NewRouter()

	// --- Middleware stack (order matters!) ---

	// RequestID injects a unique ID into every request's context.
	// Access it in handlers/middleware via middleware.GetReqID(r.Context())
	r.Use(chimiddleware.RequestID)

	// Recoverer catches panics and returns a 500 instead of crashing the server.
	// This is your safety net — equivalent to Express uncaughtException handler.
	r.Use(chimiddleware.Recoverer)

	// Rate limiter — 100 requests per minute per IP address.
	r.Use(httprate.LimitByIP(cfg.RateLimitMax, cfg.RateLimitWindow))

	// CORS — controls which origins can call your API.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // seconds — how long browsers cache preflight responses
	}))

	// Custom slog request logger — logs every request with method, path, status, duration.
	r.Use(middleware.Logger)

	// Custom error handlers for routes that don't match
	r.NotFound(middleware.NotFoundHandler)
	r.MethodNotAllowed(middleware.MethodNotAllowedHandler)

	// --- Routes ---

	// Farm routes
	farmHandler := handlers.NewFarmHandler(db)

	r.Route("/farms", func(r chi.Router) {
		r.Get("/", farmHandler.List)       // GET /farms
		r.Post("/", farmHandler.Create)    // POST /farms
		r.Get("/{id}", farmHandler.Get)    // GET /farms/{id}
		r.Put("/{id}", farmHandler.Update) // PUT /farms/{id}
		r.Delete("/{id}", farmHandler.Delete) // DELETE /farms/{id}

		// Nested routes will be added in Phase 3:
		// r.Get("/{id}/trees", farmHandler.ListTrees)
		// r.Get("/{id}/workers", farmHandler.ListWorkers)
		// r.Get("/{id}/tools", farmHandler.ListTools)
		// r.Get("/{id}/stats", farmHandler.Stats)
	})

	return r
}
```

### What's happening here:
- **`chi.NewRouter()`** creates the router. It implements `http.Handler` so it plugs directly into `http.Server`
- **Middleware order matters:**
  1. `RequestID` — first, so all downstream middleware/handlers have access to the ID
  2. `Recoverer` — early, so it catches panics from any middleware below it
  3. Rate limiter — before any business logic, reject excess requests early
  4. CORS — must run before handlers to handle preflight OPTIONS requests
  5. Logger — last, so it can capture the final status code and duration
- **`r.Route("/farms", ...)`** groups routes under a prefix. Inside the function, paths are relative: `"/"` means `/farms`, `"/{id}"` means `/farms/{id}`
- **`{id}`** is a URL parameter. Access it in handlers with `chi.URLParam(r, "id")`

---

## Step 13: Create `internal/middleware/logging.go`

Create the `internal/middleware/` directory.

```go
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
// Go's http.ResponseWriter doesn't expose the status code after WriteHeader is called,
// so we need this wrapper to capture it for logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the underlying ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger is a middleware that logs every HTTP request using slog.
// It captures: method, path, status code, duration, and request ID.
//
// Log level is based on status code:
//   - 5xx → ERROR
//   - 4xx → WARN
//   - 2xx/3xx → INFO
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Build log attributes
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", duration.String(),
			"request_id", chimiddleware.GetReqID(r.Context()),
		}

		// Log at appropriate level based on status code
		switch {
		case wrapped.statusCode >= 500:
			slog.Error("request completed", attrs...)
		case wrapped.statusCode >= 400:
			slog.Warn("request completed", attrs...)
		default:
			slog.Info("request completed", attrs...)
		}
	})
}
```

### What's happening here:
- **`responseWriter` wrapper** — Go's `http.ResponseWriter` interface doesn't let you read the status code after it's been written. This is a common Go pattern: wrap the writer, intercept `WriteHeader`, and store the code
- **`statusCode: http.StatusOK`** — default to 200. If a handler calls `w.Write()` without calling `w.WriteHeader()` first, Go automatically sends 200
- **Log level by status** — errors (5xx) are `slog.Error`, client errors (4xx) are `slog.Warn`, success (2xx/3xx) are `slog.Info`. This makes it easy to filter logs in production
- **`chimiddleware.GetReqID`** reads the request ID that was injected by the `RequestID` middleware earlier in the chain

---

## Step 14: Create `internal/middleware/error.go`

```go
package middleware

import (
	"net/http"

	"github.com/justincordova/banana-farm-api/internal/helpers"
)

// NotFoundHandler returns a JSON 404 response for unmatched routes.
// Without this, Chi returns a plain text "404 page not found".
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	helpers.RespondError(w, http.StatusNotFound, "route not found")
}

// MethodNotAllowedHandler returns a JSON 405 response when the HTTP method
// is not supported for a matched route.
// Example: sending DELETE to a route that only supports GET and POST.
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	helpers.RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
}
```

### What's happening here:
- Without these, Chi returns plain text errors which break JSON API clients
- Every error response from your API is now consistent JSON: `{"error": "message"}`
- This is the Go equivalent of Express's catch-all error middleware

---

## Step 15: Wire everything in `cmd/api/main.go`

Update `cmd/api/main.go` to use the Chi router instead of the placeholder `http.NewServeMux()`.

### Changes to make:

**Add import:**
```go
"github.com/justincordova/banana-farm-api/internal/routes"
```

**Replace the placeholder router section with:**
```go
	// 5. Build router
	router := routes.Setup(cfg, db)

	// 6. Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}
```

**Delete these lines** (the old placeholder):
```go
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Banana Farm API"}`))
	})
```

---

## Verify Phase 2 is complete

Start the server:

```bash
air
# or: go run ./cmd/api
```

### Test CRUD operations:

**Create a farm:**
```bash
curl -X POST http://localhost:8080/farms \
  -H "Content-Type: application/json" \
  -d '{"name": "Cordova Banana Plantation", "location": "Hawaii", "size_acres": 50.5, "established": "2020-01-15"}'
# → 201 Created + farm JSON with id
```

**List farms:**
```bash
curl http://localhost:8080/farms
# → 200 OK + paginated response

curl "http://localhost:8080/farms?page=1&limit=5"
# → paginated with limit 5

curl "http://localhost:8080/farms?location=hawaii"
# → filtered by location
```

**Get a farm:**
```bash
curl http://localhost:8080/farms/1
# → 200 OK + farm JSON

curl http://localhost:8080/farms/999
# → 404 {"error": "farm not found"}
```

**Update a farm:**
```bash
curl -X PUT http://localhost:8080/farms/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Farm Name"}'
# → 200 OK + updated farm JSON
```

**Delete a farm:**
```bash
curl -X DELETE http://localhost:8080/farms/1
# → 204 No Content
```

**Test error handling:**
```bash
curl http://localhost:8080/nonexistent
# → 404 {"error": "route not found"}

curl -X PATCH http://localhost:8080/farms
# → 405 {"error": "method not allowed"}

curl -X POST http://localhost:8080/farms \
  -H "Content-Type: application/json" \
  -d '{"name": ""}'
# → 400 {"error": "validation failed", "details": "..."}
```

### Check your terminal for colored slog output:
```
2024-01-15 10:30:00 INFO  request completed method=POST path=/farms status=201 duration=2.1ms request_id=abc123
2024-01-15 10:30:01 INFO  request completed method=GET path=/farms status=200 duration=0.5ms request_id=def456
2024-01-15 10:30:02 WARN  request completed method=GET path=/farms/999 status=404 duration=0.3ms request_id=ghi789
```

---

## Phase 2 Checklist

- [ ] `internal/models/farm.go` with GORM tags, JSON tags, and validation tags
- [ ] `internal/models/stubs.go` with temporary stub types (to be replaced in Phase 3)
- [ ] `internal/database/database.go` updated with Farm in AutoMigrate
- [ ] `internal/helpers/response.go` with RespondJSON, RespondError, DecodeJSON
- [ ] `internal/helpers/pagination.go` with ParsePagination, PaginatedResponse
- [ ] `internal/handlers/farm.go` with List, Create, Get, Update, Delete
- [ ] `internal/handlers/helpers.go` with ParseDate utility (exported for testability)
- [ ] `internal/routes/routes.go` with Chi router, full middleware stack, farm routes
- [ ] `internal/middleware/logging.go` with slog request logger
- [ ] `internal/middleware/error.go` with NotFound and MethodNotAllowed handlers
- [ ] `cmd/api/main.go` updated to use `routes.Setup()`
- [ ] All CRUD operations work via curl
- [ ] Colored request logs visible in terminal
- [ ] 404/405 return consistent JSON errors
- [ ] Pagination works with ?page=&limit=
- [ ] Filtering works with ?location=
