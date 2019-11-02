package web

import (
	"errors"
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

// Server is a web-server for rlb-stats REST API and UI
type Server struct {
	Engine       store.Engine
	Port         int
	Version      string
	address      string // set only in tests
	webappPrefix string // set only in tests
}

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate web server on port %v", s.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%v", s.address, s.Port), s.routes()))
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
	r.Use(appInfo("rlb-stats", s.Version), Ping)

	r.Get("/", s.getDashboard)
	r.Get("/file_stats", s.getFileStats)
	r.Get("/chart", s.drawChart)

	r.Route("/api", func(r chi.Router) {
		r.Get("/candle", s.getCandle)
	})

	return r
}

func sendErrorJSON(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	log.Printf("[WARN] %s", details)
	render.Status(r, code)
	render.JSON(w, r, JSON{"error": err.Error(), "details": details})
}

// GET /
func (s Server) getDashboard(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	fromTime, toTime, aggDuration := calculateTimePeriod(from, to)
	candles, err := loadCandles(s.Engine, fromTime, toTime, aggDuration)
	if err != nil {
		log.Printf("[WARN] /: unable to load candles: %v", err)
		http.Error(w, fmt.Sprintf("unable to load candles: %v", err), http.StatusInternalServerError)
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

	t := template.Must(template.ParseFiles(fmt.Sprintf("%vwebapp/dashboard.html.tpl", s.webappPrefix)))
	err = t.Execute(w, result)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to execute template: %v", err), http.StatusInternalServerError)
		log.Printf("[WARN] dashboard: unable to execute template: %v", err)
		return
	}
	render.Status(r, http.StatusOK)
}

// GET /file_stats
func (s Server) getFileStats(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, fmt.Sprint("'filename' parameter is required"), http.StatusUnprocessableEntity)
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	fromTime, toTime, aggDuration := calculateTimePeriod(from, to)
	candles, err := loadCandles(s.Engine, fromTime, toTime, aggDuration)
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

	t := template.Must(template.ParseFiles(fmt.Sprintf("%vwebapp/file_stats.html.tpl", s.webappPrefix)))
	err = t.Execute(w, result)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to execute template: %v", err), http.StatusInternalServerError)
		log.Printf("[WARN] dashboard: unable to execute template: %v", err)
		return
	}
	render.Status(r, http.StatusOK)
}

// GET /chart
func (s Server) drawChart(w http.ResponseWriter, r *http.Request) {
	fromTime, toTime, aggDuration := calculateTimePeriod(
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
	)
	candles, err := loadCandles(s.Engine, fromTime, toTime, aggDuration)
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

// GET /api/candle
func (s Server) getCandle(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	if from == "" {
		sendErrorJSON(w, r, http.StatusBadRequest, errors.New("no 'from' field passed"), "")
		return
	}
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'from' field")
		return
	}
	toTime := time.Now()
	if to := r.URL.Query().Get("to"); to != "" {
		t, terr := time.Parse(time.RFC3339, to)
		if terr != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, terr, "can't parse 'to' field")
			return
		}
		toTime = t
	}
	duration := time.Minute
	if a := r.URL.Query().Get("aggregate"); a != "" {
		duration, err = time.ParseDuration(a)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'aggregate' field")
			return
		}
	}
	candles, err := loadCandles(s.Engine, fromTime, toTime, duration)
	if err != nil {
		sendErrorJSON(w, r, http.StatusBadRequest, err, "can't load candles")
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, candles)
}
