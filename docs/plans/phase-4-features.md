# Phase 4: Features

Add the health check endpoint, stats endpoint, and ensure all filtering and pagination is working across every entity.

By the end of this phase, your API has observability (health check), analytics (stats), and clean query capabilities.

---

## Step 29: Create `internal/handlers/health.go`

```go
package handlers

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"gorm.io/gorm"

	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/helpers"
)

// HealthHandler holds dependencies for the health check endpoint.
type HealthHandler struct {
	DB        *gorm.DB
	Cfg       *config.Config
	StartTime time.Time
}

// NewHealthHandler creates a new HealthHandler. Pass time.Now() as startTime from cmd/api/main.go.
func NewHealthHandler(db *gorm.DB, cfg *config.Config, startTime time.Time) *HealthHandler {
	return &HealthHandler{
		DB:        db,
		Cfg:       cfg,
		StartTime: startTime,
	}
}

// HealthResponse is the JSON response for the health check endpoint.
type HealthResponse struct {
	Status    string `json:"status"`    // "healthy" or "unhealthy"
	Env       string `json:"env"`       // development, production, test
	Uptime    string `json:"uptime"`    // human-readable uptime
	GoVersion string `json:"go_version"`
	PID       int    `json:"pid"`
	DB        DBHealth `json:"db"`
}

// DBHealth reports the database connection status.
type DBHealth struct {
	Status      string `json:"status"`       // "connected" or "disconnected"
	OpenConns   int    `json:"open_connections"`
	InUse       int    `json:"in_use"`
	Idle        int    `json:"idle"`
}

// Check handles GET /health
//
// Returns 200 if everything is healthy, 503 if the database is unreachable.
// This endpoint is useful for:
// - Load balancers to check if the server is alive
// - Monitoring tools (Uptime Robot, Datadog, etc.)
// - Your own sanity during development
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbHealth := DBHealth{Status: "connected"}
	sqlDB, err := h.DB.DB()
	if err != nil {
		dbHealth.Status = "disconnected"
	} else {
		// Ping the database to verify the connection is alive
		if err := sqlDB.Ping(); err != nil {
			dbHealth.Status = "disconnected"
		} else {
			stats := sqlDB.Stats()
			dbHealth.OpenConns = stats.OpenConnections
			dbHealth.InUse = stats.InUse
			dbHealth.Idle = stats.Idle
		}
	}

	// Determine overall health
	status := "healthy"
	httpStatus := http.StatusOK
	if dbHealth.Status != "connected" {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable // 503
	}

	response := HealthResponse{
		Status:    status,
		Env:       h.Cfg.AppEnv,
		Uptime:    time.Since(h.StartTime).Round(time.Second).String(),
		GoVersion: runtime.Version(),
		PID:       os.Getpid(),
		DB:        dbHealth,
	}

	helpers.RespondJSON(w, httpStatus, response)
}
```

### What's happening here:
- **`sqlDB.Ping()`** sends a real query to the database to verify the connection works. Without this, you might report "connected" even if the DB file was deleted or corrupted
- **`sqlDB.Stats()`** gives you connection pool metrics — useful for debugging connection leaks in production
- **503 Service Unavailable** — the correct HTTP status for "server is alive but can't serve requests". Load balancers use this to route traffic away
- **`runtime.Version()`** returns `go1.22.0` or similar — handy for verifying deployments
- **`StartTime`** is passed from `cmd/api/main.go` so we can calculate uptime

### Example response:
```json
{
  "status": "healthy",
  "env": "development",
  "uptime": "2h15m30s",
  "go_version": "go1.22.0",
  "pid": 12345,
  "db": {
    "status": "connected",
    "open_connections": 1,
    "in_use": 0,
    "idle": 1
  }
}
```

---

## Step 30: Add stats endpoint to Farm handler

Add this method to `internal/handlers/farm.go`:

**Note:** The `internal/handlers/farm.go` file already imports `errors`, `log/slog`, `chi`, `gorm`, and `helpers` from Phase 2 (we added `"errors"` in that phase), so you don't need to add any imports for this method.

```go
// FarmStats is the response for GET /farms/{id}/stats
type FarmStats struct {
	FarmID      uint   `json:"farm_id"`
	FarmName    string `json:"farm_name"`
	TreeCount   int64  `json:"tree_count"`
	BunchCount  int64  `json:"bunch_count"`
	BananaCount int64  `json:"banana_count"`
	WorkerCount int64  `json:"worker_count"`
	ToolCount   int64  `json:"tool_count"`

	// Breakdown by tree status
	TreesByStatus map[string]int64 `json:"trees_by_status"`

	// Breakdown by worker role
	WorkersByRole map[string]int64 `json:"workers_by_role"`
}

// Stats handles GET /farms/{id}/stats
// Returns aggregate counts and breakdowns for a farm.
func (h *FarmHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify farm exists
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

	stats := FarmStats{
		FarmID:   farm.ID,
		FarmName: farm.Name,
	}

	// NOTE: The following makes 7 separate DB queries sequentially.
	// This is fine for a learning project and low traffic. In a high-traffic
	// production API you'd combine these into a single SQL query or use caching.

	// Count trees
	h.DB.Model(&models.BananaTree{}).Where("farm_id = ?", id).Count(&stats.TreeCount)

	// Count bunches (bunches belong to trees that belong to this farm)
	h.DB.Model(&models.Bunch{}).
		Joins("JOIN banana_trees ON banana_trees.id = bunches.banana_tree_id").
		Where("banana_trees.farm_id = ?", id).
		Count(&stats.BunchCount)

	// Count bananas (bananas belong to bunches that belong to trees that belong to this farm)
	h.DB.Model(&models.Banana{}).
		Joins("JOIN bunches ON bunches.id = bananas.bunch_id").
		Joins("JOIN banana_trees ON banana_trees.id = bunches.banana_tree_id").
		Where("banana_trees.farm_id = ?", id).
		Count(&stats.BananaCount)

	// Count workers and tools
	h.DB.Model(&models.Worker{}).Where("farm_id = ?", id).Count(&stats.WorkerCount)
	h.DB.Model(&models.Tool{}).Where("farm_id = ?", id).Count(&stats.ToolCount)

	// Trees by status breakdown
	stats.TreesByStatus = make(map[string]int64)
	var treeStatusCounts []struct {
		Status string
		Count  int64
	}
	h.DB.Model(&models.BananaTree{}).
		Select("status, COUNT(*) as count").
		Where("farm_id = ?", id).
		Group("status").
		Find(&treeStatusCounts)

	for _, sc := range treeStatusCounts {
		stats.TreesByStatus[sc.Status] = sc.Count
	}

	// Workers by role breakdown
	stats.WorkersByRole = make(map[string]int64)
	var workerRoleCounts []struct {
		Role  string
		Count int64
	}
	h.DB.Model(&models.Worker{}).
		Select("role, COUNT(*) as count").
		Where("farm_id = ?", id).
		Group("role").
		Find(&workerRoleCounts)

	for _, rc := range workerRoleCounts {
		stats.WorkersByRole[rc.Role] = rc.Count
	}

	helpers.RespondJSON(w, http.StatusOK, stats)
}
```

### What's happening here:

**Simple counts:**
```go
h.DB.Model(&models.BananaTree{}).Where("farm_id = ?", id).Count(&stats.TreeCount)
```
GORM translates this to: `SELECT COUNT(*) FROM banana_trees WHERE farm_id = ?`

**Counts through relationships (JOINs):**
```go
h.DB.Model(&models.Banana{}).
    Joins("JOIN bunches ON bunches.id = bananas.bunch_id").
    Joins("JOIN banana_trees ON banana_trees.id = bunches.banana_tree_id").
    Where("banana_trees.farm_id = ?", id).
    Count(&stats.BananaCount)
```
This counts all bananas that belong to bunches that belong to trees on this farm. It's a two-level JOIN. GORM's `.Joins()` lets you write raw SQL JOINs when `.Preload()` isn't enough.

**GROUP BY for breakdowns:**
```go
h.DB.Model(&models.BananaTree{}).
    Select("status, COUNT(*) as count").
    Where("farm_id = ?", id).
    Group("status").
    Find(&treeStatusCounts)
```
This is `SELECT status, COUNT(*) FROM banana_trees WHERE farm_id = ? GROUP BY status`. The result is scanned into an anonymous struct slice, then converted to a map for clean JSON output.

### Example response:
```json
{
  "farm_id": 1,
  "farm_name": "Cordova Plantation",
  "tree_count": 150,
  "bunch_count": 45,
  "banana_count": 2700,
  "worker_count": 12,
  "tool_count": 25,
  "trees_by_status": {
    "planted": 20,
    "growing": 50,
    "flowering": 30,
    "fruiting": 40,
    "harvested": 8,
    "dead": 2
  },
  "workers_by_role": {
    "farmer": 5,
    "harvester": 4,
    "irrigator": 2,
    "supervisor": 1
  }
}
```

---

## Step 31: Update `internal/routes/routes.go` with new endpoints

Add the health route and uncomment the stats route:

```go
func Setup(cfg *config.Config, db *gorm.DB, startTime time.Time) *chi.Mux {
	r := chi.NewRouter()

	// ... middleware ...

	// Health check — no middleware group, always accessible
	healthHandler := handlers.NewHealthHandler(db, cfg, startTime)
	r.Get("/health", healthHandler.Check)

	// Farm routes
	farmHandler := handlers.NewFarmHandler(db)
	r.Route("/farms", func(r chi.Router) {
		r.Get("/", farmHandler.List)
		r.Post("/", farmHandler.Create)
		r.Get("/{id}", farmHandler.Get)
		r.Put("/{id}", farmHandler.Update)
		r.Delete("/{id}", farmHandler.Delete)
		r.Get("/{id}/trees", farmHandler.ListTrees)
		r.Get("/{id}/workers", farmHandler.ListWorkers)
		r.Get("/{id}/tools", farmHandler.ListTools)
		r.Get("/{id}/stats", farmHandler.Stats) // NEW
	})

	// ... rest of routes unchanged ...
```

### Update `cmd/api/main.go` to pass startTime:

Add `startTime` before the server setup:

```go
func main() {
	startTime := time.Now()

	// ... config, logger, db setup ...

	// 5. Build router
	router := routes.Setup(cfg, db, startTime) // pass startTime

	// ... rest unchanged ...
}
```

Don't forget to add `"time"` to the imports in `cmd/api/main.go` and update the `routes.Setup` function signature.

---

## Step 32: Verify all filtering works

Here's a complete list of every filter parameter on every list endpoint. Test each one:

### Farm filters
```bash
# Filter by location (LIKE search — partial match)
curl "http://localhost:8080/farms?location=hawaii"
```

### BananaTree filters
```bash
# Filter by lifecycle status
curl "http://localhost:8080/trees?status=flowering"

# Filter by variety
curl "http://localhost:8080/trees?variety=cavendish"

# Filter by farm
curl "http://localhost:8080/trees?farm_id=1"

# Combine filters
curl "http://localhost:8080/trees?status=fruiting&variety=plantain"
```

### Banana filters
```bash
# Filter by ripeness
curl "http://localhost:8080/bananas?ripeness=green"

# Filter by size
curl "http://localhost:8080/bananas?size=large"

# Filter by bunch
curl "http://localhost:8080/bananas?bunch_id=1"
```

### Tool filters
```bash
# Filter by type
curl "http://localhost:8080/tools?type=machete"

# Filter by condition
curl "http://localhost:8080/tools?condition=worn"

# Filter by farm
curl "http://localhost:8080/tools?farm_id=1"
```

### Worker filters
```bash
# Filter by role
curl "http://localhost:8080/workers?role=harvester"

# Filter by farm
curl "http://localhost:8080/workers?farm_id=1"
```

### Bunch filters
```bash
# Filter by tree
curl "http://localhost:8080/bunches?banana_tree_id=1"
```

### Pagination on any endpoint
```bash
curl "http://localhost:8080/farms?page=1&limit=5"
curl "http://localhost:8080/trees?page=2&limit=10&status=growing"
```

### Nested routes with filters
```bash
# Trees on a farm, filtered by status
curl "http://localhost:8080/farms/1/trees?status=flowering"

# Workers on a farm, filtered by role
curl "http://localhost:8080/farms/1/workers?role=harvester"

# Tools on a farm, filtered by type
curl "http://localhost:8080/farms/1/tools?type=machete"
```

---

## Verify Phase 4 is complete

### Health check:
```bash
curl http://localhost:8080/health
# → 200 with status, uptime, db info
```

### Stats:
```bash
# First seed some data (create farm, trees, bunches, bananas, workers, tools)
# Then:
curl http://localhost:8080/farms/1/stats
# → 200 with all counts and breakdowns
```

---

## Phase 4 Checklist

- [ ] `internal/handlers/health.go` with DB ping, uptime, connection stats
- [ ] `GET /health` returns 200 when healthy, 503 when DB is down
- [ ] `internal/handlers/farm.go` has Stats method with JOIN queries
- [ ] `GET /farms/{id}/stats` returns counts and breakdowns
- [ ] `internal/routes/routes.go` updated with health and stats routes
- [ ] `cmd/api/main.go` passes `startTime` to routes
- [ ] All list endpoints support `?page=&limit=` pagination
- [ ] All filter parameters work as documented
- [ ] Nested route filters work (`/farms/1/trees?status=flowering`)
- [ ] Combined filters work (`/trees?status=fruiting&variety=plantain`)
