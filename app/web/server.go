package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
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

// Global anonymous struct, is it bad?
var apiClient struct {
	apiURL     string
	httpClient *http.Client
}

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate UI web server on port %v", s.Port)
	apiClient.apiURL = fmt.Sprintf("http://localhost:%v", s.APIPort)
	apiClient.httpClient = &http.Client{Timeout: 60 * time.Second}
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
		log.Printf("[WARN] /: unable to load candles: %v", err)
		http.Error(w, fmt.Sprintf("unable to load candles: %v", err), http.StatusBadRequest)
		return
	}

	result := struct {
		TopFiles []volumeStats
		TopNodes []volumeStats
		From, To string
	}{
		getTop("files", candles, 10),
		getTop("nodes", candles, 10),
		from,
		to,
	}

	t := template.Must(template.ParseFiles("webapp/dashboard.html.tpl"))
	err = t.Execute(w, result)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to execute template: %v", err), http.StatusInternalServerError)
		log.Printf("[WARN] dashboard: unable to execute template: %v", err)
		return
	}
	render.Status(r, http.StatusOK)
}

// GET /file_stats
func getFileStats(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, fmt.Sprint("'filename' parameter is required"), http.StatusUnprocessableEntity)
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	fromTime, toTime, aggDuration := calculateTimePeriod(from, to)
	candles, err := loadCandles(fromTime, toTime, aggDuration)
	if err != nil {
		log.Printf("[WARN] /file_stats: unable to load candles: %v", err)
		http.Error(w, fmt.Sprintf("unable to load candles: %v", err), http.StatusInternalServerError)
		return
	}

	result := struct {
		Filename string
		Candles  []store.Candle
		From, To string
	}{
		filename,
		candles,
		from,
		to,
	}

	t := template.Must(template.ParseFiles("webapp/file_stats.html.tpl"))
	err = t.Execute(w, result)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to execute template: %v", err), http.StatusInternalServerError)
		log.Printf("[WARN] dashboard: unable to execute template: %v", err)
		return
	}
	render.Status(r, http.StatusOK)
}

// GET /chart
func drawChart(w http.ResponseWriter, r *http.Request) {
	fromTime, toTime, aggDuration := calculateTimePeriod(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
	)
	candles, err := loadCandles(fromTime, toTime, aggDuration)
	if err != nil {
		log.Printf("[WARN] dashboard: unable to load candles: %v", err)
		http.Error(w, fmt.Sprintf("unable to load candles: %v", err), http.StatusInternalServerError)
		return
	}
	qType := r.URL.Query().Get("type")
	filename := r.URL.Query().Get("filename")
	series := prepareSeries(candles, qType, filename)

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Style:          chart.StyleShow(),
			ValueFormatter: chart.TimeValueFormatterWithFormat(time.RFC3339),
		},
		YAxis: chart.YAxis{
			Style:          chart.StyleShow(),
			ValueFormatter: valueFormatter,
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
		http.Error(w, fmt.Sprintf("unable to render graph: %v", err), http.StatusBadRequest)
		log.Printf("[WARN] dashboard: unable to render graph: %v", err)
	}
}
