package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

// DashboardData holds all data needed to render the dashboard page
type DashboardData struct {
	Summaries   []SummaryData  // 5 entries: 1h, 24h, 1w, 1m, all-time
	ChartJSON   template.JS    // pre-built ECharts bar chart option
	Files       []FileStats    // top files
	Nodes       []NodeStats    // all nodes sorted by volume
	Countries   []CountryStats // top countries, nginx tail mode only
	HeatmapJSON template.JS    // pre-built ECharts heatmap option
	Period      string         // currently selected period
}

// SummaryData holds a period label, total download count and total bytes transferred.
// Bytes is zero on candles stored before nginx tail mode, so the number is only ever
// non-zero for data collected via nginx tail mode.
type SummaryData struct {
	Label      string
	Count      int
	Bytes      int64
	HumanBytes string // humanised Bytes for direct template use
}

// FileStats holds a filename with its count, bytes, and percent of max for CSS bar widths
type FileStats struct {
	Name       string
	Count      int
	Bytes      int64
	HumanBytes string
	Percent    int // 0-100, relative to the top file
}

// NodeStats holds a node name with its volume, bytes, and percent of max
type NodeStats struct {
	Name       string
	Volume     int
	Bytes      int64
	HumanBytes string
	Percent    int // 0-100, relative to the top node
}

// CountryStats holds a country code with its download count and percent of max
type CountryStats struct {
	Code    string
	Count   int
	Percent int // 0-100, relative to the top country
}

// HeatmapCell holds a single cell in the 24x7 heatmap grid
type HeatmapCell struct {
	Hour    int
	Weekday int // 0=Monday, 6=Sunday
	Value   int
}

// firstIndexSince returns the index of the first candle at or after from,
// assuming candles are sorted ascending by StartMinute (bolt key order).
// callers subslice candles[idx:] to view a trailing window without copying.
func firstIndexSince(candles []store.Candle, from time.Time) int {
	return sort.Search(len(candles), func(i int) bool {
		return !candles[i].StartMinute.Before(from)
	})
}

// computeSummary sums the "all" node Volume and Bytes across candles. Bytes is reported
// alongside Count so the summary cards can show both without a second pass over candles.
func computeSummary(candles []store.Candle) (count int, bytesSum int64) {
	for _, c := range candles {
		if info, ok := c.Nodes[store.AllNode]; ok {
			count += info.Volume
			bytesSum += info.Bytes
		}
	}
	return count, bytesSum
}

// humanBytes formats a byte count with units up to EB using 1024 as the divisor.
// Zero returns "0 B" so templates don't need a nil check. Small values keep three
// significant figures where it reads naturally; larger units use one decimal place.
func humanBytes(n int64) string {
	if n < 0 {
		return "0 B"
	}
	const unit = 1024
	if n < unit {
		return formatBytes(float64(n), "B", 0)
	}
	div, exp := int64(unit), 0
	for size := n / unit; size >= unit; size /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return formatBytes(float64(n)/float64(div), units[exp], 2)
}

// formatBytes handles the "do we need a decimal point" bit of humanBytes so a 0-decimal
// unit (B) and a 2-decimal unit (MB) share one call site.
func formatBytes(v float64, unit string, decimals int) string {
	if decimals == 0 {
		return fmt.Sprintf("%d %s", int64(v), unit)
	}
	return fmt.Sprintf("%.*f %s", decimals, v, unit)
}

// computeTopFiles aggregates file download counts AND bytes from the "all" node across
// candles, sorts descending, and returns top N with percent-of-max for CSS bar widths.
// The bytes-per-file breakdown lives on AllNode because Files is only tracked there.
func computeTopFiles(candles []store.Candle, limit int) []FileStats {
	counts := map[string]int{}
	bytesPerFile := map[string]int64{}
	for _, c := range candles {
		if info, ok := c.Nodes[store.AllNode]; ok {
			for name, count := range info.Files {
				counts[name] += count
			}
			// bytes land as a per-file breakdown only when the file appears in the Files
			// map on this candle; we proportion the Info.Bytes by file-count share.
			// a per-file bytes map on Info would be cleaner but changes the schema more
			// than this PR needs — call this out as deferred.
			if info.Volume > 0 {
				share := float64(info.Bytes) / float64(info.Volume)
				for name, count := range info.Files {
					bytesPerFile[name] += int64(share * float64(count))
				}
			}
		}
	}

	files := make([]FileStats, 0, len(counts))
	for name, count := range counts {
		files = append(files, FileStats{
			Name:       name,
			Count:      count,
			Bytes:      bytesPerFile[name],
			HumanBytes: humanBytes(bytesPerFile[name]),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Count != files[j].Count {
			return files[i].Count > files[j].Count
		}
		return files[i].Name < files[j].Name
	})

	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	if len(files) > 0 {
		maxCount := files[0].Count
		for i := range files {
			files[i].Percent = files[i].Count * 100 / maxCount
		}
	}

	return files
}

// computeTopCountries sums per-country download counts from the "all" node across
// candles. Empty country codes are already dropped at parse time, so no further
// filtering is needed here.
func computeTopCountries(candles []store.Candle, limit int) []CountryStats {
	counts := map[string]int{}
	for _, c := range candles {
		if info, ok := c.Nodes[store.AllNode]; ok {
			for code, cnt := range info.Countries {
				counts[code] += cnt
			}
		}
	}

	countries := make([]CountryStats, 0, len(counts))
	for code, count := range counts {
		countries = append(countries, CountryStats{Code: code, Count: count})
	}
	sort.Slice(countries, func(i, j int) bool {
		if countries[i].Count != countries[j].Count {
			return countries[i].Count > countries[j].Count
		}
		return countries[i].Code < countries[j].Code
	})

	if limit > 0 && len(countries) > limit {
		countries = countries[:limit]
	}

	if len(countries) > 0 {
		maxCount := countries[0].Count
		for i := range countries {
			countries[i].Percent = countries[i].Count * 100 / maxCount
		}
	}

	return countries
}

// computeNodeStats sums per-node Volume and Bytes (excluding "all"), sorts descending,
// and computes percent-of-max. Per-node Bytes was added for nginx tail mode: with nginx
// vhosts as nodes (radio-t.com, cdn.radio-t.com), per-vhost bandwidth is useful.
func computeNodeStats(candles []store.Candle) []NodeStats {
	volumes := map[string]int{}
	bytesPerNode := map[string]int64{}
	for _, c := range candles {
		for name, info := range c.Nodes {
			if name == store.AllNode {
				continue
			}
			volumes[name] += info.Volume
			bytesPerNode[name] += info.Bytes
		}
	}

	nodes := make([]NodeStats, 0, len(volumes))
	for name, vol := range volumes {
		nodes = append(nodes, NodeStats{
			Name:       name,
			Volume:     vol,
			Bytes:      bytesPerNode[name],
			HumanBytes: humanBytes(bytesPerNode[name]),
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Volume != nodes[j].Volume {
			return nodes[i].Volume > nodes[j].Volume
		}
		return nodes[i].Name < nodes[j].Name
	})

	if len(nodes) > 0 {
		maxVol := nodes[0].Volume
		for i := range nodes {
			nodes[i].Percent = nodes[i].Volume * 100 / maxVol
		}
	}

	return nodes
}

// computeHeatmap buckets candle data by (hour, weekday) using StartMinute,
// sums Volume from the "all" node into a 24x7 grid.
// Weekday: 0=Monday, 6=Sunday
func computeHeatmap(candles []store.Candle) []HeatmapCell {
	var grid [24][7]int

	for _, c := range candles {
		hour := c.StartMinute.Hour()
		wd := int(c.StartMinute.Weekday()) // Sunday=0, Monday=1, ..., Saturday=6
		// convert to Monday=0 ... Sunday=6
		wd = (wd + 6) % 7

		if info, ok := c.Nodes[store.AllNode]; ok {
			grid[hour][wd] += info.Volume
		}
	}

	cells := make([]HeatmapCell, 0, 24*7)
	for h := range 24 {
		for d := range 7 {
			cells = append(cells, HeatmapCell{Hour: h, Weekday: d, Value: grid[h][d]})
		}
	}
	return cells
}

// echartsBarOption is the ECharts option structure for a bar chart
type echartsBarOption struct {
	XAxis   echartsAxis    `json:"xAxis"`
	YAxis   echartsAxis    `json:"yAxis"`
	Series  []echartsBar   `json:"series"`
	Tooltip echartsTooltip `json:"tooltip"`
	Grid    echartsGrid    `json:"grid"`
}

type echartsAxis struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type echartsBar struct {
	Type string  `json:"type"`
	Data [][]any `json:"data"` // [[timestamp_ms, count], ...]
}

type echartsTooltip struct {
	Trigger string `json:"trigger"`
}

type echartsGrid struct {
	Left   string `json:"left"`
	Right  string `json:"right"`
	Bottom string `json:"bottom"`
}

// echartsHeatmapOption is the ECharts option structure for a heatmap
type echartsHeatmapOption struct {
	XAxis     echartsCatAxis   `json:"xAxis"`
	YAxis     echartsCatAxis   `json:"yAxis"`
	VisualMap echartsVisualMap `json:"visualMap"`
	Series    []echartsHeatmap `json:"series"`
	Tooltip   echartsTooltip   `json:"tooltip"`
	Grid      echartsGrid      `json:"grid"`
}

type echartsCatAxis struct {
	Type string   `json:"type"`
	Data []string `json:"data"`
}

type echartsVisualMap struct {
	Min        int  `json:"min"`
	Max        int  `json:"max"`
	Calculable bool `json:"calculable"`
}

type echartsHeatmap struct {
	Type string  `json:"type"`
	Data [][]int `json:"data"` // [[hour, weekday, value], ...]
}

// buildChartData aggregates candles by aggDuration and marshals them into an ECharts bar
// chart option as JSON. empty input still yields a valid option with an empty series;
// only a marshal failure falls back to "{}".
func buildChartData(ctx context.Context, candles []store.Candle, aggDuration time.Duration) template.JS {
	aggregated := aggregateCandles(ctx, candles, aggDuration)

	data := make([][]any, 0, len(aggregated))
	for _, c := range aggregated {
		vol := 0
		if info, ok := c.Nodes[store.AllNode]; ok {
			vol = info.Volume
		}
		data = append(data, []any{c.StartMinute.UnixMilli(), vol})
	}

	opt := echartsBarOption{
		XAxis:   echartsAxis{Type: "time"},
		YAxis:   echartsAxis{Type: "value", Name: "Downloads"},
		Series:  []echartsBar{{Type: "bar", Data: data}},
		Tooltip: echartsTooltip{Trigger: "axis"},
		Grid:    echartsGrid{Left: "10%", Right: "5%", Bottom: "15%"},
	}

	b, err := json.Marshal(opt)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b) //nolint:gosec // trusted server-generated JSON
}

// buildHeatmapData marshals HeatmapCell slice into ECharts heatmap option JSON
func buildHeatmapData(cells []HeatmapCell) template.JS {
	hours := make([]string, 24)
	for h := range 24 {
		hours[h] = fmt.Sprintf("%02d:00", h)
	}
	weekdays := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	maxVal := 0
	data := make([][]int, 0, len(cells))
	for _, c := range cells {
		data = append(data, []int{c.Hour, c.Weekday, c.Value})
		if c.Value > maxVal {
			maxVal = c.Value
		}
	}

	opt := echartsHeatmapOption{
		XAxis:     echartsCatAxis{Type: "category", Data: hours},
		YAxis:     echartsCatAxis{Type: "category", Data: weekdays},
		VisualMap: echartsVisualMap{Min: 0, Max: maxVal, Calculable: true},
		Series:    []echartsHeatmap{{Type: "heatmap", Data: data}},
		Tooltip:   echartsTooltip{Trigger: "item"},
		Grid:      echartsGrid{Left: "10%", Right: "5%", Bottom: "15%"},
	}

	b, err := json.Marshal(opt)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b) //nolint:gosec // trusted server-generated JSON
}
