# Dashboard Rewrite — SSR with HTMX + Picocss + ECharts

## Overview

Complete rewrite of the rlb-stats dashboard from a client-side SPA (4 chart libraries, ~900KB JS) to server-side rendered HTML with HTMX for interactivity, Picocss for styling, and ECharts as the single chart library.

**Problems solved:**
- Current UI is "not too helpful" (issue #23) — only two line charts, no summary stats
- Licence issues: AnyChart is proprietary, ApexCharts changed from MIT to revenue-gated
- No all-time or per-episode download totals (issue #11)
- `max_points` query parameter capped at 127 due to int8 parsing bug

**Key design decisions (from brainstorm):**
- CSS bars for ranked lists/tables, ECharts only for time-series histogram + peak hours heatmap
- `html/template` with `embed.FS`, no external templating deps
- Picocss CDN for base styling (includes dark mode)
- HTMX for period switching — single `/fragment/dashboard?period=...` endpoint
- Single scrollable page, all sections visible
- Existing `/api/candle` and `/api/insert` endpoints preserved untouched

**Addresses:** #23 (UI rewrite), #11 items 1-3 (totals, per-episode stats), max_points bug

## Context

- Module: `github.com/umputun/rlb-stats`, Go 1.25
- Engine interface: `app/store/store.go` — `Save()` and `Load()` only
- BoltDB impl: `app/store/bolt.go` — cursor-based range scans, key = Unix timestamp string
- Server: `app/web/server.go` — `routegroup` routing, `rest` middleware
- max_points bug: `app/web/server.go:130` — `strconv.ParseInt(n, 10, 8)`
- Current frontend: `webapp/` — index.html + 6 JS files (index, data, anycharts, apexcharts, echarts, taucharts)
- Data model: per-minute `Candle` with per-node `Volume` and per-file counts on `"all"` node

## Development Approach

- **Testing approach**: regular (code first, then tests)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- Run `go test -race ./...` after each change
- Run `golangci-lint run` before committing
- Maintain backward compatibility for `/api/candle` and `/api/insert`

## Testing Strategy

- **Unit tests**: required for every task — table-driven with `stretchr/testify`
- **Integration tests**: HTTP handler tests using `httptest.NewServer` (existing pattern)
- **No e2e tests**: project has no Playwright/Cypress setup

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Implementation Steps

### Task 1: Extend Engine interface with TimeRange

**Files:**
- Modify: `app/store/store.go`
- Modify: `app/store/bolt.go`
- Modify: `app/store/bolt_test.go`

- [x] add `TimeRange(ctx context.Context) (oldest, newest time.Time, err error)` to `Engine` interface in `app/store/store.go`
- [x] implement `TimeRange` on `*Bolt` in `app/store/bolt.go` using BoltDB cursor `First()` and `Last()` on the `"stats"` bucket
- [x] handle empty bucket case — return `time.Time{}` zero values with no error
- [x] write tests for `TimeRange`: non-empty DB returns correct oldest/newest timestamps
- [x] write tests for `TimeRange`: empty DB returns zero times
- [x] add `TimeRange` method to `MockDB` and `goodDB` in `app/web/helpers_test.go` (required — interface change causes compile failure otherwise)
- [x] run tests — must pass before next task

### Task 2: Fix max_points int8 parsing bug

**Files:**
- Modify: `app/web/server.go`
- Modify: `app/web/server_test.go`

- [ ] change `strconv.ParseInt(n, 10, 8)` to `strconv.ParseInt(n, 10, 64)` at `app/web/server.go:130`
- [ ] update existing test case at `server_test.go:76` that expects `max_points=256` to return 400 — it should now succeed
- [ ] write test for `max_points=200` returning valid response
- [ ] run tests — must pass before next task

### Task 3a: Dashboard data computation functions

**Files:**
- Create: `app/web/dashboard.go`
- Create: `app/web/dashboard_test.go`

Server-side functions that transform candles into view-ready data structures.

- [ ] define view data structs: `DashboardData`, `SummaryData` (period label + total count), `FileStats` (name, count, percent of max), `NodeStats` (name, volume, percent of max), `HeatmapCell` (hour, weekday, value)
- [ ] implement `computeSummary(candles []store.Candle) int` — sums `"all"` node Volume across candles; the handler calls this 5 times with different candle slices (1h, 24h, 1w, 1m, all) and assembles `[]SummaryData` with period labels
- [ ] implement `computeTopFiles(candles []store.Candle, limit int) []FileStats` — aggregates `"all"` node Files maps, sorts desc, returns top N with percent-of-max for CSS bar widths
- [ ] implement `computeNodeStats(candles []store.Candle) []NodeStats` — sums per-node Volume (excluding `"all"`), sorts desc, computes percent-of-max
- [ ] implement `computeHeatmap(candles []store.Candle) []HeatmapCell` — buckets by `(hour, weekday)` using candle StartMinute, sums Volume from `"all"` node into 24x7 grid
- [ ] write table-driven tests for `computeSummary` with various candle inputs
- [ ] write tests for `computeTopFiles` — verifies ranking, limit, percent calculation
- [ ] write tests for `computeNodeStats` — verifies "all" node excluded, ranking
- [ ] write tests for `computeHeatmap` — verifies correct hour/weekday bucketing
- [ ] run tests — must pass before next task

### Task 3b: ECharts JSON builder functions

**Files:**
- Modify: `app/web/dashboard.go`
- Modify: `app/web/dashboard_test.go`

Build ECharts-compatible JSON option objects server-side for embedding in templates.

- [ ] implement `buildChartData(candles []store.Candle, aggDuration time.Duration) []byte` — reuses `aggregateCandles`, marshals into ECharts bar chart option JSON (time axis + download counts)
- [ ] implement `buildHeatmapData(cells []HeatmapCell) []byte` — marshals into ECharts heatmap option JSON (24x7 grid with visualMap)
- [ ] write tests for `buildChartData` — verifies valid JSON output with expected structure
- [ ] write tests for `buildHeatmapData` — verifies valid JSON with correct axes and data points
- [ ] run tests — must pass before next task

### Task 4: HTML templates

**Files:**
- Create: `app/web/templates/layout.html`
- Create: `app/web/templates/dashboard.html`
- Create: `app/web/templates/partials/summary.html`
- Create: `app/web/templates/partials/chart.html`
- Create: `app/web/templates/partials/files.html`
- Create: `app/web/templates/partials/nodes.html`
- Create: `app/web/templates/partials/heatmap.html`

- [ ] create `layout.html` — HTML5 shell with `<head>` loading picocss (CDN), echarts (CDN), htmx (CDN); `<body>` with period buttons (`hx-get="/fragment/dashboard?period=..."` `hx-target="#dashboard"`), `{{template "dashboard" .}}` block
- [ ] create `dashboard.html` — defines `"dashboard"` template, includes all partials in order: summary, chart, files, nodes, heatmap
- [ ] create `partials/summary.html` — row of picocss `<article>` cards, each showing period label and total download count
- [ ] create `partials/chart.html` — `<div id="downloads-chart">` container + `<script type="application/json" id="chart-data">` with `{{.ChartJSON}}`
- [ ] create `partials/files.html` — `<table>` with rank, filename, count, CSS bar (`<div style="width:{{.Percent}}%">`)
- [ ] create `partials/nodes.html` — `<table>` with node name, volume, CSS bar
- [ ] create `partials/heatmap.html` — `<div id="heatmap-chart">` container + `<script type="application/json" id="heatmap-data">` with `{{.HeatmapJSON}}`
- [ ] write a simple template parse test — `template.Must(template.ParseFS(...))` to verify all templates are syntactically valid
- [ ] run tests — must pass before next task

### Task 5: Dashboard handler and routes

**Files:**
- Modify: `app/web/server.go`
- Create: `app/web/embed.go`
- Modify: `app/web/server_test.go`

- [ ] create `app/web/embed.go` with `//go:embed templates` directive exposing `embed.FS`
- [ ] add template parsing in `Server` — parse all templates from embedded FS on startup (in `routes()` or a new init method)
- [ ] implement `GET /` handler — loads candles for default period (24h), computes all dashboard data (5 summary loads + selected period), renders `layout.html` with `DashboardData`
- [ ] implement `GET /fragment/dashboard` handler — same computation, renders only `dashboard.html` template (no layout wrapper) for HTMX swap
- [ ] parse `period` query param: `1h`, `12h`, `24h`, `10d`, `30d`, `all` (using `TimeRange` for `all`)
- [ ] register new routes in `routes()`: `GET /` for full page, `GET /fragment/dashboard` for HTMX fragment
- [ ] update static file serving — serve `webapp/` only for `favicon.ico` (or move favicon to embedded FS)
- [ ] update existing `TestServerUI` tests in `server_test.go` — remove assertions for deleted JS files, update `GET /` to expect new SSR HTML
- [ ] remove `webappPrefix` field from `Server` struct and update `startupT` helper accordingly
- [ ] write integration test: `GET /` returns full HTML page with expected sections
- [ ] write integration test: `GET /fragment/dashboard?period=1h` returns HTML fragment without `<html>` wrapper
- [ ] write integration test: `GET /fragment/dashboard?period=all` works with TimeRange
- [ ] write integration test: invalid period returns 400
- [ ] run tests — must pass before next task

### Task 6: ECharts initialisation script

**Files:**
- Create: `app/web/static/charts.js`

Minimal JS that initialises ECharts instances after HTMX content swaps.

- [ ] write `initCharts()` function — finds all `[data-echarts]` containers, reads JSON from associated `<script type="application/json">` sibling, calls `echarts.init(container).setOption(data)`
- [ ] handle chart resize on `window.resize` — call `echarts.getInstanceByDom(el).resize()` for each chart container
- [ ] add `htmx:afterSettle` event listener that calls `initCharts()`
- [ ] call `initCharts()` on `DOMContentLoaded` for initial page load
- [ ] create separate `//go:embed static` in `app/web/embed.go` and serve at `/static/charts.js`
- [ ] add `<script src="/static/charts.js">` to `layout.html`
- [ ] manual verification: charts render correctly after period switch (no automated test for JS)

### Task 7: Remove old frontend

**Files:**
- Delete: `webapp/index.html`
- Delete: `webapp/index.js`
- Delete: `webapp/data.js`
- Delete: `webapp/anycharts.js`
- Delete: `webapp/apexcharts.js`
- Delete: `webapp/echarts.js`
- Delete: `webapp/taucharts.js`
- Keep: `webapp/favicon.ico`

- [ ] delete all JS files from `webapp/`: index.js, data.js, anycharts.js, apexcharts.js, echarts.js, taucharts.js
- [ ] delete `webapp/index.html`
- [ ] keep `webapp/favicon.ico` — serve via embedded FS or keep as static file
- [ ] update `routes()` — remove or adjust old `HandleFiles` for `webapp/` directory (only favicon needed now)
- [ ] verify no remaining references to deleted files in Go code
- [ ] update `Dockerfile` — remove or adjust `COPY webapp /srv/webapp` line (templates are embedded in binary, only favicon may need copying)
- [ ] run tests — must pass before next task

### Task 8: Verify acceptance criteria

- [ ] verify all requirements from issues #23 and #11 (items 1-3) are implemented
- [ ] verify existing `/api/candle` endpoint still works unchanged
- [ ] verify existing `/api/insert` endpoint still works unchanged
- [ ] verify `max_points=200` works correctly
- [ ] run full test suite: `go test -race ./...`
- [ ] run linter: `golangci-lint run`
- [ ] format code: `gofmt -w` on all modified Go files

### Task 9: Update documentation

- [ ] update README.md — document new dashboard, remove references to chart library selector, update `max_points` documentation (no longer capped at 255)
- [ ] move this plan to `docs/plans/completed/`

## Technical Details

### Period mapping

| Period param | Duration | Aggregation step |
|---|---|---|
| `1h` | 1 hour | 1 minute (60 points) |
| `12h` | 12 hours | ~7 minutes (100 points) |
| `24h` | 24 hours | ~15 minutes (100 points) |
| `10d` | 10 days | ~2.4 hours (100 points) |
| `30d` | 30 days | ~7.2 hours (100 points) |
| `all` | TimeRange oldest→now | auto (100 points) |

### Summary card periods

The 5 summary cards always show totals for fixed time ranges, independent of the selected chart period:

| Card label | Time range |
|---|---|
| 1 hour | now - 1h → now |
| 24 hours | now - 24h → now |
| 1 week | now - 7d → now |
| 1 month | now - 30d → now |
| All time | TimeRange oldest → now |

### ECharts option structure for bar chart

```json
{
  "xAxis": {"type": "time"},
  "yAxis": {"type": "value", "name": "Downloads"},
  "series": [{"type": "bar", "data": [[timestamp_ms, count], ...]}],
  "tooltip": {"trigger": "axis"},
  "grid": {"left": "10%", "right": "5%", "bottom": "15%"}
}
```

### ECharts option structure for heatmap

```json
{
  "xAxis": {"type": "category", "data": ["00:00", "01:00", ..., "23:00"]},
  "yAxis": {"type": "category", "data": ["Mon", "Tue", ..., "Sun"]},
  "visualMap": {"min": 0, "max": "auto", "calculable": true},
  "series": [{"type": "heatmap", "data": [[hour, weekday, value], ...]}]
}
```

### Template data structure

```go
type DashboardData struct {
    Summaries   []SummaryData  // 5 entries: 1h, 24h, 1w, 1m, all-time
    ChartJSON   template.JS    // pre-built ECharts bar chart option
    Files       []FileStats    // top 20 files
    Nodes       []NodeStats    // all nodes sorted by volume
    HeatmapJSON template.JS    // pre-built ECharts heatmap option
    Period      string         // currently selected period
}
```

## Post-Completion

**Manual verification:**
- Visual check of dashboard layout on desktop and mobile
- Verify HTMX period switching works smoothly
- Verify ECharts renders correctly after HTMX swap
- Check dark mode appearance with picocss
- Test with real production data

**Future work (requires RLB changes):**
- User-Agent collection and per-app stats (#9, #11 item 5)
- GeoIP lookup at ingestion time for country stats (#9, #11 item 6)
- Bot filtering via UA patterns (#9)
