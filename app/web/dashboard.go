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
	Summaries   []SummaryData // 5 entries: 1h, 24h, 1w, 1m, all-time
	ChartJSON   template.JS   // pre-built ECharts bar chart option
	Files       []FileStats   // top files
	Nodes       []NodeStats   // all nodes sorted by volume
	HeatmapJSON template.JS   // pre-built ECharts heatmap option
	Period      string        // currently selected period
}

// SummaryData holds a period label and total download count
type SummaryData struct {
	Label string
	Count int
}

// FileStats holds a filename with its count and percent of max for CSS bar widths
type FileStats struct {
	Name    string
	Count   int
	Percent int // 0-100, relative to the top file
}

// NodeStats holds a node name with its volume and percent of max
type NodeStats struct {
	Name    string
	Volume  int
	Percent int // 0-100, relative to the top node
}

// HeatmapCell holds a single cell in the 24x7 heatmap grid
type HeatmapCell struct {
	Hour    int
	Weekday int // 0=Monday, 6=Sunday
	Value   int
}

// computeSummary sums the "all" node Volume across candles
func computeSummary(candles []store.Candle) int {
	total := 0
	for _, c := range candles {
		if info, ok := c.Nodes["all"]; ok {
			total += info.Volume
		}
	}
	return total
}

// computeTopFiles aggregates file download counts from the "all" node across candles,
// sorts descending, and returns top N with percent-of-max for CSS bar widths
func computeTopFiles(candles []store.Candle, limit int) []FileStats {
	counts := map[string]int{}
	for _, c := range candles {
		if info, ok := c.Nodes["all"]; ok {
			for name, count := range info.Files {
				counts[name] += count
			}
		}
	}

	files := make([]FileStats, 0, len(counts))
	for name, count := range counts {
		files = append(files, FileStats{Name: name, Count: count})
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

// computeNodeStats sums per-node Volume (excluding "all"), sorts descending,
// and computes percent-of-max
func computeNodeStats(candles []store.Candle) []NodeStats {
	volumes := map[string]int{}
	for _, c := range candles {
		for name, info := range c.Nodes {
			if name == "all" {
				continue
			}
			volumes[name] += info.Volume
		}
	}

	nodes := make([]NodeStats, 0, len(volumes))
	for name, vol := range volumes {
		nodes = append(nodes, NodeStats{Name: name, Volume: vol})
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

		if info, ok := c.Nodes["all"]; ok {
			grid[hour][wd] += info.Volume
		}
	}

	cells := make([]HeatmapCell, 0, 24*7)
	for h := 0; h < 24; h++ {
		for d := 0; d < 7; d++ {
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

// buildChartData aggregates candles by aggDuration and marshals into ECharts bar chart option JSON.
// returns empty JSON object on error or empty input.
func buildChartData(candles []store.Candle, aggDuration time.Duration) template.JS {
	aggregated := aggregateCandles(context.Background(), candles, aggDuration)

	data := make([][]any, 0, len(aggregated))
	for _, c := range aggregated {
		vol := 0
		if info, ok := c.Nodes["all"]; ok {
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
	for h := 0; h < 24; h++ {
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
