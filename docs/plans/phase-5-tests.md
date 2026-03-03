# Phase 5: Tests

Write unit tests for handlers, middleware, and helpers using testify and httptest with an in-memory SQLite database.

By the end of this phase, every CRUD operation, middleware behavior, and helper function is covered by tests.

---

## Step 33: Create test helpers

Create a shared test setup file that every test file will use.

### Create `handlers/test_helpers_test.go`

```go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/models"
	"github.com/justincordova/banana-farm-api/routes"
)

// setupTestDB creates a fresh in-memory SQLite database with all tables.
// Each test gets its own isolated database — no shared state between tests.
// We use a unique DSN per call so concurrent tests don't share the same DB.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper() // marks this as a helper so test failures show the caller's line number

	// Use a unique name per test so each gets a truly isolated in-memory DB.
	// "file::memory:?cache=shared" would share state across tests — avoid it.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign key constraints — required for ON DELETE CASCADE to work.
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)

	err = db.AutoMigrate(
		&models.Farm{},
		&models.BananaTree{},
		&models.Bunch{},
		&models.Banana{},
		&models.Tool{},
		&models.Worker{},
	)
	require.NoError(t, err)

	return db
}

// setupTestRouter creates a Chi router with all routes wired up, pointing at the test DB.
func setupTestRouter(t *testing.T, db *gorm.DB) *chi.Mux {
	t.Helper()

	cfg := &config.Config{
		AppEnv:             "test",
		Port:               8080,
		LogLevel:           "error",
		CORSAllowedOrigins: []string{"*"},
		RateLimitMax:       1000, // high limit so rate limiter doesn't interfere with tests
		RateLimitWindow:    time.Minute,
	}

	return routes.Setup(cfg, db, time.Now())
}

// makeRequest is a helper that executes an HTTP request against the test router and returns the response.
//
// Usage:
//
//	resp := makeRequest(t, router, "POST", "/farms", map[string]any{"name": "Test Farm", ...})
//	assert.Equal(t, 201, resp.Code)
func makeRequest(t *testing.T, router *chi.Mux, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

// parseResponse decodes the JSON response body into the given destination.
func parseResponse(t *testing.T, rr *httptest.ResponseRecorder, dest any) {
	t.Helper()
	err := json.NewDecoder(rr.Body).Decode(dest)
	require.NoError(t, err)
}

// seedFarm creates a farm in the database and returns it.
// Useful for tests that need a farm to exist before testing other entities.
func seedFarm(t *testing.T, db *gorm.DB) models.Farm {
	t.Helper()

	farm := models.Farm{
		Name:        "Test Farm",
		Location:    "Hawaii",
		SizeAcres:   100,
		Established: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	result := db.Create(&farm)
	require.NoError(t, result.Error)

	return farm
}

// seedTree creates a banana tree on the given farm and returns it.
func seedTree(t *testing.T, db *gorm.DB, farmID uint) models.BananaTree {
	t.Helper()

	tree := models.BananaTree{
		FarmID:    farmID,
		Variety:   "cavendish",
		PlantedAt: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		Status:    "planted",
		Health:    "healthy",
	}
	result := db.Create(&tree)
	require.NoError(t, result.Error)

	return tree
}

// seedBunch creates a bunch on the given tree and returns it.
func seedBunch(t *testing.T, db *gorm.DB, treeID uint) models.Bunch {
	t.Helper()

	bunch := models.Bunch{
		BananaTreeID: treeID,
		WeightKg:     20.0,
	}
	result := db.Create(&bunch)
	require.NoError(t, result.Error)

	return bunch
}
```

### What's happening here:

**`t.Helper()`** — tells Go's test runner that this is a helper function. When a test fails inside a helper, the error message points to the line in the actual test file, not the helper. Without this, every failure would point to `test_helpers_test.go` which is useless.

**`file::memory:?cache=shared`** — SQLite in-memory database. It's created fresh for each test, runs entirely in RAM (no disk I/O), and is automatically destroyed when the connection closes. The `cache=shared` parameter allows multiple connections to share the same in-memory database within a single test.

**`_test` package name** — notice the package is `handlers_test`, not `handlers`. This is a Go convention called "black box testing" — tests can only access exported (capitalized) functions. This tests your API the same way external code would use it.

**`makeRequest` helper** — wraps the boilerplate of creating a request, setting headers, and recording the response. Without this, every test would have 5+ lines of setup before the actual assertion.

**Seed functions** — `seedFarm`, `seedTree`, `seedBunch` create test data directly in the DB. Tests that need a farm to exist (e.g., creating a tree) call `seedFarm` first.

---

## Step 34: Write Farm handler tests

### Create `handlers/farm_test.go`

```go
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

func TestCreateFarm(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("valid farm", func(t *testing.T) {
		body := map[string]any{
			"name":        "Cordova Plantation",
			"location":    "Hawaii",
			"size_acres":  50.5,
			"established": "2020-01-15",
		}

		resp := makeRequest(t, router, "POST", "/farms", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var farm models.Farm
		parseResponse(t, resp, &farm)
		assert.Equal(t, "Cordova Plantation", farm.Name)
		assert.Equal(t, "Hawaii", farm.Location)
		assert.Equal(t, 50.5, farm.SizeAcres)
		assert.NotZero(t, farm.ID)
	})

	t.Run("missing required fields", func(t *testing.T) {
		body := map[string]any{
			"name": "Incomplete Farm",
		}

		resp := makeRequest(t, router, "POST", "/farms", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)

		var errResp helpers.ErrorResponse
		parseResponse(t, resp, &errResp)
		assert.Equal(t, "validation failed", errResp.Error)
	})

	t.Run("invalid date format", func(t *testing.T) {
		body := map[string]any{
			"name":        "Bad Date Farm",
			"location":    "Mars",
			"size_acres":  10,
			"established": "not-a-date",
		}

		resp := makeRequest(t, router, "POST", "/farms", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		resp := makeRequest(t, router, "POST", "/farms", "not json")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("negative size", func(t *testing.T) {
		body := map[string]any{
			"name":        "Negative Farm",
			"location":    "Nowhere",
			"size_acres":  -5,
			"established": "2020-01-01",
		}

		resp := makeRequest(t, router, "POST", "/farms", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestGetFarm(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("existing farm", func(t *testing.T) {
		farm := seedFarm(t, db)

		resp := makeRequest(t, router, "GET", fmt.Sprintf("/farms/%d", farm.ID), nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var got models.Farm
		parseResponse(t, resp, &got)
		assert.Equal(t, farm.Name, got.Name)
	})

	t.Run("non-existent farm", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/farms/999", nil)
		assert.Equal(t, http.StatusNotFound, resp.Code)

		var errResp helpers.ErrorResponse
		parseResponse(t, resp, &errResp)
		assert.Equal(t, "farm not found", errResp.Error)
	})
}

func TestListFarms(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("empty list", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/farms", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(0), result.Total)
	})

	t.Run("with pagination", func(t *testing.T) {
		// Seed 3 farms
		for i := 0; i < 3; i++ {
			seedFarm(t, db)
		}

		resp := makeRequest(t, router, "GET", "/farms?page=1&limit=2", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(3), result.Total)
		assert.Equal(t, 2, result.TotalPages)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("filter by location", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/farms?location=Hawaii", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Greater(t, result.Total, int64(0))
	})
}

func TestUpdateFarm(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("update name only", func(t *testing.T) {
		farm := seedFarm(t, db)

		body := map[string]any{
			"name": "Updated Name",
		}

		resp := makeRequest(t, router, "PUT", fmt.Sprintf("/farms/%d", farm.ID), body)
		assert.Equal(t, http.StatusOK, resp.Code)

		var updated models.Farm
		parseResponse(t, resp, &updated)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, farm.Location, updated.Location) // unchanged
	})

	t.Run("update non-existent farm", func(t *testing.T) {
		body := map[string]any{"name": "Ghost Farm"}
		resp := makeRequest(t, router, "PUT", "/farms/999", body)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestDeleteFarm(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("delete existing farm", func(t *testing.T) {
		farm := seedFarm(t, db)

		resp := makeRequest(t, router, "DELETE", fmt.Sprintf("/farms/%d", farm.ID), nil)
		assert.Equal(t, http.StatusNoContent, resp.Code)

		// Verify it's actually gone
		resp = makeRequest(t, router, "GET", fmt.Sprintf("/farms/%d", farm.ID), nil)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("delete non-existent farm", func(t *testing.T) {
		resp := makeRequest(t, router, "DELETE", "/farms/999", nil)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}
```

### Key testing patterns to understand:

**`t.Run("name", func(t *testing.T) {...})`** — subtests. Each subtest gets its own name and can be run individually:
```bash
go test ./handlers/ -run TestCreateFarm/valid_farm
```
Spaces in the name become underscores when running.

**`assert` vs `require`:**
- `assert.Equal(t, expected, actual)` — logs failure but continues the test
- `require.NoError(t, err)` — logs failure and STOPS the test immediately

Use `require` for setup/preconditions (if setup fails, no point continuing). Use `assert` for the actual assertions you're testing.

**Fresh DB per test function** — each `TestXxx` function calls `setupTestDB(t)` which creates a brand new in-memory database. Tests don't share state and can run in any order.

---

## Step 35: Write BananaTree handler tests

### Create `handlers/banana_tree_test.go`

```go
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

func TestCreateBananaTree(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("valid tree", func(t *testing.T) {
		farm := seedFarm(t, db)

		body := map[string]any{
			"farm_id":    farm.ID,
			"variety":    "cavendish",
			"planted_at": "2024-03-01",
		}

		resp := makeRequest(t, router, "POST", "/trees", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var tree models.BananaTree
		parseResponse(t, resp, &tree)
		assert.Equal(t, "cavendish", tree.Variety)
		assert.Equal(t, "planted", tree.Status)  // default
		assert.Equal(t, "healthy", tree.Health)   // default
	})

	t.Run("invalid variety", func(t *testing.T) {
		farm := seedFarm(t, db)

		body := map[string]any{
			"farm_id":    farm.ID,
			"variety":    "invalid_variety",
			"planted_at": "2024-03-01",
		}

		resp := makeRequest(t, router, "POST", "/trees", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("non-existent farm", func(t *testing.T) {
		body := map[string]any{
			"farm_id":    999,
			"variety":    "cavendish",
			"planted_at": "2024-03-01",
		}

		resp := makeRequest(t, router, "POST", "/trees", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)

		var errResp helpers.ErrorResponse
		parseResponse(t, resp, &errResp)
		assert.Equal(t, "farm not found", errResp.Error)
	})
}

func TestUpdateBananaTreeStatus(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("update lifecycle status", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)

		// Progress through lifecycle: planted → growing → flowering → fruiting
		statuses := []string{"growing", "flowering", "fruiting", "harvested", "dead"}

		for _, status := range statuses {
			body := map[string]any{"status": status}
			resp := makeRequest(t, router, "PUT", fmt.Sprintf("/trees/%d", tree.ID), body)
			assert.Equal(t, http.StatusOK, resp.Code)

			var updated models.BananaTree
			parseResponse(t, resp, &updated)
			assert.Equal(t, status, updated.Status)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)

		body := map[string]any{"status": "exploding"}
		resp := makeRequest(t, router, "PUT", fmt.Sprintf("/trees/%d", tree.ID), body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestListBananaTreesFilter(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)
	farm := seedFarm(t, db)

	// Seed trees with different statuses
	tree1 := seedTree(t, db, farm.ID)
	db.Model(&tree1).Update("status", "flowering")

	tree2 := seedTree(t, db, farm.ID)
	db.Model(&tree2).Update("status", "fruiting")

	seedTree(t, db, farm.ID) // status: planted (default)

	t.Run("filter by status", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/trees?status=flowering", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(1), result.Total)
	})

	t.Run("filter by farm_id", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", fmt.Sprintf("/trees?farm_id=%d", farm.ID), nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(3), result.Total)
	})
}

func TestListTreeBunches(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("tree with bunches", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		seedBunch(t, db, tree.ID)
		seedBunch(t, db, tree.ID)

		resp := makeRequest(t, router, "GET", fmt.Sprintf("/trees/%d/bunches", tree.ID), nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(2), result.Total)
	})

	t.Run("non-existent tree", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/trees/999/bunches", nil)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}
```

---

## Step 36: Write Banana and Bunch handler tests

### Create `handlers/bunch_test.go`

```go
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

func TestCreateBunch(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("valid bunch", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)

		body := map[string]any{
			"banana_tree_id": tree.ID,
			"weight_kg":      25.5,
		}

		resp := makeRequest(t, router, "POST", "/bunches", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var bunch models.Bunch
		parseResponse(t, resp, &bunch)
		assert.Equal(t, tree.ID, bunch.BananaTreeID)
		assert.Equal(t, 25.5, bunch.WeightKg)
		assert.Nil(t, bunch.HarvestedAt) // not harvested yet
	})

	t.Run("non-existent tree", func(t *testing.T) {
		body := map[string]any{
			"banana_tree_id": 999,
		}

		resp := makeRequest(t, router, "POST", "/bunches", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestUpdateBunchHarvest(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("mark as harvested", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		bunch := seedBunch(t, db, tree.ID)

		body := map[string]any{
			"harvested_at": "2024-06-15",
			"weight_kg":    30.0,
		}

		resp := makeRequest(t, router, "PUT", fmt.Sprintf("/bunches/%d", bunch.ID), body)
		assert.Equal(t, http.StatusOK, resp.Code)

		var updated models.Bunch
		parseResponse(t, resp, &updated)
		assert.NotNil(t, updated.HarvestedAt)
		assert.Equal(t, 30.0, updated.WeightKg)
	})
}
```

### Create `handlers/banana_test.go`

```go
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

func TestCreateBanana(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("valid banana", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		bunch := seedBunch(t, db, tree.ID)

		body := map[string]any{
			"bunch_id":     bunch.ID,
			"hand_number":  3,
			"size":         "large",
			"ripeness":     "green",
			"weight_grams": 120.5,
		}

		resp := makeRequest(t, router, "POST", "/bananas", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var banana models.Banana
		parseResponse(t, resp, &banana)
		assert.Equal(t, 3, banana.HandNumber)
		assert.Equal(t, "large", banana.Size)
		assert.Equal(t, "green", banana.Ripeness)
	})

	t.Run("defaults applied", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		bunch := seedBunch(t, db, tree.ID)

		body := map[string]any{
			"bunch_id":    bunch.ID,
			"hand_number": 1,
		}

		resp := makeRequest(t, router, "POST", "/bananas", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var banana models.Banana
		parseResponse(t, resp, &banana)
		assert.Equal(t, "medium", banana.Size)    // default
		assert.Equal(t, "green", banana.Ripeness)  // default
	})

	t.Run("hand_number out of range", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		bunch := seedBunch(t, db, tree.ID)

		body := map[string]any{
			"bunch_id":    bunch.ID,
			"hand_number": 25, // max is 20
		}

		resp := makeRequest(t, router, "POST", "/bananas", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("invalid ripeness", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		bunch := seedBunch(t, db, tree.ID)

		body := map[string]any{
			"bunch_id":    bunch.ID,
			"hand_number": 1,
			"ripeness":    "rotten",
		}

		resp := makeRequest(t, router, "POST", "/bananas", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestUpdateBananaRipeness(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	t.Run("ripen a banana", func(t *testing.T) {
		farm := seedFarm(t, db)
		tree := seedTree(t, db, farm.ID)
		bunch := seedBunch(t, db, tree.ID)

		// Create a banana
		banana := models.Banana{
			BunchID:    bunch.ID,
			HandNumber: 1,
			Size:       "medium",
			Ripeness:   "green",
		}
		db.Create(&banana)

		// Update ripeness: green → turning → ripe
		for _, ripeness := range []string{"turning", "ripe"} {
			body := map[string]any{"ripeness": ripeness}
			resp := makeRequest(t, router, "PUT", fmt.Sprintf("/bananas/%d", banana.ID), body)
			assert.Equal(t, http.StatusOK, resp.Code)

			var updated models.Banana
			parseResponse(t, resp, &updated)
			assert.Equal(t, ripeness, updated.Ripeness)
		}
	})
}

func TestListBananasFilter(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	farm := seedFarm(t, db)
	tree := seedTree(t, db, farm.ID)
	bunch := seedBunch(t, db, tree.ID)

	// Seed bananas with different attributes
	db.Create(&models.Banana{BunchID: bunch.ID, HandNumber: 1, Size: "large", Ripeness: "green"})
	db.Create(&models.Banana{BunchID: bunch.ID, HandNumber: 2, Size: "small", Ripeness: "ripe"})
	db.Create(&models.Banana{BunchID: bunch.ID, HandNumber: 3, Size: "large", Ripeness: "ripe"})

	t.Run("filter by ripeness", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/bananas?ripeness=ripe", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(2), result.Total)
	})

	t.Run("filter by size", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/bananas?size=large", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(2), result.Total)
	})

	t.Run("combined filters", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/bananas?size=large&ripeness=ripe", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(1), result.Total)
	})
}
```

---

## Step 37: Write Tool and Worker handler tests

### Create `handlers/tool_test.go`

```go
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/models"
)

func TestToolCRUD(t *testing.T) {
	// Each subtest sets up its own state — no shared toolID that creates ordering dependency.

	t.Run("create tool with defaults", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		body := map[string]any{
			"farm_id": farm.ID,
			"name":    "Big Machete",
			"type":    "machete",
		}

		resp := makeRequest(t, router, "POST", "/tools", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var tool models.Tool
		parseResponse(t, resp, &tool)
		assert.Equal(t, "Big Machete", tool.Name)
		assert.Equal(t, "machete", tool.Type)
		assert.Equal(t, "new", tool.Condition) // default
	})

	t.Run("get existing tool", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		// Create the tool directly in DB
		tool := models.Tool{FarmID: farm.ID, Name: "Knife", Type: "harvesting_knife", Condition: "good"}
		db.Create(&tool)

		resp := makeRequest(t, router, "GET", fmt.Sprintf("/tools/%d", tool.ID), nil)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("update tool condition", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		tool := models.Tool{FarmID: farm.ID, Name: "Wheelbarrow", Type: "wheelbarrow", Condition: "new"}
		db.Create(&tool)

		body := map[string]any{"condition": "worn"}
		resp := makeRequest(t, router, "PUT", fmt.Sprintf("/tools/%d", tool.ID), body)
		assert.Equal(t, http.StatusOK, resp.Code)

		var updated models.Tool
		parseResponse(t, resp, &updated)
		assert.Equal(t, "worn", updated.Condition)
	})

	t.Run("delete tool", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		tool := models.Tool{FarmID: farm.ID, Name: "Pruner", Type: "pruning_shears", Condition: "good"}
		db.Create(&tool)

		resp := makeRequest(t, router, "DELETE", fmt.Sprintf("/tools/%d", tool.ID), nil)
		assert.Equal(t, http.StatusNoContent, resp.Code)

		// Verify deleted
		resp = makeRequest(t, router, "GET", fmt.Sprintf("/tools/%d", tool.ID), nil)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("invalid tool type rejected", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		body := map[string]any{
			"farm_id": farm.ID,
			"name":    "Laser Gun",
			"type":    "laser",
		}

		resp := makeRequest(t, router, "POST", "/tools", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}
```

### Create `handlers/worker_test.go`

```go
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
)

func TestWorkerCRUD(t *testing.T) {
	// Each subtest sets up its own state — no shared workerID that creates ordering dependency.

	t.Run("create worker", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		body := map[string]any{
			"farm_id": farm.ID,
			"name":    "Juan",
			"role":    "harvester",
		}

		resp := makeRequest(t, router, "POST", "/workers", body)
		assert.Equal(t, http.StatusCreated, resp.Code)

		var worker models.Worker
		parseResponse(t, resp, &worker)
		assert.Equal(t, "Juan", worker.Name)
		assert.Equal(t, "harvester", worker.Role)
		assert.NotZero(t, worker.ID)
	})

	t.Run("get worker", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		worker := models.Worker{FarmID: farm.ID, Name: "Maria", Role: "farmer"}
		db.Create(&worker)

		resp := makeRequest(t, router, "GET", fmt.Sprintf("/workers/%d", worker.ID), nil)
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("update worker role", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		worker := models.Worker{FarmID: farm.ID, Name: "Carlos", Role: "irrigator"}
		db.Create(&worker)

		body := map[string]any{"role": "supervisor"}
		resp := makeRequest(t, router, "PUT", fmt.Sprintf("/workers/%d", worker.ID), body)
		assert.Equal(t, http.StatusOK, resp.Code)

		var updated models.Worker
		parseResponse(t, resp, &updated)
		assert.Equal(t, "supervisor", updated.Role)
	})

	t.Run("delete worker", func(t *testing.T) {
		db := setupTestDB(t)
		router := setupTestRouter(t, db)
		farm := seedFarm(t, db)

		worker := models.Worker{FarmID: farm.ID, Name: "Ana", Role: "farmer"}
		db.Create(&worker)

		resp := makeRequest(t, router, "DELETE", fmt.Sprintf("/workers/%d", worker.ID), nil)
		assert.Equal(t, http.StatusNoContent, resp.Code)

		// Verify deleted
		resp = makeRequest(t, router, "GET", fmt.Sprintf("/workers/%d", worker.ID), nil)
		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestWorkerFilters(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(t, db)
	farm := seedFarm(t, db)

	db.Create(&models.Worker{FarmID: farm.ID, Name: "Alice", Role: "farmer"})
	db.Create(&models.Worker{FarmID: farm.ID, Name: "Bob", Role: "harvester"})
	db.Create(&models.Worker{FarmID: farm.ID, Name: "Charlie", Role: "harvester"})

	t.Run("filter by role", func(t *testing.T) {
		resp := makeRequest(t, router, "GET", "/workers?role=harvester", nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var result helpers.PaginatedResponse
		parseResponse(t, resp, &result)
		assert.Equal(t, int64(2), result.Total)
	})
}
```

---

## Step 38: Write middleware tests

### Create `middleware/logging_test.go`

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/middleware"
)

func TestLoggerMiddleware(t *testing.T) {
	t.Run("passes request through", func(t *testing.T) {
		// The logger middleware should call the next handler and not block
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Logger(next)

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("preserves status code from handler", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		handler := middleware.Logger(next)

		req := httptest.NewRequest("GET", "/missing", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
```

### Create `middleware/error_test.go`

```go
package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/middleware"
)

func TestNotFoundHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()

	middleware.NotFoundHandler(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var errResp helpers.ErrorResponse
	err := json.NewDecoder(rr.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "route not found", errResp.Error)
}

func TestMethodNotAllowedHandler(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/farms", nil)
	rr := httptest.NewRecorder()

	middleware.MethodNotAllowedHandler(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

	var errResp helpers.ErrorResponse
	err := json.NewDecoder(rr.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "method not allowed", errResp.Error)
}
```

---

## Step 39: Write pagination helper tests

### Create `helpers/pagination_test.go`

```go
package helpers_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/helpers"
)

func TestParsePagination(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/farms", nil)
		p := helpers.ParsePagination(req)

		assert.Equal(t, 1, p.Page)
		assert.Equal(t, 20, p.Limit)
	})

	t.Run("custom values", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/farms?page=3&limit=50", nil)
		p := helpers.ParsePagination(req)

		assert.Equal(t, 3, p.Page)
		assert.Equal(t, 50, p.Limit)
	})

	t.Run("max limit enforced", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/farms?limit=999", nil)
		p := helpers.ParsePagination(req)

		assert.Equal(t, 100, p.Limit) // capped at MaxLimit
	})

	t.Run("negative values use defaults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/farms?page=-1&limit=-5", nil)
		p := helpers.ParsePagination(req)

		assert.Equal(t, 1, p.Page)
		assert.Equal(t, 20, p.Limit)
	})

	t.Run("non-numeric values use defaults", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/farms?page=abc&limit=xyz", nil)
		p := helpers.ParsePagination(req)

		assert.Equal(t, 1, p.Page)
		assert.Equal(t, 20, p.Limit)
	})
}

func TestPaginationOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		limit    int
		expected int
	}{
		{"page 1", 1, 20, 0},
		{"page 2", 2, 20, 20},
		{"page 3 with limit 10", 3, 10, 20},
		{"page 5 with limit 5", 5, 5, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := helpers.Pagination{Page: tt.page, Limit: tt.limit}
			assert.Equal(t, tt.expected, p.Offset())
		})
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	t.Run("calculates total pages", func(t *testing.T) {
		pagination := helpers.Pagination{Page: 1, Limit: 10}
		resp := helpers.NewPaginatedResponse([]string{}, 25, pagination)

		assert.Equal(t, 3, resp.TotalPages) // 25/10 = 2.5 → 3
		assert.Equal(t, int64(25), resp.Total)
	})

	t.Run("exact division", func(t *testing.T) {
		pagination := helpers.Pagination{Page: 1, Limit: 10}
		resp := helpers.NewPaginatedResponse([]string{}, 20, pagination)

		assert.Equal(t, 2, resp.TotalPages) // 20/10 = 2 exactly
	})

	t.Run("zero results", func(t *testing.T) {
		pagination := helpers.Pagination{Page: 1, Limit: 10}
		resp := helpers.NewPaginatedResponse([]string{}, 0, pagination)

		assert.Equal(t, 0, resp.TotalPages)
		assert.Equal(t, int64(0), resp.Total)
	})
}
```

### Key pattern — table-driven tests:
```go
tests := []struct {
    name     string
    page     int
    limit    int
    expected int
}{
    {"page 1", 1, 20, 0},
    {"page 2", 2, 20, 20},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic using tt.page, tt.limit, tt.expected
    })
}
```
This is a most common Go testing pattern. Instead of writing separate test functions for each case, you define a slice of test cases and loop over them. Each case gets its own subtest name.
---

## Step 40: Write response helper tests

### Create `helpers/response_test.go`

```go
package helpers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/justincordova/banana-farm-api/helpers"
)

func TestRespondJSON(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		w := httptest.NewRecorder()

		payload := map[string]any{"message": "hello", "count": 42}
		helpers.RespondJSON(w, http.StatusOK, payload)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]any
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "hello", response["message"])
		assert.Equal(t, float64(42), response["count"]) // JSON numbers decode to float64
	})

	t.Run("sets status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		helpers.RespondJSON(w, http.StatusCreated, map[string]any{"id": 1})

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestRespondError(t *testing.T) {
	t.Run("error response", func(t *testing.T) {
		w := httptest.NewRecorder()
		helpers.RespondError(w, http.StatusBadRequest, "invalid input")

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response helpers.ErrorResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "invalid input", response.Error)
	})

	t.Run("not found error", func(t *testing.T) {
		w := httptest.NewRecorder()
		helpers.RespondError(w, http.StatusNotFound, "item not found")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRespondErrorWithDetails(t *testing.T) {
	t.Run("error with details", func(t *testing.T) {
		w := httptest.NewRecorder()
		helpers.RespondErrorWithDetails(w, http.StatusBadRequest, "validation failed", "field 'name' is required")

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response helpers.ErrorResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "validation failed", response.Error)
		assert.Equal(t, "field 'name' is required", response.Details)
	})
}

func TestDecodeJSON(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		body := `{"name": "test", "count": 5}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

		var result map[string]any
		err := helpers.DecodeJSON(req, &result)

		require.NoError(t, err)
		assert.Equal(t, "test", result["name"])
		assert.Equal(t, float64(5), result["count"])
	})

	t.Run("malformed JSON", func(t *testing.T) {
		body := `{"name": "test", invalid}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

		var result map[string]any
		err := helpers.DecodeJSON(req, &result)

		assert.Error(t, err)
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		// DecodeJSON calls DisallowUnknownFields(), which only triggers when
		// decoding into a struct — map[string]any accepts any key by design.
		type strictBody struct {
			Name string `json:"name"`
		}
		body := `{"name": "test", "unknown_field": "value"}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

		var result strictBody
		err := helpers.DecodeJSON(req, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader(""))

		var result map[string]any
		err := helpers.DecodeJSON(req, &result)

		assert.Error(t, err)
	})
}
```

---

## Step 41: Write ParseDate tests

### Create `handlers/helpers_test.go`

Create a test file for the `ParseDate` utility function. Because `ParseDate` is now exported,
it's accessible from the `handlers_test` (black-box) package.

```go
package handlers_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/justincordova/banana-farm-api/handlers"
)

func TestParseDate(t *testing.T) {
	t.Run("valid date formats", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected time.Time
		}{
			{"standard date", "2024-01-15", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
			{"january 1st", "2024-01-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			{"december 31st", "2024-12-31", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := handlers.ParseDate(tt.input)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("invalid formats", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{"US format", "01/15/2024"},
			{"ISO with time", "2024-01-15T10:30:00"},
			{"text month", "January 15, 2024"},
			{"missing year", "01-15"},
			{"garbage", "not-a-date"},
			{"empty string", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := handlers.ParseDate(tt.input)
				assert.Error(t, err)
			})
		}
	})

	t.Run("leap year handling", func(t *testing.T) {
		// February 29th is valid in leap years
		result, err := handlers.ParseDate("2024-02-29")
		assert.NoError(t, err)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.February, result.Month())
		assert.Equal(t, 29, result.Day())

		// Invalid in non-leap year
		_, err = handlers.ParseDate("2023-02-29")
		assert.Error(t, err)
	})
}
```

---

## Step 42: Write model tests

### Create `models/models_test.go`

```go
package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/justincordova/banana-farm-api/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Enable foreign key constraints — required for ON DELETE CASCADE to work.
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)

	err = db.AutoMigrate(
		&models.Farm{},
		&models.BananaTree{},
		&models.Bunch{},
		&models.Banana{},
		&models.Tool{},
		&models.Worker{},
	)
	require.NoError(t, err)

	return db
}

func TestFarmRelationships(t *testing.T) {
	db := setupTestDB(t)

	t.Run("farm has many trees", func(t *testing.T) {
		farm := models.Farm{
			Name:        "Test Farm",
			Location:    "Hawaii",
			SizeAcres:   100,
			Established: time.Time{},
		}
		db.Create(&farm)

		// Create trees on this farm
		tree1 := models.BananaTree{
			FarmID:    farm.ID,
			Variety:   "cavendish",
			PlantedAt: time.Time{},
			Status:    "planted",
			Health:    "healthy",
		}
		tree2 := models.BananaTree{
			FarmID:    farm.ID,
			Variety:   "plantain",
			PlantedAt: time.Time{},
			Status:    "planted",
			Health:    "healthy",
		}
		db.Create(&tree1)
		db.Create(&tree2)

		// Preload trees and verify
		var loadedFarm models.Farm
		db.Preload("BananaTrees").First(&loadedFarm, farm.ID)

		assert.Len(t, loadedFarm.BananaTrees, 2)
		assert.Equal(t, "cavendish", loadedFarm.BananaTrees[0].Variety)
	})

	t.Run("farm has many workers", func(t *testing.T) {
		farm := models.Farm{
			Name:        "Worker Farm",
			Location:    "Costa Rica",
			SizeAcres:   50,
			Established: time.Time{},
		}
		db.Create(&farm)

		worker := models.Worker{
			FarmID: farm.ID,
			Name:   "Juan",
			Role:   "harvester",
		}
		db.Create(&worker)

		var loadedFarm models.Farm
		db.Preload("Workers").First(&loadedFarm, farm.ID)

		assert.Len(t, loadedFarm.Workers, 1)
		assert.Equal(t, "Juan", loadedFarm.Workers[0].Name)
	})
}

func TestCascadingDeletes(t *testing.T) {
	db := setupTestDB(t)

	t.Run("delete farm deletes trees", func(t *testing.T) {
		farm := models.Farm{
			Name:        "Cascade Farm",
			Location:    "Ecuador",
			SizeAcres:   75,
			Established: time.Time{},
		}
		db.Create(&farm)

		tree := models.BananaTree{
			FarmID:    farm.ID,
			Variety:   "cavendish",
			PlantedAt: time.Time{},
			Status:    "planted",
			Health:    "healthy",
		}
		db.Create(&tree)

		// Delete farm
		db.Delete(&farm)

		// Verify tree is deleted (cascading)
		var treeCount int64
		db.Model(&models.BananaTree{}).Where("id = ?", tree.ID).Count(&treeCount)
		assert.Equal(t, int64(0), treeCount)
	})

	t.Run("delete tree deletes bunches and bananas", func(t *testing.T) {
		farm := models.Farm{
			Name:        "Tree Farm",
			Location:    "Philippines",
			SizeAcres:   60,
			Established: time.Time{},
		}
		db.Create(&farm)

		tree := models.BananaTree{
			FarmID:    farm.ID,
			Variety:   "cavendish",
			PlantedAt: time.Time{},
			Status:    "planted",
			Health:    "healthy",
		}
		db.Create(&tree)

		bunch := models.Bunch{
			BananaTreeID: tree.ID,
			WeightKg:     20.0,
		}
		db.Create(&bunch)

		banana := models.Banana{
			BunchID:     bunch.ID,
			HandNumber:  1,
			Size:        "medium",
			Ripeness:    "green",
			WeightGrams: 100.0,
		}
		db.Create(&banana)

		// Delete tree
		db.Delete(&tree)

		// Verify cascading
		var bunchCount int64
		db.Model(&models.Bunch{}).Where("id = ?", bunch.ID).Count(&bunchCount)
		assert.Equal(t, int64(0), bunchCount)

		var bananaCount int64
		db.Model(&models.Banana{}).Where("id = ?", banana.ID).Count(&bananaCount)
		assert.Equal(t, int64(0), bananaCount)
	})
}

func TestNestedRelationships(t *testing.T) {
	db := setupTestDB(t)

	t.Run("tree to bunches to bananas", func(t *testing.T) {
		farm := models.Farm{
			Name:        "Nested Farm",
			Location:    "India",
			SizeAcres:   80,
			Established: time.Time{},
		}
		db.Create(&farm)

		tree := models.BananaTree{
			FarmID:    farm.ID,
			Variety:   "cavendish",
			PlantedAt: time.Time{},
			Status:    "planted",
			Health:    "healthy",
		}
		db.Create(&tree)

		bunch := models.Bunch{
			BananaTreeID: tree.ID,
			WeightKg:     25.0,
		}
		db.Create(&bunch)

		banana := models.Banana{
			BunchID:     bunch.ID,
			HandNumber:  2,
			Size:        "large",
			Ripeness:    "green",
			WeightGrams: 150.0,
		}
		db.Create(&banana)

		// Preload entire chain
		var loadedTree models.BananaTree
		db.Preload("Bunches.Bananas").First(&loadedTree, tree.ID)

		assert.Len(t, loadedTree.Bunches, 1)
		assert.Len(t, loadedTree.Bunches[0].Bananas, 1)
		assert.Equal(t, 2, loadedTree.Bunches[0].Bananas[0].HandNumber)
	})
}
```

---

## Step 43: Write health handler tests

### Create `handlers/health_test.go`

```go
package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/justincordova/banana-farm-api/handlers"
)

func TestHealthCheck(t *testing.T) {
	t.Run("healthy when DB connected", func(t *testing.T) {
		// Each subtest gets its own DB so closing one doesn't affect others.
		db := setupTestDB(t)
		router := setupTestRouter(t, db)

		rr := makeRequest(t, router, "GET", "/health", nil)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response handlers.HealthResponse
		parseResponse(t, rr, &response)

		assert.Equal(t, "healthy", response.Status)
		assert.Equal(t, "test", response.Env) // setupTestRouter sets AppEnv="test"
		assert.NotEmpty(t, response.Uptime)
		assert.NotEmpty(t, response.GoVersion)
		assert.Greater(t, response.PID, 0)
		assert.Equal(t, "connected", response.DB.Status)
	})

	t.Run("returns 503 when DB disconnected", func(t *testing.T) {
		// Use a separate DB for this subtest — we're going to close it.
		db := setupTestDB(t)
		router := setupTestRouter(t, db)

		// Close the underlying connection to simulate DB failure.
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.Close()

		rr := makeRequest(t, router, "GET", "/health", nil)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

		var response handlers.HealthResponse
		parseResponse(t, rr, &response)

		assert.Equal(t, "unhealthy", response.Status)
		assert.Equal(t, "disconnected", response.DB.Status)
	})
}

func TestHealthHandler_Uptime(t *testing.T) {
	// The router is built with startTime = time.Now(), so we verify the uptime
	// is a short positive duration (not that it's a specific value, since that
	// would be flaky). The HealthHandler stores startTime; we can't override it
	// from the outside once the router is built. Test that the field is non-empty
	// and parseable instead.
	db := setupTestDB(t)
	router := setupTestRouter(t, db)

	rr := makeRequest(t, router, "GET", "/health", nil)

	var response handlers.HealthResponse
	parseResponse(t, rr, &response)

	// Uptime should be a non-empty string like "0s" or "1ms"
	assert.NotEmpty(t, response.Uptime)
}
```

---

## Step 44: Run all tests

```bash
# Run all tests
go test ./...

# Run with verbose output (see each test name)
go test -v ./...

# Run with coverage
go test -cover ./...

# Run a specific test file
go test -v ./handlers/ -run TestCreateFarm

# Run a specific subtest
go test -v ./handlers/ -run TestCreateFarm/valid_farm

# Generate coverage report (HTML)
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
# Open coverage.html in your browser to see line-by-line coverage
```

### Expected output:
```
=== RUN   TestCreateFarm
=== RUN   TestCreateFarm/valid_farm
=== RUN   TestCreateFarm/missing_required_fields
=== RUN   TestCreateFarm/invalid_date_format
=== RUN   TestCreateFarm/invalid_request_body
=== RUN   TestCreateFarm/negative_size
--- PASS: TestCreateFarm (0.02s)
    --- PASS: TestCreateFarm/valid_farm (0.01s)
    --- PASS: TestCreateFarm/missing_required_fields (0.00s)
    --- PASS: TestCreateFarm/invalid_date_format (0.00s)
    --- PASS: TestCreateFarm/invalid_request_body (0.00s)
    --- PASS: TestCreateFarm/negative_size (0.00s)
...
PASS
ok  	github.com/justincordova/banana-farm-api/handlers	0.15s
ok  	github.com/justincordova/banana-farm-api/helpers	0.01s
ok  	github.com/justincordova/banana-farm-api/middleware	0.01s
```

---

## Phase 5 Checklist

- [ ] `handlers/test_helpers_test.go` with setupTestDB, setupTestRouter, makeRequest, parseResponse, seed functions
- [ ] `handlers/helpers_test.go` with ParseDate tests
- [ ] `handlers/farm_test.go` — Create, Get, List, Update, Delete with edge cases
- [ ] `handlers/health_test.go` — healthy response, DB failure detection, uptime calculation
- [ ] `handlers/banana_tree_test.go` — CRUD, lifecycle status updates, filters, nested bunches
- [ ] `handlers/bunch_test.go` — CRUD, harvest date update
- [ ] `handlers/banana_test.go` — CRUD, defaults, validation, ripeness updates, combined filters
- [ ] `handlers/tool_test.go` — full CRUD lifecycle, invalid type rejection
- [ ] `handlers/worker_test.go` — full CRUD lifecycle, role filter
- [ ] `middleware/logging_test.go` — passes requests through, preserves status codes
- [ ] `middleware/error_test.go` — NotFound and MethodNotAllowed return JSON
- [ ] `helpers/response_test.go` — RespondJSON, RespondError, RespondErrorWithDetails, DecodeJSON
- [ ] `helpers/pagination_test.go` — defaults, max limit, offset calculation, total pages
- [ ] `models/models_test.go` — relationships, cascading deletes, nested preloading
- [ ] `go test ./...` passes with no failures
- [ ] `go test -cover ./...` shows coverage you're happy with

---

## Project Complete!

At this point your Banana Farm API has:

1. **6 entities** with full CRUD (Farm, BananaTree, Bunch, Banana, Tool, Worker)
2. **Relationships** navigable via nested routes (Farm → Tree → Bunch → Banana)
3. **Lifecycle tracking** for banana trees (planted → dead)
4. **Pagination** on all list endpoints
5. **Filtering** by relevant fields on each entity
6. **Farm stats** with aggregate counts and breakdowns
7. **Health check** with DB ping and uptime
8. **Middleware** — CORS, rate limiting, request ID, panic recovery, request logging
9. **Graceful shutdown** with signal handling
10. **Colored logging** in dev, JSON in production
11. **Environment-based config** loaded from `.env`
12. **Comprehensive tests** using in-memory SQLite

Nice work learning Go!
