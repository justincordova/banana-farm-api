# Design: Refactor to `cmd/` + `internal/` Layout

**Date:** 2026-03-03

## Problem

The current plan docs (and the existing `config/` directory) place all packages at the module root. The Go standard layout places application code inside `internal/` (preventing external import) and the entry point inside `cmd/<name>/`. The plans need to reflect this before any further implementation phases are executed.

## Decision

Adopt the standard Go project layout:

- `cmd/api/main.go` — entry point only
- `internal/` — all application packages

## New Directory Structure

```
banana-farm-api/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── logger.go
│   ├── database/
│   │   └── database.go
│   ├── models/
│   │   ├── farm.go
│   │   ├── banana_tree.go
│   │   └── ...
│   ├── handlers/
│   │   ├── farm.go
│   │   ├── farm_test.go
│   │   └── ...
│   ├── routes/
│   │   └── routes.go
│   ├── middleware/
│   │   ├── logging.go
│   │   ├── logging_test.go
│   │   └── error.go
│   └── helpers/
│       ├── response.go
│       ├── pagination.go
│       └── pagination_test.go
├── .env
├── .env.example
├── go.mod
├── .air.toml
└── README.md
```

## Import Path Changes

Every internal package import adds `/internal/` after the module name:

| Before | After |
|--------|-------|
| `banana-farm-api/config` | `banana-farm-api/internal/config` |
| `banana-farm-api/database` | `banana-farm-api/internal/database` |
| `banana-farm-api/models` | `banana-farm-api/internal/models` |
| `banana-farm-api/handlers` | `banana-farm-api/internal/handlers` |
| `banana-farm-api/routes` | `banana-farm-api/internal/routes` |
| `banana-farm-api/middleware` | `banana-farm-api/internal/middleware` |
| `banana-farm-api/helpers` | `banana-farm-api/internal/helpers` |

## Affected Files

### Physical files
- `config/config.go` → `internal/config/config.go`
- `config/logger.go` → `internal/config/logger.go`
- `config/` directory removed after move

### Documentation
- `docs/PLAN.md` — project structure tree + build order
- `docs/plans/phase-1-foundation.md` — file paths, `main.go` location, `.air.toml` build cmd
- `docs/plans/phase-2-first-entity.md` — all file paths + import paths in code blocks
- `docs/plans/phase-3-remaining-entities.md` — same
- `docs/plans/phase-4-features.md` — same
- `docs/plans/phase-5-tests.md` — same

## Key Command Changes

| Context | Before | After |
|---------|--------|-------|
| Run server | `go run .` | `go run ./cmd/api` |
| Air build cmd | `go build -o ./tmp/main .` | `go build -o ./tmp/main ./cmd/api` |
| Tests | `go test ./...` | `go test ./...` (unchanged) |
