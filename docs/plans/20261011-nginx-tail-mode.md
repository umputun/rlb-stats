# Nginx Tail Mode — Ingest from Access Log

## Overview

Decouple rlb-stats from rlb. Instead of only accepting records via `POST /api/insert` from
rlb, add a built-in nginx access log tailer so rlb-stats can be deployed next to any nginx
and produce download stats directly from its access log. Keep `/api/insert` working for
back-compat.

**Problems solved:**
- rlb is effectively retired at the main deployment (radio-t); rlb-stats has no live source
- nginx access log already contains everything needed and more (bytes transferred, status
  code, country via Cloudflare)
- current pipeline has no filtering, so every record counts as a download — fine with
  rlb, broken with raw nginx (304s, API endpoints, bots)

**Key design decisions:**
- parser hardcoded to one `log_format` (umputun's current one plus three new CF fields);
  configurable patterns deferred
- the regex REQUIRES the three trailing CF fields; old-format lines (umputun's current
  11-field lines without CF fields) fail parse at DEBUG level — a soft failure keeps the
  nginx update ordered after the rlb-stats deploy without hard-breaking
- in-house tailer (polling, inode-based rotation detection), no new dependency
- real client IP: prefer `$http_cf_connecting_ip` and fall back to `$remote_addr` so the
  tool works with or without Cloudflare in front
- add `Bytes int64` and `Countries map[string]int` to `store.Info`, additive and
  backward-compatible with existing boltdb files (JSON unmarshal of old candles gives
  zero/nil for new fields); `Bytes` is written to BOTH the `DestHost` node AND `AllNode`
  (per-host bandwidth is useful), `Countries` only on `AllNode`
- `$http_range` is parsed and discarded for now — kept in the log_format so the format
  does not need another change later when probe-vs-play analysis is added
- filtering before Aggregator: method allowlist (GET only, by default; HEAD probes
  from Apple Podcasts log as 200 with 0 bytes and would inflate counts), status
  allowlist, path glob allowlist, host allowlist
- dedup unchanged for Volume/Files; bytes sum across all deduplicated entries so a single
  listener's range-request burst produces one "download" but accurate bandwidth
- **"download" semantics shift from rlb**: one rlb redirect = one download regardless of
  stream duration. With nginx tail, a listener streaming via `<audio>` for an hour
  produces up to ~60 downloads (unique file+ip per minute). Apple Podcasts fetches in
  a few large ranges up front so inflates less. Historical rlb-stats numbers are not
  comparable to nginx-tail-mode numbers; call this out in the PR body
- docker-compose: drop the `rlb` service block and `rlb.config.yml`; add a sample nginx
  volume mount for the tail path

## Context

- Module: `github.com/umputun/rlb-stats`, Go 1.25
- Insert path: `app/web/server.go:320` → `saveLogRecord` → `Aggregator.Store` → `Engine.Save`
- Aggregator: `app/store/aggregate.go` — minute-truncates, dedups `(FileName, FromIP)`
- Candle shape: `app/store/candle.go:11` — `Candle.Nodes[host].{Volume, Files}` with
  `"all"` as the synthetic cross-host node; bolt storage is JSON per minute
- Dashboard: `app/web/dashboard.go` — summaries, histogram, top files, nodes, heatmap
- umputun's current nginx log sample (from Telegram 2026-04-26 msg 496162):

      108.162.220.92 - - [26/Apr/2026:21:52:30 +0000] "GET /p/... HTTP/2.0" 304 0 \
      "radio-t.com" "https://radio-t.com/" "Mozilla/5.0 ..." https

  implied log_format:

      $remote_addr - - [$time_local] "$request" $status $body_bytes_sent \
      "$http_host" "$http_referer" "$http_user_agent" $scheme

- umputun agreed (msg 496165) to add the three CF fields to the format

## Development Approach

- Testing: table-driven, `stretchr/testify`, same style as existing packages
- Every task includes new/updated tests; `go test -race ./...` must pass before next task
- `golangci-lint run` clean before commit
- Backward compatibility: `/api/insert`, `/api/candle`, existing boltdb files

## Testing Strategy

- Unit: parser (every line of umputun's sample + synthetic edge cases), filters, tailer
  rotation
- Integration: feed a temp log file to a tail goroutine, verify candles land in BoltDB
  with expected `Volume`, `Bytes`, `Countries`
- Regression assertion for the e2e task: parsing umputun's seven real sample lines with
  default filters (status 200,206; path *.mp3) yields exactly 2 downloads
  (/rt_podcast1010.mp3 twice from two IPs), total bytes = 2097152 + 139264 = 2236416,
  and dropping any one field from the parser must fail at least one assertion

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered items with + prefix
- Document blockers with ! prefix

## Implementation Steps

### Task 1: Extend storage schema with Bytes and Countries

**Files:**
- Modify: `app/store/candle.go`
- Modify: `app/store/candle_test.go`
- Modify: `app/store/aggregate.go` (byte summing)
- Modify: `app/store/aggregate_test.go`

- [ ] add `Bytes int64` and `Countries map[string]int` to `Info` struct
- [ ] extend `LogRecord` with `Bytes int64` and `Country string`
- [ ] update `NewInfo()` to initialise `Countries` as empty map (nil-safe JSON unmarshal
  of old candles still works — reads give zero/nil; writes always init)
- [ ] extend `Update(LogRecord)` to add `Bytes` to BOTH the `DestHost` node AND `AllNode`
  (same path as `Volume`); increment `Countries[Country]` on `AllNode` only when Country
  is non-empty
- [ ] add `AddBytes(LogRecord)` method on `Candle` for deduplicated-entry bytes-only
  merge (adds to both nodes, same as `Update`'s bytes path)
- [ ] change `buildCandle` so the first (file, ip) entry calls `Update`, subsequent
  duplicates call `AddBytes` only — tests assert Volume=1 and Bytes=sum
- [ ] extend `mergeCandles` to sum `Bytes` and merge `Countries` map
- [ ] run tests

### Task 2: Add nginx log parser

**Files:**
- Create: `app/nginx/parser.go`
- Create: `app/nginx/parser_test.go`

- [ ] package `nginx`; `ParseLine(line string) (store.LogRecord, bool, error)` — bool is
  "parsed OK but filtered out" vs error = malformed line
- [ ] parse the fixed log_format:

      $remote_addr - - [$time_local] "$request" $status $body_bytes_sent \
      "$http_host" "$http_referer" "$http_user_agent" $scheme \
      "$http_cf_connecting_ip" "$http_cf_ipcountry" "$http_range"

- [ ] use `regexp` with named groups; one compiled regex at package init; the three
  trailing CF fields are REQUIRED by the regex (old-format lines must fail parse so they
  log at DEBUG rather than silently matching with empty CF fields)
- [ ] parse `$time_local` with layout `"02/Jan/2006:15:04:05 -0700"`
- [ ] extract method and path from `$request` by splitting on space (`GET /foo HTTP/2.0`
  → method=GET, path=/foo); strip query string (`?...`) — podcast stats don't need
  query vars
- [ ] client IP: `$http_cf_connecting_ip` if non-empty and not `"-"`, else `$remote_addr`
- [ ] country: `$http_cf_ipcountry` if non-empty and not `"-"` or `"XX"`, else empty
- [ ] bytes: parse `$body_bytes_sent` as int64; status as int
- [ ] tests:
  - every one of umputun's seven real log lines WITH the three CF fields appended
  - synthetic lines missing fields
  - malformed lines return error without panicking
  - **negative**: one of umputun's unmodified real log lines (no CF fields) must return
    a parse error — proves the regex requires the trailing fields

### Task 3: Add filter config

**Files:**
- Create: `app/nginx/filter.go`
- Create: `app/nginx/filter_test.go`

- [ ] `Filter` struct with Methods (set[string]), Status (set[int]), Paths ([]glob),
  Hosts (set[string])
- [ ] `(Filter) Keep(method string, status int, path, host string) bool`
- [ ] empty Methods means "GET only" (hard default — HEAD probes from podcast clients
  must not count as downloads); empty Status/Hosts means "all"; empty Paths means "all"
- [ ] Path glob: `path/filepath.Match` on the final URL path component; `*.mp3` matches
  `/rt_podcast1010.mp3`; `*` matches everything
- [ ] go-flags slice tag `env-delim:","` for comma-separated flag values:
  `--nginx-status 200,206`, `--nginx-path '*.mp3'` (repeat or comma-separate),
  `--nginx-host cdn.radio-t.com` (repeat or comma-separate),
  `--nginx-method GET` (default `GET`)
- [ ] tests: table of (filter config, method, status, path, host) → kept/dropped;
  explicit test for HEAD /rt_podcast1010.mp3 status=200 → dropped under default filter

### Task 4: Add file tailer

**Files:**
- Create: `app/nginx/tail.go`
- Create: `app/nginx/tail_test.go`

- [ ] `Tailer` struct opens a file, seeks to end on first open (configurable:
  `fromStart bool` for e2e tests), reads full lines via `bufio.Scanner`
- [ ] rotation detection: `os.Stat` every 500ms; reopen on inode change (same path,
  different inode via `syscall.Stat_t.Ino`) or on size decrease (truncate)
- [ ] handles partial final line by not emitting until newline arrives
- [ ] `Run(ctx context.Context, out chan<- string)` loops until ctx cancelled; sends
  one full line per iteration
- [ ] cross-platform: use `os.Stat` → `Sys().(*syscall.Stat_t).Ino` on Linux + Darwin
- [ ] tests: write lines to a tempfile, verify Tailer emits them; truncate, verify
  reopen; rotate (rename + create new), verify switch
- [ ] bounded line buffer: fall back to a plain `bufio.Reader` with explicit `ReadString`
  when a line exceeds Scanner's default 64 KiB (nginx access log lines in the wild can
  grow with long user agents + headers)

### Task 5: Wire tailer into main

**Files:**
- Modify: `app/main.go`
- Modify: `app/web/server.go` (nothing — keep `/api/insert` as-is)

- [ ] new flags on `opts`:
  - `NginxLog    string   `  — `--nginx-log /path/to/access.log`
  - `NginxMethod []string `  — `--nginx-method GET` (default `GET`)
  - `NginxStatus []int    `  — `--nginx-status 200,206` (default `200,206`)
  - `NginxPath   []string `  — `--nginx-path '*.mp3'` (default `*.mp3`)
  - `NginxHost   []string `  — `--nginx-host cdn.radio-t.com` (default empty = all)
- [ ] when `NginxLog` is set, start the pipeline before `webServer.Run`:

      var wg sync.WaitGroup
      lines := make(chan string, 256)
      wg.Add(1)
      go func() { defer wg.Done(); Tailer{...}.Run(ctx, lines) }() // tailer closes lines on ctx done
      wg.Add(1)
      go func() { defer wg.Done(); consumeLines(lines, filter, aggregator, storage) }()

- [ ] tailer MUST close `lines` on ctx cancel so the consumer's `for range` exits
- [ ] consumer: parse → filter → `saveLogRecord(storage, aggregator, record)`; log parse
  errors at DEBUG only
- [ ] **shutdown ordering (closes advisor's race)**: after `webServer.Run(ctx)` returns,
  `wg.Wait()` before the existing `aggregator.Flush()` call, so no in-flight line is
  lost on SIGTERM
- [ ] test: construct a stub Engine, feed sample lines through consumer, assert candles
  saved (combined with Task 8's e2e test)

### Task 6: Dashboard additions

**Files:**
- Modify: `app/web/dashboard.go`
- Modify: `app/web/templates/partials/summary.html`
- Modify: `app/web/templates/partials/files.html`
- Create: `app/web/templates/partials/countries.html`
- Modify: `app/web/templates/dashboard.html`
- Modify: `app/web/dashboard_test.go`

- [ ] `SummaryData` gets `Bytes int64` + a `HumanBytes` helper (KB/MB/GB/TB)
- [ ] `FileStats` gets `Bytes int64`
- [ ] new `CountryStats` struct + `computeTopCountries(candles, limit int)` summing
  `Countries` from AllNode
- [ ] `computeSummary` returns `(count int, bytes int64)`
- [ ] summary cards: show count + humanised bytes side by side
- [ ] top files table: new "Bandwidth" column
- [ ] new "Top countries" panel below nodes table, CSS bars same style
- [ ] tests update for the new return types

### Task 7: Strip rlb from docker-compose and README

**Files:**
- Modify: `docker-compose.yml`
- Delete: `rlb.config.yml`
- Modify: `README.md`

- [ ] remove the `rlb:` service block, keep only `rlb-stats`
- [ ] add commented-out example of mounting `/var/log/nginx/access.log` read-only and
  setting `--nginx-log` / env vars
- [ ] README rewrite "Run instructions":
  - new section "Nginx tail mode" with the required `log_format` directive (named
    `rlbstats`) and the `access_log /var/log/nginx/access.log rlbstats;` line
  - note that `$http_cf_connecting_ip` and `$http_cf_ipcountry` require Cloudflare IP
    geolocation enabled; without CF they just stay empty and `$remote_addr` is used
  - drop the rlb curl loop
  - add the new flags to the parameters table
- [ ] README's `POST /api/insert` section stays — it's still supported

### Task 8: End-to-end verification

- [ ] write umputun's seven real sample lines (from Telegram msg 496162, stripping the
  terminal-paste `^[` artefact from one line) plus the three CF fields appended with
  plausible values into `app/nginx/testdata/sample.log`
- [ ] **regression trap**: give the two mp3 lines the SAME `$remote_addr` (a CF edge
  IP like `108.162.220.92`) but DIFFERENT `$http_cf_connecting_ip` values (e.g.
  `31.46.245.136` and `109.196.194.69`). If the parser falls back to `$remote_addr`,
  dedup collapses the two mp3 downloads to one and the test fails — proves CF IP is
  used for client identification
- [ ] a `TestEndToEnd` that:
      1. boots consumer pipeline on an in-memory `store.Engine` with the real `Aggregator`
      2. feeds every line from the sample file through the consumer channel
      3. cancels ctx; `wg.Wait()` for consumer; `aggregator.Flush()`; save returned candle
      4. `Load` from the engine and assert on the resulting candle(s)
   (do not rely on minute-change to emit — all seven lines are in minute 21:52; the
   aggregator only emits on minute boundary, so the flush-on-shutdown path is the one
   under test, which also exercises the shutdown ordering from Task 5)
- [ ] assertions:
  - 2 downloads counted (the two .mp3 lines; everything else filtered)
  - bytes = 2097152 + 139264 = 2236416
  - countries map contains both country codes seeded in testdata, each with count 1
  - the 304 html pages and the /site-api/ json call do not appear in Files
- [ ] regression check: temporarily removing `$http_cf_connecting_ip` from the parsed
  regex must break the test — proves the assertion actually touches that field

### Task 9: PR via ship-a-pr workflow

- [ ] run revmux review per project policy
- [ ] address findings
- [ ] push `nginx-tail-mode`
- [ ] `gh pr create` with description linking to this plan

## Out of scope (deferred)

- Configurable `log_format` strings — hardcoded for now
- `$http_range` probe-vs-play analysis (field is logged, parsed, discarded)
- User-Agent classification (bots vs clients vs podcatchers)
- Status-code split per candle (200/206/304 breakdown)
- Per-host file breakdown (today `Files` only lives on `AllNode`)
- Sub-minute aggregation
