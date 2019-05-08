package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/wcharczuk/go-chart"

	"github.com/umputun/rlb-stats/app/store"
)

// Server is a UI for rlb-stats rest backend
type Server struct {
	Port     int
	APIPort  int
	RESTPort int
}

// Global variable, is it bad?
var apiPort int

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate UI web server on port %v", s.Port)
	apiPort = s.APIPort
	r := chi.NewRouter()

	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))

	r.Get("/", getDashboard)
	r.Get("/file_stats", getFileStats)
	r.Get("/chart", drawChart)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", s.Port), r))
}

// GET /
func getDashboard(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	fromTime, toTime, aggDuration := calculateTimePeriod(from, to)
	candles, err := loadCandles(fromTime, toTime, aggDuration)
	if err != nil {
		// TODO handle being unable to get candles
		log.Printf("[WARN] dashboard: unable to load candles: %v", err)
		return
	}

	result := struct {
		TopFiles []volumeStats
		TopNodes []volumeStats
		Charts   []string
	}{
		getTop("files", candles, 10),
		getTop("nodes", candles, 10),
		[]string{
			fmt.Sprintf("/chart?from=%v&to=%v&type=by_file", from, to),
			fmt.Sprintf("/chart?from=%v&to=%v&type=by_node", from, to),
		},
	}

	t := template.Must(template.ParseFiles("webapp/dashboard.html.tpl"))
	err = t.Execute(w, result)
	if err != nil {
		// TODO handle template execution problem
		log.Printf("[WARN] dashboard: unable to execute template: %v", err)
		return
	}
	render.Status(r, http.StatusOK)
}

// GET /file_stats
func getFileStats(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		log.Printf("no 'filename' field passed")
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	fromTime, toTime, aggDuration := calculateTimePeriod(from, to)
	candles, err := loadCandles(fromTime, toTime, aggDuration)
	if err != nil {
		// TODO handle being unable to get candles
		log.Printf("[WARN] dashboard: unable to load candles: %v", err)
		return
	}

	result := struct {
		Filename string
		Charts   []string
		Candles  []store.Candle
	}{
		filename,
		[]string{
			fmt.Sprintf("/chart?from=%v&to=%v&type=by_file&filename=%v", from, to, filename),
		},
		candles,
	}

	t := template.Must(template.ParseFiles("webapp/file_stats.html.tpl"))
	err = t.Execute(w, result)
	if err != nil {
		// TODO handle template execution problem
		log.Printf("[WARN] dashboard: unable to execute template: %v", err)
		return
	}
	render.Status(r, http.StatusOK)
}

func drawChart(w http.ResponseWriter, r *http.Request) {
	fromTime, toTime, aggDuration := calculateTimePeriod(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
	)
	candles, err := loadCandles(fromTime, toTime, aggDuration)
	if err != nil {
		// TODO handle being unable to get candles
		log.Printf("[WARN] dashboard: unable to load candles: %v", err)
		return
	}
	qType := r.URL.Query().Get("type")
	filename := r.URL.Query().Get("filename")
	series := prepareSeries(candles, fromTime, toTime, aggDuration, qType, filename)

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Style: chart.StyleShow(),
		},
		YAxis: chart.YAxis{
			Style: chart.StyleShow(),
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:  20,
				Left: 20,
			},
		},
		Series: series,
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	w.Header().Set("Content-Type", "image/png")
	err = graph.Render(chart.PNG, w)
	if err != nil {
		// TODO handle graph generation problem
		log.Printf("[WARN] dashboard: unable to render graph: %v", err)
	}
}

func loadCandles(from time.Time, to time.Time, duration time.Duration) ([]store.Candle, error) {
	var result []store.Candle
	candleGetURL := fmt.Sprintf("http://localhost:%v/api/candle?from=%v&to=%v&aggregate=%v",
		apiPort,
		url.QueryEscape(from.Format(time.RFC3339)),
		url.QueryEscape(to.Format(time.RFC3339)),
		duration)
	var myClient = &http.Client{Timeout: 60 * time.Second}
	r, err := myClient.Get(candleGetURL)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, r.Body.Close()
}
