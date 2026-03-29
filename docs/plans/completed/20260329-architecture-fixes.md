# Architecture Fixes for rlb-stats

## Overview
Fix operational safety gaps and code quality issues discovered during architectural review.
Changes address: graceful shutdown with context propagation, data safety on exit (aggregator flush),
security cleanup (docker socket), API correctness, dead code removal, deprecated API cleanup,
and improved testability via aggregator interface extraction.

All changes go on the existing `ci-updates-and-hardening` branch as additional commits.

### Acceptance Criteria
- SIGTERM triggers clean shutdown with aggregator flush and bolt close
- API returns 400 (not 417) for parse errors
- no `ioutil` usage in any Go file
- no dead code (`LoadStream`)
- docker-compose has no socket mount
- `LogAggregator` interface enables mock-based testing of insert paths

## Context (from discovery)
- files/components involved: main.go, store/{bolt,store,aggregate,candle}.go, web/{server,helpers}.go, Dockerfile, docker-compose.yml, all test files
- the app has no graceful shutdown — SIGTERM hard-kills, losing the in-progress aggregator minute
- `LoadStream` on Bolt is dead code (not in Engine interface, never called via abstraction)
- `Aggregator` is a concrete type in Server, blocking mock-based testing of insert paths
- test files still use deprecated `ioutil` (go fix didn't touch them)
- docker-compose mounts docker socket with no code using it

## Development Approach
- **testing approach**: Regular (code first, then tests)
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- run `go test -race ./...`, `golangci-lint run`, `gofmt -w` after each task

## Testing Strategy
- **unit tests**: required for every task
- mock aggregator enables testing insert handler's save-error path without minute-boundary timing tricks
- Bolt.Close() tested via NewBolt/Close round-trip
- Aggregator.Flush() tested alongside existing Store() tests
- graceful shutdown tested via context cancellation

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues/blockers with ⚠️ prefix

## Implementation Steps

### Task 1: Add Bolt.Close() and fix NewBolt error return

**Files:**
- Modify: `app/store/bolt.go`
- Modify: `app/store/bolt_test.go`

- [x] add `Close() error` method to `Bolt` that calls `s.db.Close()`
- [x] fix `NewBolt` to return `nil, err` on both error paths (lines 27, 34) instead of `&result, err`
- [x] remove dead `_ = v` statement (line 80)
- [x] update `TestSaveAndLoadLogEntryBolt`: capture return value from broken-DB `NewBolt("/dev/null")` call and assert it is nil
- [x] add test for `Close()`: open a bolt, close it, verify no error
- [x] replace ALL `ioutil.TempFile` with `os.CreateTemp` in bolt_test.go (2 occurrences: lines 16 and 39), remove `"io/ioutil"` import
- [x] run tests — must pass before next task

### Task 2: Add Aggregator.Flush() method

**Files:**
- Modify: `app/store/aggregate.go`
- Modify: `app/store/aggregate_test.go`

- [x] add `Flush() (Candle, bool)` to `Aggregator` — if `len(p.entries) > 0`, build candle from buffered entries using same dedup logic as `Store`, clear entries, return `(candle, true)`; if empty return `(Candle{}, false)`
- [x] add `TestFlush` — store entries for one minute without triggering boundary, call `Flush()`, verify candle contains expected data
- [x] add test: `Flush()` on empty aggregator returns `(Candle{}, false)`
- [x] add test: `Flush()` after a `Store()` that already emitted a candle — verify only the trailing buffered entries are flushed
- [x] add test: after `Flush()`, subsequent `Flush()` returns false (buffer cleared)
- [x] run tests — must pass before next task

### Task 3: Graceful shutdown with context propagation

**Files:**
- Modify: `app/web/server.go`
- Modify: `app/web/server_test.go`
- Modify: `app/main.go`

- [x] change `Server.Run()` to `Server.Run(ctx context.Context)`: start `srv.ListenAndServe()` in a goroutine, block on `<-ctx.Done()`, then call `srv.Shutdown` with a 5-second timeout context
- [x] add test for `Run(ctx)`: create Server, call `Run(ctx)` with cancellable context in a goroutine, wait for server to be listening, cancel context, verify `Run` returns cleanly
- [x] in `main.go`: use `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)` to create cancellable context
- [x] store `*store.Bolt` (concrete type) in main so we can call `Close()` on shutdown
- [x] extract `*store.Aggregator` into a local variable `aggregator` in `main()`, pass it to `Server` struct, so it can be used in the shutdown sequence
- [x] after `webServer.Run(ctx)` returns: flush aggregator (call `Flush()`, save candle if ok), call `storage.Close()`, log clean exit
- [x] set `revision` to `"unknown"` if empty after ldflags
- [x] run tests — must pass before next task

### Task 4: Remove docker socket mount from docker-compose

**Files:**
- Modify: `docker-compose.yml`

- [x] remove `/var/run/docker.sock:/var/run/docker.sock` volume mount from rlb-stats service
- [x] no tests needed (infrastructure only)

### Task 5: HTTP 417 → 400 for parse errors

**Files:**
- Modify: `app/web/server.go`
- Modify: `app/web/server_test.go`

- [x] replace all `http.StatusExpectationFailed` with `http.StatusBadRequest` in `getCandle` handler (5 occurrences in server.go: lines 89, 96, 105, 112, 126)
- [x] update test table in `TestServerAPI`: change expected `responseCode` from `http.StatusExpectationFailed` to `http.StatusBadRequest` for the 4 existing affected test cases (indices 1-4)
- [x] add test case for `files` parse error path (the 5th occurrence at line 126 is currently untested)
- [x] run tests — must pass before next task

### Task 6: Remove dead LoadStream, document bolt key format

**Files:**
- Modify: `app/store/bolt.go`
- Modify: `app/store/bolt_test.go`

- [x] remove `LoadStream` method entirely from bolt.go
- [x] remove `TestBolt_LoadStream` from bolt_test.go
- [x] add comment above `Save` method explaining decimal key format: keys are decimal Unix timestamps; lexicographic ordering matches numeric ordering because all timestamps since 1973 are 10 digits (until 2286)
- [x] run tests — must pass before next task

### Task 7: Dockerfile fixes

**Files:**
- Modify: `Dockerfile`

- [x] change `ADD . /app` to `COPY . /app`
- [x] change `ADD webapp /srv/webapp` to `COPY webapp /srv/webapp`
- [x] remove `CMD ["/srv/rlb-stats"]` line — ENTRYPOINT already includes the binary path, and having both causes the binary path to be passed as an extra argument to init.sh
- [x] no Go tests needed (Docker only)

### Task 8: Fix deprecated ioutil in web test files

**Files:**
- Modify: `app/web/server_test.go`
- Modify: `app/web/helpers_test.go`

- [x] in `server_test.go`: replace `ioutil.ReadAll` with `io.ReadAll` (2 occurrences), remove `"io/ioutil"` from imports
- [x] in `helpers_test.go`: replace `ioutil.TempFile` with `os.CreateTemp`, remove `"io/ioutil"` from imports
- [x] fix test bug at `server_test.go:124-125`: replace `require.Nil(t, string(body), "problem parsing response body")` with `require.NoError(t, err, "problem parsing response body: %s", string(body))`
- [x] run tests — must pass before next task

### Task 9: Extract LogAggregator interface and add mock-based insert tests

**Files:**
- Modify: `app/web/server.go`
- Modify: `app/web/helpers.go`
- Modify: `app/web/helpers_test.go`
- Modify: `app/web/server_test.go`

- [x] define `LogAggregator` interface in `server.go` (consumer package): `type LogAggregator interface { Store(store.LogRecord) (store.Candle, bool) }`
- [x] change `Server.Aggregator` field type from `*store.Aggregator` to `LogAggregator`
- [x] update `saveLogRecord` signature: change `parser *store.Aggregator` to `parser LogAggregator`
- [x] create `mockAggregator` in test file with configurable return values: fields for `candle store.Candle` and `ok bool` (error testing is done via mock Engine's `Save`, not the aggregator)
- [x] update `startupT` to still use `&store.Aggregator{}` as the real implementation (interface satisfied implicitly)
- [x] add test: insert with mock aggregator returning `ok=true` and working engine — verify 200 and candle saved
- [x] add test: insert with mock aggregator returning `ok=true` and failing engine — verify 500 error response
- [x] add test: insert with mock aggregator returning `ok=false` — verify 200 and no save attempted
- [x] run tests — must pass before next task

### Task 10: Verify acceptance criteria

- [x] verify all acceptance criteria from overview are met
- [x] verify edge cases are handled
- [x] run full test suite: `go test -race ./...`
- [x] run linter: `golangci-lint run`
- [x] run `gofmt -w` on all modified Go files
- [x] run `go fix ./...` on modified packages

### Task 11: [Final] Commit and update PR

- [ ] commit changes (logical grouping: safety, API, cleanup, docker, tests)
- [ ] push to branch and update PR description
- [ ] move this plan to `docs/plans/completed/`

## Technical Details

### Graceful shutdown flow (main.go)
```
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer cancel()
aggregator := &store.Aggregator{}
...
webServer.Run(ctx)  // blocks until ctx cancelled
// shutdown sequence:
if candle, ok := aggregator.Flush(); ok {
    storage.Save(candle)
}
storage.Close()
```

### Server.Run with context
```go
func (s *Server) Run(ctx context.Context) {
    srv := &http.Server{...}
    go func() { ... srv.ListenAndServe() ... }()
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    srv.Shutdown(shutdownCtx)
}
```

### LogAggregator interface (defined in web package, consumer side)
```go
type LogAggregator interface {
    Store(store.LogRecord) (store.Candle, bool)
}
```
`*store.Aggregator` satisfies this implicitly — no changes to store package needed.

### Mock aggregator for tests
```go
type mockAggregator struct {
    candle store.Candle
    ok     bool
}

func (m *mockAggregator) Store(store.LogRecord) (store.Candle, bool) {
    return m.candle, m.ok
}
```

## Post-Completion

**Manual verification:**
- test `docker-compose up` still works without docker socket mount
- verify `docker stop rlb-stats` triggers clean shutdown (check logs for flush/close messages)
- verify API returns 400 (not 417) for parse errors
