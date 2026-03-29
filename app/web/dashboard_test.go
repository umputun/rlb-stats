package web

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

func TestComputeSummary(t *testing.T) {
	tests := []struct {
		name    string
		candles []store.Candle
		want    int
	}{
		{
			name:    "empty candles",
			candles: nil,
			want:    0,
		},
		{
			name: "single candle with all node",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 42}}},
			},
			want: 42,
		},
		{
			name: "multiple candles summed",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 10}, "node1": {Volume: 5}}},
				{Nodes: map[string]store.Info{"all": {Volume: 20}, "node2": {Volume: 15}}},
				{Nodes: map[string]store.Info{"all": {Volume: 30}}},
			},
			want: 60,
		},
		{
			name: "candle without all node",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"node1": {Volume: 100}}},
			},
			want: 0,
		},
		{
			name: "mixed candles with and without all node",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 10}}},
				{Nodes: map[string]store.Info{"node1": {Volume: 100}}},
				{Nodes: map[string]store.Info{"all": {Volume: 5}}},
			},
			want: 15,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeSummary(tc.candles)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestComputeTopFiles(t *testing.T) {
	tests := []struct {
		name    string
		candles []store.Candle
		limit   int
		want    []FileStats
	}{
		{
			name:    "empty candles",
			candles: nil,
			limit:   10,
			want:    []FileStats{},
		},
		{
			name: "single file",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 5, Files: map[string]int{"ep01.mp3": 5}}}},
			},
			limit: 10,
			want:  []FileStats{{Name: "ep01.mp3", Count: 5, Percent: 100}},
		},
		{
			name: "multiple files sorted and limited",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 10, Files: map[string]int{
					"ep01.mp3": 5, "ep02.mp3": 3, "ep03.mp3": 8,
				}}}},
				{Nodes: map[string]store.Info{"all": {Volume: 5, Files: map[string]int{
					"ep01.mp3": 2, "ep04.mp3": 1,
				}}}},
			},
			limit: 2,
			want: []FileStats{
				{Name: "ep03.mp3", Count: 8, Percent: 100},
				{Name: "ep01.mp3", Count: 7, Percent: 87},
			},
		},
		{
			name: "percent calculation",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 30, Files: map[string]int{
					"a.mp3": 100, "b.mp3": 50, "c.mp3": 25,
				}}}},
			},
			limit: 10,
			want: []FileStats{
				{Name: "a.mp3", Count: 100, Percent: 100},
				{Name: "b.mp3", Count: 50, Percent: 50},
				{Name: "c.mp3", Count: 25, Percent: 25},
			},
		},
		{
			name: "non-all nodes ignored",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{
					"all":   {Volume: 5, Files: map[string]int{"ep01.mp3": 5}},
					"node1": {Volume: 100, Files: map[string]int{"ep99.mp3": 100}},
				}},
			},
			limit: 10,
			want:  []FileStats{{Name: "ep01.mp3", Count: 5, Percent: 100}},
		},
		{
			name: "limit zero means no limit",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 6, Files: map[string]int{
					"a.mp3": 3, "b.mp3": 2, "c.mp3": 1,
				}}}},
			},
			limit: 0,
			want: []FileStats{
				{Name: "a.mp3", Count: 3, Percent: 100},
				{Name: "b.mp3", Count: 2, Percent: 66},
				{Name: "c.mp3", Count: 1, Percent: 33},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeTopFiles(tc.candles, tc.limit)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestComputeNodeStats(t *testing.T) {
	tests := []struct {
		name    string
		candles []store.Candle
		want    []NodeStats
	}{
		{
			name:    "empty candles",
			candles: nil,
			want:    []NodeStats{},
		},
		{
			name: "all node excluded",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{"all": {Volume: 100}}},
			},
			want: []NodeStats{},
		},
		{
			name: "multiple nodes sorted by volume",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{
					"all":   {Volume: 30},
					"node1": {Volume: 10},
					"node2": {Volume: 20},
				}},
				{Nodes: map[string]store.Info{
					"all":   {Volume: 15},
					"node1": {Volume: 5},
					"node3": {Volume: 15},
				}},
			},
			want: []NodeStats{
				{Name: "node2", Volume: 20, Percent: 100},
				{Name: "node1", Volume: 15, Percent: 75},
				{Name: "node3", Volume: 15, Percent: 75},
			},
		},
		{
			name: "single non-all node",
			candles: []store.Candle{
				{Nodes: map[string]store.Info{
					"all":   {Volume: 10},
					"node1": {Volume: 10},
				}},
			},
			want: []NodeStats{
				{Name: "node1", Volume: 10, Percent: 100},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeNodeStats(tc.candles)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestComputeHeatmap(t *testing.T) {
	tests := []struct {
		name    string
		candles []store.Candle
		checks  func(t *testing.T, cells []HeatmapCell)
	}{
		{
			name:    "empty candles returns full 24x7 grid of zeros",
			candles: nil,
			checks: func(t *testing.T, cells []HeatmapCell) {
				require.Len(t, cells, 24*7)
				for _, c := range cells {
					assert.Zero(t, c.Value, "all cells should be zero for empty input")
				}
			},
		},
		{
			name: "single candle buckets correctly",
			candles: []store.Candle{
				{
					// Wednesday 2024-01-10 at 14:00 UTC
					StartMinute: time.Date(2024, 1, 10, 14, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 42}},
				},
			},
			checks: func(t *testing.T, cells []HeatmapCell) {
				require.Len(t, cells, 24*7)
				// Wednesday = weekday 2 (Monday=0)
				found := false
				for _, c := range cells {
					if c.Hour == 14 && c.Weekday == 2 {
						assert.Equal(t, 42, c.Value)
						found = true
					}
				}
				assert.True(t, found, "should find cell for hour=14, weekday=2 (Wednesday)")
			},
		},
		{
			name: "multiple candles same hour/day accumulate",
			candles: []store.Candle{
				{
					// Monday 2024-01-08 at 09:00
					StartMinute: time.Date(2024, 1, 8, 9, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 10}},
				},
				{
					// Monday 2024-01-15 at 09:30 (same hour, same weekday)
					StartMinute: time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 15}},
				},
			},
			checks: func(t *testing.T, cells []HeatmapCell) {
				// Monday = weekday 0
				for _, c := range cells {
					if c.Hour == 9 && c.Weekday == 0 {
						assert.Equal(t, 25, c.Value)
						return
					}
				}
				t.Fatal("should find cell for hour=9, weekday=0 (Monday)")
			},
		},
		{
			name: "sunday maps to weekday 6",
			candles: []store.Candle{
				{
					// Sunday 2024-01-07 at 22:00
					StartMinute: time.Date(2024, 1, 7, 22, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 7}},
				},
			},
			checks: func(t *testing.T, cells []HeatmapCell) {
				for _, c := range cells {
					if c.Hour == 22 && c.Weekday == 6 {
						assert.Equal(t, 7, c.Value)
						return
					}
				}
				t.Fatal("should find cell for hour=22, weekday=6 (Sunday)")
			},
		},
		{
			name: "non-all nodes ignored in heatmap",
			candles: []store.Candle{
				{
					StartMinute: time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"node1": {Volume: 999}},
				},
			},
			checks: func(t *testing.T, cells []HeatmapCell) {
				for _, c := range cells {
					assert.Zero(t, c.Value, "non-all node volumes should not appear in heatmap")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cells := computeHeatmap(tc.candles)
			tc.checks(t, cells)
		})
	}
}

func TestBuildChartData(t *testing.T) {
	tests := []struct {
		name        string
		candles     []store.Candle
		aggDuration time.Duration
		checks      func(t *testing.T, js string)
	}{
		{
			name:        "empty candles produces valid JSON",
			candles:     nil,
			aggDuration: time.Minute,
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))
				assert.Equal(t, "time", opt["xAxis"].(map[string]any)["type"])
				assert.Equal(t, "value", opt["yAxis"].(map[string]any)["type"])
				series := opt["series"].([]any)
				require.Len(t, series, 1)
				s := series[0].(map[string]any)
				assert.Equal(t, "bar", s["type"])
				// empty candles = empty data
				data := s["data"].([]any)
				assert.Empty(t, data)
			},
		},
		{
			name: "single candle produces correct data point",
			candles: []store.Candle{
				{
					StartMinute: time.Date(2024, 1, 10, 14, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 42}},
				},
			},
			aggDuration: time.Minute,
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))
				series := opt["series"].([]any)
				s := series[0].(map[string]any)
				data := s["data"].([]any)
				require.Len(t, data, 1)
				point := data[0].([]any)
				require.Len(t, point, 2)
				// timestamp in ms
				ts := int64(point[0].(float64))
				assert.Equal(t, time.Date(2024, 1, 10, 14, 0, 0, 0, time.UTC).UnixMilli(), ts)
				// volume
				assert.Equal(t, float64(42), point[1])
			},
		},
		{
			name: "aggregation groups candles",
			candles: []store.Candle{
				{
					StartMinute: time.Date(2024, 1, 10, 14, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 10}},
				},
				{
					StartMinute: time.Date(2024, 1, 10, 14, 1, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 20}},
				},
				{
					StartMinute: time.Date(2024, 1, 10, 14, 2, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 5}},
				},
			},
			aggDuration: 5 * time.Minute,
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))
				series := opt["series"].([]any)
				s := series[0].(map[string]any)
				data := s["data"].([]any)
				// all 3 candles within 5min window => single aggregated point
				require.Len(t, data, 1)
				point := data[0].([]any)
				assert.Equal(t, float64(35), point[1])
			},
		},
		{
			name: "has expected top-level fields",
			candles: []store.Candle{
				{
					StartMinute: time.Date(2024, 1, 10, 14, 0, 0, 0, time.UTC),
					Nodes:       map[string]store.Info{"all": {Volume: 1}},
				},
			},
			aggDuration: time.Minute,
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))
				assert.Contains(t, opt, "xAxis")
				assert.Contains(t, opt, "yAxis")
				assert.Contains(t, opt, "series")
				assert.Contains(t, opt, "tooltip")
				assert.Contains(t, opt, "grid")
				tooltip := opt["tooltip"].(map[string]any)
				assert.Equal(t, "axis", tooltip["trigger"])
				grid := opt["grid"].(map[string]any)
				assert.Equal(t, "10%", grid["left"])
				assert.Equal(t, "5%", grid["right"])
				assert.Equal(t, "15%", grid["bottom"])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildChartData(tc.candles, tc.aggDuration)
			tc.checks(t, string(result))
		})
	}
}

func TestBuildHeatmapData(t *testing.T) {
	tests := []struct {
		name   string
		cells  []HeatmapCell
		checks func(t *testing.T, js string)
	}{
		{
			name:  "empty cells produces valid JSON with axes",
			cells: nil,
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))

				xAxis := opt["xAxis"].(map[string]any)
				assert.Equal(t, "category", xAxis["type"])
				hours := xAxis["data"].([]any)
				require.Len(t, hours, 24)
				assert.Equal(t, "00:00", hours[0])
				assert.Equal(t, "23:00", hours[23])

				yAxis := opt["yAxis"].(map[string]any)
				assert.Equal(t, "category", yAxis["type"])
				weekdays := yAxis["data"].([]any)
				require.Len(t, weekdays, 7)
				assert.Equal(t, "Mon", weekdays[0])
				assert.Equal(t, "Sun", weekdays[6])

				series := opt["series"].([]any)
				require.Len(t, series, 1)
				s := series[0].(map[string]any)
				assert.Equal(t, "heatmap", s["type"])
			},
		},
		{
			name: "data points match input cells",
			cells: []HeatmapCell{
				{Hour: 10, Weekday: 0, Value: 100},
				{Hour: 15, Weekday: 3, Value: 50},
			},
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))

				series := opt["series"].([]any)
				s := series[0].(map[string]any)
				data := s["data"].([]any)
				require.Len(t, data, 2)

				p1 := data[0].([]any)
				assert.Equal(t, float64(10), p1[0])
				assert.Equal(t, float64(0), p1[1])
				assert.Equal(t, float64(100), p1[2])

				p2 := data[1].([]any)
				assert.Equal(t, float64(15), p2[0])
				assert.Equal(t, float64(3), p2[1])
				assert.Equal(t, float64(50), p2[2])
			},
		},
		{
			name: "visualMap max set to highest value",
			cells: []HeatmapCell{
				{Hour: 0, Weekday: 0, Value: 10},
				{Hour: 1, Weekday: 1, Value: 200},
				{Hour: 2, Weekday: 2, Value: 50},
			},
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))
				vm := opt["visualMap"].(map[string]any)
				assert.Equal(t, float64(0), vm["min"])
				assert.Equal(t, float64(200), vm["max"])
				assert.Equal(t, true, vm["calculable"])
			},
		},
		{
			name: "full 24x7 grid produces 168 data points",
			cells: func() []HeatmapCell {
				cells := make([]HeatmapCell, 0, 24*7)
				for h := 0; h < 24; h++ {
					for d := 0; d < 7; d++ {
						cells = append(cells, HeatmapCell{Hour: h, Weekday: d, Value: h + d})
					}
				}
				return cells
			}(),
			checks: func(t *testing.T, js string) {
				var opt map[string]any
				require.NoError(t, json.Unmarshal([]byte(js), &opt))
				series := opt["series"].([]any)
				s := series[0].(map[string]any)
				data := s["data"].([]any)
				assert.Len(t, data, 24*7)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildHeatmapData(tc.cells)
			tc.checks(t, string(result))
		})
	}
}

func TestTemplateParse(t *testing.T) {
	templatesFS := os.DirFS("templates")

	funcMap := template.FuncMap{
		"list": func(args ...string) []string { return args },
		"inc":  func(i int) int { return i + 1 },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "*.html", "partials/*.html")
	require.NoError(t, err, "templates must parse without errors")

	// verify all expected templates are present
	expectedTemplates := []string{"dashboard", "summary", "chart", "files", "nodes", "heatmap"}
	for _, name := range expectedTemplates {
		assert.NotNil(t, tmpl.Lookup(name), "template %q should be defined", name)
	}
}

func TestTemplateRender(t *testing.T) {
	templatesFS := os.DirFS("templates")

	funcMap := template.FuncMap{
		"list": func(args ...string) []string { return args },
		"inc":  func(i int) int { return i + 1 },
	}

	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFS(templatesFS, "*.html", "partials/*.html")
	require.NoError(t, err)

	data := DashboardData{
		Summaries: []SummaryData{
			{Label: "1 hour", Count: 100},
			{Label: "24 hours", Count: 500},
		},
		ChartJSON:   template.JS(`{"xAxis":{"type":"time"}}`),
		Files:       []FileStats{{Name: "file1.mp3", Count: 10, Percent: 100}},
		Nodes:       []NodeStats{{Name: "node1", Volume: 50, Percent: 100}},
		HeatmapJSON: template.JS(`{"xAxis":{"type":"category"}}`),
		Period:      "24h",
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err, "layout template must render without errors")

	html := buf.String()
	assert.Contains(t, html, "RLB Stats")
	assert.Contains(t, html, "picocss")
	assert.Contains(t, html, "echarts")
	assert.Contains(t, html, "htmx")
	assert.Contains(t, html, "100")                 // summary count
	assert.Contains(t, html, "1 hour")              // summary label
	assert.Contains(t, html, "file1.mp3")           // file name
	assert.Contains(t, html, "node1")               // node name
	assert.Contains(t, html, `aria-current="true"`) // active period button
	assert.Contains(t, html, "chart-data")          // chart JSON container
	assert.Contains(t, html, "heatmap-data")        // heatmap JSON container
}
