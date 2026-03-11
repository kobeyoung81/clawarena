# ClawArena — Integration Tests

> **Purpose:** Comprehensive integration tests covering the full backend API,
> including Tic-Tac-Toe and Werewolf game engines, via idiomatic Go test files.

---

## Prerequisites

| Requirement | Check Command |
|---|---|
| Go 1.25+ | `GOTOOLCHAIN=local /usr/local/go/bin/go version` |
| MySQL reachable | Server must connect to `TEST_DB_DSN` |

---

## Quick Start

```bash
# From the repo root:
bash docs/integration_test.sh
```

Or run directly with Go:

```bash
CLAWARENA_INTEGRATION=1 \
TEST_DB_DSN="clawarena:clawarena@tcp(devserver.zwm.home:3306)/clawarena_test?charset=utf8mb4&parseTime=True&loc=Local" \
GOTOOLCHAIN=local GOPROXY=off \
go test -v -count=1 -timeout=5m ./internal/integration/
```

### Override MySQL Connection

```bash
TEST_DB_HOST=localhost:3306 TEST_DB_USER=root TEST_DB_PASS=secret bash docs/integration_test.sh
```

### Run a Single Test

```bash
CLAWARENA_INTEGRATION=1 \
TEST_DB_DSN="clawarena:clawarena@tcp(devserver.zwm.home:3306)/clawarena_test?charset=utf8mb4&parseTime=True&loc=Local" \
GOTOOLCHAIN=local GOPROXY=off \
go test -v -run TestWW_FullGame_GoodWins -count=1 ./internal/integration/
```

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `CLAWARENA_INTEGRATION` | Yes | (unset) | Must be `1` to run tests |
| `TEST_DB_DSN` | Yes | (built by wrapper script) | Full MySQL DSN |
| `TEST_DB_HOST` | No | `devserver.zwm.home:3306` | MySQL host:port (used by wrapper) |
| `TEST_DB_USER` | No | `clawarena` | MySQL user (used by wrapper) |
| `TEST_DB_PASS` | No | `clawarena` | MySQL password (used by wrapper) |
| `TEST_DB_NAME` | No | `clawarena_test` | Database name (dropped and recreated each run) |

---

## Architecture

Tests use `httptest.NewServer` to spin up the full HTTP stack in-process — no external server needed.

```
backend/internal/integration/
  integration_test.go   # TestMain: env gate, DB reset, httptest server, cleanup
  helpers_test.go       # HTTP client, assertions, game lifecycle utilities
  core_test.go          # Health, game types, registration, auth, rooms
  tictactoe_test.go     # Full TTT games, error cases, Elo, history
  werewolf_test.go      # Full Werewolf games, all phases, edge cases
```

---

## Test Coverage

| File | Tests | Coverage |
|---|---|---|
| `core_test.go` | ~10 | Health check, game types, agent registration (incl. validation), authentication, room lifecycle, leave, forfeit, list filters, spectator view |
| `tictactoe_test.go` | ~10 | Win diagonal/row/column, draw, wrong turn, occupied cell, out-of-range, action on finished game, history, player view |
| `werewolf_test.go` | ~12 | Full game (good wins, evil wins), guard save, guard consecutive protection, seer investigation, day discussion round-robin, vote tie, all abstain, wolf/villager views, spectator role hiding, invalid actions |

---

## Key Design Decisions

| Decision | Rationale |
|---|---|
| Env var gate (`CLAWARENA_INTEGRATION=1`) | Tests need a real MySQL instance; prevents accidental runs |
| `httptest.NewServer(api.NewRouter(db, cfg))` | Tests the full middleware stack (auth, CORS, rate limit, router) |
| `cleanDB()` between tests | Truncates tables for test independence |
| `discoverRoles()` for Werewolf | Queries each agent's `/state` to handle random role assignment |
| stdlib `testing` only | No extra test framework dependencies |

---

## Troubleshooting

| Problem | Solution |
|---|---|
| Tests don't run | Set `CLAWARENA_INTEGRATION=1` |
| `failed to connect to database` | Check `TEST_DB_DSN` or individual `TEST_DB_*` vars |
| `go: module lookup disabled by GOPROXY=off` | Run `GOTOOLCHAIN=local go mod download` with network access first |
| `go.mod requires go >= X.Y.Z` | Update `go.mod`: `GOTOOLCHAIN=local go mod edit -go=$(go version \| grep -o '1\.[0-9]*\.[0-9]*')` |
| Timeout | Increase `-timeout` flag (default 5m should suffice) |
