package web

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/wcharczuk/go-chart"

	"fmt"

	"net/url"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/umputun/rlb-stats/app/store"
)

// Server is a UI for rlb-stats rest backend
type Server struct {
	Port     int
	RESTPort int
}

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate UI web server on port %v", s.Port)
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
	if from == "" {
		from = "168h"
	}
	fromDuration, err := time.ParseDuration(from)
	if err != nil {
		// TODO write a warning about being unable to parse from field
		// TODO handle negative duration
		log.Print("[WARN] dashboard: can't parse from field")
		fromDuration = time.Hour * 24 * 7
	}
	fromTime := time.Now().Add(-fromDuration)
	toTime := time.Now()
	if to := r.URL.Query().Get("to"); to != "" {
		t, terr := time.ParseDuration(to)
		if terr != nil {
			log.Print("[WARN] dashboard: can't parse to field")
			//	TODO write a warning about being unable to parse to field
			//	TODO handle negative duration
		}
		toTime = toTime.Add(-t)
	}
	candles, err := loadCandles(fromTime, toTime, time.Minute)
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
			"/chart",
			"/chart",
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
	name := r.URL.Query().Get("name")
	if name == "" {
		log.Printf("no 'name' field passed")
		return
	}
	from := r.URL.Query().Get("from")
	if from == "" {
		from = "168h"
	}
	fromDuration, err := time.ParseDuration(from)
	if err != nil {
		// TODO write a warning about being unable to parse from field
		// TODO handle negative duration
		log.Print("[WARN] dashboard: can't parse from field")
		fromDuration = time.Hour * 24 * 7
	}
	fromTime := time.Now().Add(-fromDuration)
	toTime := time.Now()
	if to := r.URL.Query().Get("to"); to != "" {
		t, terr := time.ParseDuration(to)
		if terr != nil {
			log.Print("[WARN] dashboard: can't parse to field")
			//	TODO write a warning about being unable to parse to field
			//	TODO handle negative duration
		}
		toTime = toTime.Add(-t)
	}
	candles, err := loadCandles(fromTime, toTime, time.Minute)
	if err != nil {
		// TODO handle being unable to get candles
		log.Printf("[WARN] dashboard: unable to load candles: %v", err)
		return
	}

	result := struct {
		Name    string
		Charts  []string
		Candles []store.Candle
	}{
		name,
		[]string{"/chart"},
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
		Series: []chart.Series{
			chart.TimeSeries{
				Name: "A test series",
				XValues: []time.Time{
					time.Now().AddDate(0, 0, -10),
					time.Now().AddDate(0, 0, -9),
					time.Now().AddDate(0, 0, -8),
					time.Now().AddDate(0, 0, -7),
					time.Now().AddDate(0, 0, -6),
					time.Now().AddDate(0, 0, -5),
					time.Now().AddDate(0, 0, -4),
					time.Now().AddDate(0, 0, -3),
					time.Now().AddDate(0, 0, -2),
					time.Now().AddDate(0, 0, -1),
					time.Now(),
				},
				YValues: []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0},
			},
			chart.TimeSeries{
				Name: "A test series",
				XValues: []time.Time{
					time.Now().AddDate(0, 0, -10),
					time.Now().AddDate(0, 0, -9),
					time.Now().AddDate(0, 0, -8),
					time.Now().AddDate(0, 0, -7),
					time.Now().AddDate(0, 0, -6),
					time.Now().AddDate(0, 0, -5),
					time.Now().AddDate(0, 0, -4),
					time.Now().AddDate(0, 0, -3),
					time.Now().AddDate(0, 0, -2),
					time.Now().AddDate(0, 0, -1),
					time.Now(),
				},
				YValues: []float64{11.0, 10.0, 9.0, 8.0, 7.0, 6.0, 5.0, 4.0, 3.0, 2.0, 1.0},
			},
		},
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	w.Header().Set("Content-Type", "image/png")
	err := graph.Render(chart.PNG, w)
	if err != nil {
		// TODO handle graph generation problem
		log.Printf("[WARN] dashboard: unable to render graph: %v", err)
	}
}

func loadCandles(from time.Time, to time.Time, duration time.Duration) ([]store.Candle, error) {
	var result []store.Candle
	candleGetURL := fmt.Sprintf("http://localhost:8080/api/candle?from=%v&to=%v&aggregate=%v",
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
