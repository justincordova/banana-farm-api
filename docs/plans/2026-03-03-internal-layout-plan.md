# cmd/internal Layout Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor all plan docs and the existing `config/` directory to use the standard Go `cmd/api/` + `internal/` layout.

**Architecture:** Move `config/` to `internal/config/`, update `docs/PLAN.md` and all 5 phase plans to reflect the new directory structure and corrected import paths. No application logic changes — this is purely a structural and documentation refactor.

**Tech Stack:** Go standard project layout, markdown docs.

---

### Task 1: Move existing `config/` to `internal/config/`

**Files:**
- Create: `internal/config/config.go` (move from `config/config.go`)
- Create: `internal/config/logger.go` (move from `config/logger.go`)
- Delete: `config/config.go`, `config/logger.go`, `config/` directory

**Step 1: Create `internal/config/` directory and move files**

```bash
mkdir -p internal/config
mv config/config.go internal/config/config.go
mv config/logger.go internal/config/logger.go
rmdir config
```

**Step 2: Verify the files moved correctly**

```bash
ls internal/config/
# Expected: config.go  logger.go
ls config/
# Expected: ls: config/: No such file or directory
```

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor: move config/ to internal/config/"
```

---

### Task 2: Update `docs/PLAN.md` — project structure section

**Files:**
- Modify: `docs/PLAN.md`

The `## Project Structure` section (lines 73–125) currently shows all packages at the root. Replace the entire tree with the `cmd/internal` layout, and update the `## Build Order` section (lines 330–368) to reflect new file paths.

**Step 1: Update the project structure tree**

In `docs/PLAN.md`, replace:

```
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
```

With:

```
## Project Structure

```
banana-farm-api/
├── cmd/
│   └── api/
│       └── main.go             # Entry point: server setup, graceful shutdown
├── internal/
│   ├── config/
│   │   ├── config.go           # Config struct, env loading, APP_ENV
│   │   └── logger.go           # slog setup (tint for dev, JSON+lumberjack for prod)
│   │
│   ├── database/
│   │   └── database.go         # GORM connection, auto-migrate, close
│   │
│   ├── models/
│   │   ├── farm.go
│   │   ├── banana_tree.go
│   │   ├── bunch.go
│   │   ├── banana.go
│   │   ├── tool.go
│   │   └── worker.go
│   │
│   ├── handlers/
│   │   ├── farm.go
│   │   ├── farm_test.go
│   │   ├── banana_tree.go
│   │   ├── banana_tree_test.go
│   │   ├── bunch.go
│   │   ├── bunch_test.go
│   │   ├── banana.go
│   │   ├── banana_test.go
│   │   ├── tool.go
│   │   ├── tool_test.go
│   │   ├── worker.go
│   │   ├── worker_test.go
│   │   └── health.go
│   │
│   ├── routes/
│   │   └── routes.go           # Chi router, route registration, middleware stack
│   │
│   ├── middleware/
│   │   ├── logging.go          # slog request logger (method, path, status, duration)
│   │   ├── logging_test.go
│   │   └── error.go            # NotFound, MethodNotAllowed handlers
│   │
│   └── helpers/
│       ├── response.go         # respondJSON(), respondError() helpers
│       ├── pagination.go       # ParsePagination from query params
│       └── pagination_test.go
├── .env                        # Environment variables
├── .env.example                # Template for .env
├── go.mod
├── .air.toml                   # Hot reload config
└── README.md
```
```

**Step 2: Update the Build Order section**

In the `## Build Order` section, replace all file paths that don't have an `internal/` or `cmd/` prefix:

| Old path | New path |
|----------|----------|
| `config/config.go` | `internal/config/config.go` |
| `config/logger.go` | `internal/config/logger.go` |
| `database/database.go` | `internal/database/database.go` |
| `main.go` | `cmd/api/main.go` |
| `models/farm.go` | `internal/models/farm.go` |
| `helpers/response.go` | `internal/helpers/response.go` |
| `helpers/pagination.go` | `internal/helpers/pagination.go` |
| `handlers/farm.go` | `internal/handlers/farm.go` |
| `routes/routes.go` | `internal/routes/routes.go` |
| `middleware/logging.go` | `internal/middleware/logging.go` |
| `middleware/error.go` | `internal/middleware/error.go` |
| `models/banana_tree.go` | `internal/models/banana_tree.go` |
| `handlers/banana_tree.go` | `internal/handlers/banana_tree.go` |
| `models/bunch.go` | `internal/models/bunch.go` |
| `handlers/bunch.go` | `internal/handlers/bunch.go` |
| `models/banana.go` | `internal/models/banana.go` |
| `handlers/banana.go` | `internal/handlers/banana.go` |
| `models/tool.go` | `internal/models/tool.go` |
| `handlers/tool.go` | `internal/handlers/tool.go` |
| `models/worker.go` | `internal/models/worker.go` |
| `handlers/worker.go` | `internal/handlers/worker.go` |
| `handlers/health.go` | `internal/handlers/health.go` |

**Step 3: Verify the doc reads correctly**

Open `docs/PLAN.md` and confirm the project structure tree shows `cmd/api/main.go` and all packages under `internal/`. Confirm the build order step references use the updated paths.

**Step 4: Commit**

```bash
git add docs/PLAN.md
git commit -m "docs: update PLAN.md project structure and build order to cmd/internal layout"
```

---

### Task 3: Update `docs/plans/phase-1-foundation.md`

**Files:**
- Modify: `docs/plans/phase-1-foundation.md`

Phase 1 creates `config/`, `database/`, and `main.go`. Every path and import needs updating.

**Step 1: Update Step 4 header (config/config.go)**

Replace:
```
## Step 4: Create `config/config.go`

Create the `config/` directory and the config file.
```

With:
```
## Step 4: Create `internal/config/config.go`

Create the `internal/config/` directory and the config file.
```

**Step 2: Update Step 5 header (config/logger.go)**

Replace:
```
## Step 5: Create `config/logger.go`
```

With:
```
## Step 5: Create `internal/config/logger.go`
```

**Step 3: Update Step 6 header and directory instruction (database/database.go)**

Replace:
```
## Step 6: Create `database/database.go`

Create the `database/` directory and the database file.
```

With:
```
## Step 6: Create `internal/database/database.go`

Create the `internal/database/` directory and the database file.
```

**Step 4: Update the import path in the database.go code block**

In the `database.go` code block, there are no internal imports from this project yet, so nothing to change in the Go code itself.

**Step 5: Update Step 7 header and location (main.go)**

Replace:
```
## Step 7: Create `main.go`

This is the entry point. It wires everything together and handles graceful shutdown.
```

With:
```
## Step 7: Create `cmd/api/main.go`

Create the `cmd/api/` directory. This is the entry point — it wires everything together and handles graceful shutdown.
```

**Step 6: Update imports in the main.go code block**

In the `main.go` code block, replace:
```go
	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/database"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/database"
```

**Step 7: Update the Air config in Step 8**

In the `.air.toml` code block, replace:
```toml
  cmd = "go build -o ./tmp/main ."
```

With:
```toml
  cmd = "go build -o ./tmp/main ./cmd/api"
```

Also update the `.air.toml` How to use comment about `go run`:

Replace:
```bash
# Instead of `go run .`, use:
```

With:
```bash
# Instead of `go run ./cmd/api`, use:
```

**Step 8: Update the "Verify Phase 1" section**

Replace:
```bash
go run .
```

With:
```bash
go run ./cmd/api
```

**Step 9: Update the Phase 1 Checklist**

Replace:
```
- [ ] `config/config.go` loads env vars into a typed struct
- [ ] `config/logger.go` sets up slog with tint (colored dev output)
- [ ] `database/database.go` connects to SQLite and has migration placeholder
- [ ] `main.go` starts server, logs startup info, shuts down gracefully
```

With:
```
- [ ] `internal/config/config.go` loads env vars into a typed struct
- [ ] `internal/config/logger.go` sets up slog with tint (colored dev output)
- [ ] `internal/database/database.go` connects to SQLite and has migration placeholder
- [ ] `cmd/api/main.go` starts server, logs startup info, shuts down gracefully
```

**Step 10: Commit**

```bash
git add docs/plans/phase-1-foundation.md
git commit -m "docs: update phase-1 plan to cmd/internal layout"
```

---

### Task 4: Update `docs/plans/phase-2-first-entity.md`

**Files:**
- Modify: `docs/plans/phase-2-first-entity.md`

Phase 2 introduces `models/`, `helpers/`, `handlers/`, `routes/`, and `middleware/`. Every file path in headers and every import path in code blocks needs updating.

**Step 1: Update Step 8 header and directory instruction**

Replace:
```
## Step 8: Create `models/farm.go`

Create the `models/` directory and the Farm model.
```

With:
```
## Step 8: Create `internal/models/farm.go`

Create the `internal/models/` directory and the Farm model.
```

**Step 2: Update the stubs.go reference**

Replace:
```
// models/stubs.go — temporary, delete in Phase 3
```

With:
```
// internal/models/stubs.go — temporary, delete in Phase 3
```

**Step 3: Update the database.go Migrate function import**

In the "Update `database/database.go`" subsection, replace:
```go
import "github.com/justincordova/banana-farm-api/models"
```

With:
```go
import "github.com/justincordova/banana-farm-api/internal/models"
```

Also replace the header text:
```
### Update `database/database.go` to include the Farm model:
```

With:
```
### Update `internal/database/database.go` to include the Farm model:
```

**Step 4: Update Step 9 header and directory instruction**

Replace:
```
## Step 9: Create `helpers/response.go`

Create the `helpers/` directory. These are reusable functions every handler will use.
```

With:
```
## Step 9: Create `internal/helpers/response.go`

Create the `internal/helpers/` directory. These are reusable functions every handler will use.
```

**Step 5: Update Step 10 header**

Replace:
```
## Step 10: Create `helpers/pagination.go`
```

With:
```
## Step 10: Create `internal/helpers/pagination.go`
```

**Step 6: Update Step 11 header and directory instruction**

Replace:
```
## Step 11: Create `handlers/farm.go`

Create the `handlers/` directory.
```

With:
```
## Step 11: Create `internal/handlers/farm.go`

Create the `internal/handlers/` directory.
```

**Step 7: Update imports in the farm.go code block**

In the `handlers/farm.go` code block, replace:
```go
	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/helpers"
	"github.com/justincordova/banana-farm-api/internal/models"
```

**Step 8: Update the handlers/helpers.go subsection header**

Replace:
```
### Create `handlers/helpers.go` for shared handler utilities:
```

With:
```
### Create `internal/handlers/helpers.go` for shared handler utilities:
```

**Step 9: Update Step 12 header and directory instruction**

Replace:
```
## Step 12: Create `routes/routes.go`

Create the `routes/` directory.
```

With:
```
## Step 12: Create `internal/routes/routes.go`

Create the `internal/routes/` directory.
```

**Step 10: Update imports in the routes.go code block**

Replace:
```go
	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/handlers"
	"github.com/justincordova/banana-farm-api/middleware"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/handlers"
	"github.com/justincordova/banana-farm-api/internal/middleware"
```

**Step 11: Update Step 13 header and directory instruction**

Replace:
```
## Step 13: Create `middleware/logging.go`

Create the `middleware/` directory.
```

With:
```
## Step 13: Create `internal/middleware/logging.go`

Create the `internal/middleware/` directory.
```

**Step 12: Update Step 14 header**

Replace:
```
## Step 14: Create `middleware/error.go`
```

With:
```
## Step 14: Create `internal/middleware/error.go`
```

**Step 13: Update the import in the error.go code block**

Replace:
```go
	"github.com/justincordova/banana-farm-api/helpers"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/helpers"
```

**Step 14: Update Step 15 and the main.go import**

Replace:
```
**Add import:**
```go
"github.com/justincordova/banana-farm-api/routes"
```
```

With:
```
**Add import:**
```go
"github.com/justincordova/banana-farm-api/internal/routes"
```
```

**Step 15: Update the Phase 2 Checklist**

Replace every old path in the checklist:
```
- [ ] `models/farm.go` with GORM tags, JSON tags, and validation tags
- [ ] `models/stubs.go` with temporary stub types (to be replaced in Phase 3)
- [ ] `database/database.go` updated with Farm in AutoMigrate
- [ ] `helpers/response.go` with RespondJSON, RespondError, DecodeJSON
- [ ] `helpers/pagination.go` with ParsePagination, PaginatedResponse
- [ ] `handlers/farm.go` with List, Create, Get, Update, Delete
- [ ] `handlers/helpers.go` with ParseDate utility (exported for testability)
- [ ] `routes/routes.go` with Chi router, full middleware stack, farm routes
- [ ] `middleware/logging.go` with slog request logger
- [ ] `middleware/error.go` with NotFound and MethodNotAllowed handlers
- [ ] `main.go` updated to use `routes.Setup()`
```

With:
```
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
```

**Step 16: Commit**

```bash
git add docs/plans/phase-2-first-entity.md
git commit -m "docs: update phase-2 plan to cmd/internal layout"
```

---

### Task 5: Update `docs/plans/phase-3-remaining-entities.md`

**Files:**
- Modify: `docs/plans/phase-3-remaining-entities.md`

Phase 3 adds all remaining models and handlers. The file is large (~1675 lines). The changes are:
- All `## Step N: Create \`models/X\`` headers → `internal/models/X`
- All `## Step N: Create \`handlers/X\`` headers → `internal/handlers/X`
- All `package` declarations in code blocks are unchanged (they only use the last segment, e.g. `package models`)
- All import paths referencing this module need `internal/` inserted

**Step 1: Update all model file headers**

Find and update each of these section headers (search for `Create \`models/`):

| Old | New |
|-----|-----|
| `## Step 16: Create \`models/banana_tree.go\`` | `## Step 16: Create \`internal/models/banana_tree.go\`` |
| `**Delete \`models/stubs.go\`**` | `**Delete \`internal/models/stubs.go\`**` |
| `## Step 17: Create \`models/bunch.go\`` | `## Step 17: Create \`internal/models/bunch.go\`` |
| `## Step 18: Create \`models/banana.go\`` | `## Step 18: Create \`internal/models/banana.go\`` |
| `## Step 19: Create \`models/tool.go\`` | `## Step 19: Create \`internal/models/tool.go\`` |
| `## Step 20: Create \`models/worker.go\`` | `## Step 20: Create \`internal/models/worker.go\`` |

**Step 2: Update all handler file headers**

| Old | New |
|-----|-----|
| `## Step 21: Create \`handlers/banana_tree.go\`` | `## Step 21: Create \`internal/handlers/banana_tree.go\`` |
| `## Step 22: Create \`handlers/bunch.go\`` | `## Step 22: Create \`internal/handlers/bunch.go\`` |
| `## Step 23: Create \`handlers/banana.go\`` | `## Step 23: Create \`internal/handlers/banana.go\`` |
| `## Step 24: Create \`handlers/tool.go\`` | `## Step 24: Create \`internal/handlers/tool.go\`` |
| `## Step 25: Create \`handlers/worker.go\`` | `## Step 25: Create \`internal/handlers/worker.go\`` |

**Step 3: Update all import paths in code blocks**

For each handler code block, replace:
```go
	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/helpers"
	"github.com/justincordova/banana-farm-api/internal/models"
```

For the routes.go update sections, replace:
```go
	"github.com/justincordova/banana-farm-api/handlers"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/handlers"
```

**Step 4: Update the routes.go header references**

Any section that says "Update `routes/routes.go`" should say "Update `internal/routes/routes.go`".

**Step 5: Update the Phase 3 Checklist**

Replace all old-style paths in the checklist with their `internal/` equivalents. Pattern: any path like `models/X`, `handlers/X` → `internal/models/X`, `internal/handlers/X`.

**Step 6: Commit**

```bash
git add docs/plans/phase-3-remaining-entities.md
git commit -m "docs: update phase-3 plan to cmd/internal layout"
```

---

### Task 6: Update `docs/plans/phase-4-features.md`

**Files:**
- Modify: `docs/plans/phase-4-features.md`

Phase 4 adds `handlers/health.go`, `handlers/farm.go` stats, and filtering. Changes are similar to previous tasks.

**Step 1: Update Step 29 header**

Replace:
```
## Step 29: Create `handlers/health.go`
```

With:
```
## Step 29: Create `internal/handlers/health.go`
```

**Step 2: Update imports in the health.go code block**

Replace:
```go
	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/helpers"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/helpers"
```

**Step 3: Update Step 30 header reference**

Replace:
```
## Step 30: Add stats endpoint to Farm handler

Add this method to `handlers/farm.go`:
```

With:
```
## Step 30: Add stats endpoint to Farm handler

Add this method to `internal/handlers/farm.go`:
```

**Step 4: Update routes.go header references in this phase**

Any reference to "Update `routes/routes.go`" → "Update `internal/routes/routes.go`".

**Step 5: Update main.go references for health handler wiring**

Any reference to `cmd/api/main.go` when passing `startTime` — the path should already say `cmd/api/main.go` if phases were done in order, but if Phase 4 references `main.go` directly, update it.

**Step 6: Update the Phase 4 Checklist**

Replace all old-style paths with `internal/` prefixed equivalents.

**Step 7: Commit**

```bash
git add docs/plans/phase-4-features.md
git commit -m "docs: update phase-4 plan to cmd/internal layout"
```

---

### Task 7: Update `docs/plans/phase-5-tests.md`

**Files:**
- Modify: `docs/plans/phase-5-tests.md`

Phase 5 is the largest file (~1852 lines). Test files live next to their source files, so the test paths also move into `internal/`.

**Step 1: Update Step 33 header and test file path**

Replace:
```
### Create `handlers/test_helpers_test.go`
```

With:
```
### Create `internal/handlers/test_helpers_test.go`
```

**Step 2: Update imports in test_helpers_test.go code block**

Replace:
```go
	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/models"
	"github.com/justincordova/banana-farm-api/routes"
```

With:
```go
	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/models"
	"github.com/justincordova/banana-farm-api/internal/routes"
```

**Step 3: Update all test file headers throughout the document**

For every `### Create \`handlers/X_test.go\`` section, change to `### Create \`internal/handlers/X_test.go\``.

Common patterns to find and replace:
- `` `handlers/farm_test.go` `` → `` `internal/handlers/farm_test.go` ``
- `` `handlers/banana_tree_test.go` `` → `` `internal/handlers/banana_tree_test.go` ``
- `` `handlers/bunch_test.go` `` → `` `internal/handlers/bunch_test.go` ``
- `` `handlers/banana_test.go` `` → `` `internal/handlers/banana_test.go` ``
- `` `handlers/tool_test.go` `` → `` `internal/handlers/tool_test.go` ``
- `` `handlers/worker_test.go` `` → `` `internal/handlers/worker_test.go` ``
- `` `middleware/logging_test.go` `` → `` `internal/middleware/logging_test.go` ``
- `` `helpers/pagination_test.go` `` → `` `internal/helpers/pagination_test.go` ``

**Step 4: Update imports in every test file code block**

For each test code block that imports from this module, replace:
```go
	"github.com/justincordova/banana-farm-api/helpers"
	"github.com/justincordova/banana-farm-api/models"
	"github.com/justincordova/banana-farm-api/config"
	"github.com/justincordova/banana-farm-api/routes"
	"github.com/justincordova/banana-farm-api/middleware"
```

With their `internal/` equivalents:
```go
	"github.com/justincordova/banana-farm-api/internal/helpers"
	"github.com/justincordova/banana-farm-api/internal/models"
	"github.com/justincordova/banana-farm-api/internal/config"
	"github.com/justincordova/banana-farm-api/internal/routes"
	"github.com/justincordova/banana-farm-api/internal/middleware"
```

**Step 5: Update `go test` run commands for specific packages**

Replace:
```bash
go test ./handlers/...     # run handler tests only
```

With:
```bash
go test ./internal/handlers/...     # run handler tests only
```

Note: `go test ./...` is unchanged — it still runs all tests recursively.

**Step 6: Update the Phase 5 Checklist**

Replace all old-style paths with `internal/` prefixed equivalents.

**Step 7: Commit**

```bash
git add docs/plans/phase-5-tests.md
git commit -m "docs: update phase-5 plan to cmd/internal layout"
```

---

### Task 8: Final verification

**Step 1: Verify directory structure**

```bash
ls internal/config/
# Expected: config.go  logger.go
ls config/
# Expected: error — directory should not exist
```

**Step 2: Verify no old-style import paths remain in docs**

```bash
grep -r "banana-farm-api/config" docs/
grep -r "banana-farm-api/database" docs/
grep -r "banana-farm-api/models" docs/
grep -r "banana-farm-api/handlers" docs/
grep -r "banana-farm-api/routes" docs/
grep -r "banana-farm-api/middleware" docs/
grep -r "banana-farm-api/helpers" docs/
```

All output should be empty (no matches without `internal/` in the path).

**Step 3: Verify no old-style file paths remain in docs**

```bash
grep -rn "## Step.*Create \`config/" docs/
grep -rn "## Step.*Create \`database/" docs/
grep -rn "## Step.*Create \`models/" docs/
grep -rn "## Step.*Create \`handlers/" docs/
grep -rn "## Step.*Create \`routes/" docs/
grep -rn "## Step.*Create \`middleware/" docs/
grep -rn "## Step.*Create \`helpers/" docs/
grep -n "go run \." docs/
grep -n "go build.*\.\b" docs/
```

All output should be empty.

**Step 4: Verify go.mod is still correct (no changes needed)**

```bash
head -3 go.mod
# Expected: module github.com/justincordova/banana-farm-api
```

The module name itself doesn't change — only import paths within the project add `/internal/`.

**Step 5: Final commit if anything was missed**

```bash
git status
# If any files are modified, add and commit them
git add -A
git commit -m "docs: fix any remaining path references in plan docs"
```
