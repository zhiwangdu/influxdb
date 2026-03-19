# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

InfluxDB is an open source time series database. This is the `master-1.x` branch for InfluxDB 1.x. The `main` branch targets InfluxDB 3.x (not ready for production).

## Build Commands

```bash
# Build all packages
go build ./...

# Build and install binaries (to $GOPATH/bin)
export PKG_CONFIG="$(git rev-parse --show-toplevel)/pkg-config.sh"
go install ./...

# Build with version info
go install -ldflags="-X main.version=$VERSION -X main.branch=$BRANCH -X main.commit=$COMMIT" ./...
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run specific test
go test -run=TestName ./path/to/package

# Run tests with race detection
go test -race ./...

# Run tests and show coverage
go test -coverprofile /tmp/cover . && go tool cover -html /tmp/cover

# CI runs 3 variants: inmem index, tsi1 index, and race detection
# Local equivalent: set INFLUXDB_DATA_INDEX_VERSION to "inmem" or "tsi1"
```

## Code Generation

```bash
# Regenerate all generated code (protobuf, templates)
go generate ./...
```

Generated files (*.gen.go) are created from templates (*.tmpl). Do not edit *.gen.go directly—edit the corresponding .tmpl file instead.

## Key Architecture

### Storage Engine (`tsdb/`)
- `tsdb/engine.go` - Core storage engine interface
- `tsdb/index/tsi1/` - TSI (Time Series Index) - disk-based index
- `tsdb/index/inmem/` - In-memory index alternative
- `tsdb/shard.go` - Shard management
- `tsdb/store.go` - Store for managing shards on disk

### Query Processing
- `query/compile.go` - InfluxQL query compilation
- `query/executor.go` - Query execution engine
- `coordinator/` - Query coordination and shard routing
- `storage/reads/` - Storage-level read operations and cursors

### HTTP API (`services/httpd/`)
- `services/httpd/handler.go` - Main HTTP request handler
- `services/httpd/service.go` - HTTP service configuration

### Server (`cmd/influxd/`)
- `cmd/influxd/main.go` - Entry point, dispatches to subcommands
- `cmd/influxd/run/` - `influxd run` command - starts the server
- `cmd/influxd/backup/` - `influxd backup` command
- `cmd/influxd/restore/` - `influxd restore` command

### Data Models
- `models/points.go` - Time series points encoding/decoding
- `models/rows.go` - Query result rows
- `internal/meta_client.go` - Metadata client for cluster coordination

## Development Guidelines

- **Pre-commit hook**: Copy `.hooks/pre-commit` to `.git/hooks/` to run formatting checks
- **Format code**: `goimports -w ./` and `go vet ./...`
- **No third-party libraries**: Prefer standard library; third-party deps are minimized
- **Always include default case** in switch statements
- **Use defer with anonymous functions** for complex lock handling

## Important Files

- `Makefile` - Simple wrapper for `go build ./...`
- `go.mod` / `go.sum` - Go module dependencies
- `.circleci/config.yml` - CI configuration (unit_test_inmem, unit_test_tsi1, unit_test_race)
- `test-flux.sh` - Flux query engine tests
