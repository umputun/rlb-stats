package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/routegroup"

	"github.com/umputun/rlb-stats/app/store"
)

// LogAggregator buffers log records and emits minute candles
type LogAggregator interface {
	Store(store.LogRecord) (store.Candle, bool)
}

// Server is a web-server for rlb-stats REST API and UI
type Server struct {
	Engine     store.Engine
	Aggregator LogAggregator
	Port       int
	Version    string
	address    string // set only in tests
	templates  *template.Template
}

// JSON is a map alias, just for convenience
type JSON map[string]any

// Run starts a web-server and blocks until ctx is cancelled, then shuts down gracefully
func (s *Server) Run(ctx context.Context) {
	log.Printf("[INFO] activate web server on port %v", s.Port)
	srv := http.Server{
		Addr:              fmt.Sprintf("%v:%v", s.address, s.Port),
		Handler:           s.routes(),
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[WARN] http server terminated, %s", err)
		}
	}()

	<-ctx.Done()
	log.Printf("[INFO] shutting down http server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[WARN] http server shutdown error, %s", err)
	}
}

// badRequestError indicates a client input error (invalid query parameters, etc.)
type badRequestError struct{ msg string }

func (e *badRequestError) Error() string { return e.msg }

// validPeriods maps period parameter values to their durations
var validPeriods = map[string]time.Duration{
	"1h":  time.Hour,
	"12h": 12 * time.Hour,
	"24h": 24 * time.Hour,
	"10d": 10 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
	"all": 0, // special case, uses TimeRange
}

func (s *Server) parseTemplates() {
	funcMap := template.FuncMap{
		"list": func(args ...string) []string { return args },
		"inc":  func(i int) int { return i + 1 },
	}
	s.templates = template.Must(template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layout.html",
		"templates/dashboard.html",
		"templates/partials/*.html",
	))
}

func (s *Server) routes() http.Handler {
	s.parseTemplates()

	r := routegroup.New(http.NewServeMux())

	// Common middleware
	r.Use(rest.Recoverer(log.Default()))
	r.Use(rest.RealIP)
	r.Use(rest.AppInfo("rlb-stats", "umputun", s.Version), rest.Ping)

	// UI routes group
	infoLogger := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"))
	r.Route(func(rUI *routegroup.Bundle) {
		rUI.Use(infoLogger.Handler)
		rUI.Use(rest.Throttle(10))

		rUI.HandleFunc("GET /", s.dashboardPage)
		rUI.HandleFunc("GET /fragment/dashboard", s.dashboardFragment)
		// serve embedded static assets (charts.js, favicon.ico)
		staticSub, _ := fs.Sub(staticFS, "static")
		rUI.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, staticSub, "favicon.ico")
		})
		rUI.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	})

	// API routes group
	debugLogger := logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"))
	r.Route(func(rAPI *routegroup.Bundle) {
		rAPI.Use(debugLogger.Handler)

		rAPI.Mount("/api").Route(func(r *routegroup.Bundle) {
			r.With(rest.Throttle(10)).HandleFunc("GET /candle", s.getCandle)
			r.With(rest.Throttle(100)).HandleFunc("POST /insert", s.insert)
		})
	})

	return r
}

// dashboardPage renders the full dashboard HTML page (GET /)
func (s *Server) dashboardPage(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r)
	if err != nil {
		var bre *badRequestError
		if errors.As(err, &bre) {
			http.Error(w, bre.msg, http.StatusBadRequest)
		} else {
			log.Printf("[WARN] dashboard page error, %s", err)
			http.Error(w, "failed to load dashboard data", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("[WARN] failed to render dashboard page, %s", err)
	}
}

// dashboardFragment renders only the dashboard content for HTMX swap (GET /fragment/dashboard)
func (s *Server) dashboardFragment(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r)
	if err != nil {
		var bre *badRequestError
		if errors.As(err, &bre) {
			http.Error(w, bre.msg, http.StatusBadRequest)
		} else {
			log.Printf("[WARN] dashboard fragment error, %s", err)
			http.Error(w, "failed to load dashboard data", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "dashboard", data); err != nil {
		log.Printf("[WARN] failed to render dashboard fragment, %s", err)
	}
}

// buildDashboardData assembles DashboardData from candles for the requested period
func (s *Server) buildDashboardData(r *http.Request) (*DashboardData, error) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}
	dur, ok := validPeriods[period]
	if !ok {
		return nil, &badRequestError{msg: fmt.Sprintf("invalid period %q", period)}
	}

	ctx := r.Context()
	now := time.Now()

	// determine chart time range
	var chartFrom time.Time
	if period == "all" {
		oldest, _, err := s.Engine.TimeRange(ctx)
		if err != nil {
			return nil, fmt.Errorf("can't get time range: %w", err)
		}
		if oldest.IsZero() {
			chartFrom = now
		} else {
			chartFrom = oldest
		}
	} else {
		chartFrom = now.Add(-dur)
	}

	// load chart candles
	chartCandles, err := s.Engine.Load(ctx, chartFrom, now)
	if err != nil {
		return nil, fmt.Errorf("can't load candles: %w", err)
	}

	// compute summary cards for fixed periods
	type summarySpec struct {
		label string
		dur   time.Duration
		isAll bool
	}
	specs := []summarySpec{
		{label: "1 hour", dur: time.Hour},
		{label: "24 hours", dur: 24 * time.Hour},
		{label: "1 week", dur: 7 * 24 * time.Hour},
		{label: "1 month", dur: 30 * 24 * time.Hour},
		{label: "All time", isAll: true},
	}

	summaries := make([]SummaryData, 0, len(specs))
	for _, sp := range specs {
		var candles []store.Candle
		if sp.isAll {
			oldest, _, tErr := s.Engine.TimeRange(ctx)
			if tErr != nil {
				return nil, fmt.Errorf("can't get time range for summary: %w", tErr)
			}
			from := oldest
			if from.IsZero() {
				from = now
			}
			candles, err = s.Engine.Load(ctx, from, now)
		} else {
			candles, err = s.Engine.Load(ctx, now.Add(-sp.dur), now)
		}
		if err != nil {
			return nil, fmt.Errorf("can't load candles for %s: %w", sp.label, err)
		}
		summaries = append(summaries, SummaryData{Label: sp.label, Count: computeSummary(candles)})
	}

	// compute aggregation duration for chart (~100 points)
	chartRange := now.Sub(chartFrom)
	aggDuration := max(chartRange.Truncate(time.Second)/100, time.Minute)

	data := &DashboardData{
		Summaries:   summaries,
		ChartJSON:   buildChartData(ctx, chartCandles, aggDuration),
		Files:       computeTopFiles(chartCandles, 20),
		Nodes:       computeNodeStats(chartCandles),
		HeatmapJSON: buildHeatmapData(computeHeatmap(chartCandles)),
		Period:      period,
	}
	return data, nil
}

// GET /api/candle?from=2022-04-06T05:06:17.041Z&to=2022-04-06T06:06:17.041Z&max_points=100&files=10
func (s *Server) getCandle(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	if from == "" {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("no 'from' field passed"), "no 'from' field passed")
		return
	}
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't parse 'from' field")
		return
	}
	toTime := time.Now()
	if to := r.URL.Query().Get("to"); to != "" {
		t, terr := time.Parse(time.RFC3339, to)
		if terr != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, terr, "can't parse 'to' field")
			return
		}
		toTime = t
	}
	aggDuration := toTime.Sub(fromTime).Truncate(time.Second) / 100
	if a := r.URL.Query().Get("aggregate"); a != "" {
		aggDuration, err = time.ParseDuration(a)
		if err != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't parse 'aggregate' field")
			return
		}
	}
	if n := r.URL.Query().Get("max_points"); n != "" {
		i, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't parse 'max_points' field")
			return
		}
		if i <= 0 {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest,
				errors.New("max_points must be positive"), "can't parse 'max_points' field")
			return
		}
		aggDuration = toTime.Sub(fromTime).Truncate(time.Second) / time.Duration(i)
	}

	candles, err := loadCandles(r.Context(), s.Engine, fromTime, toTime, aggDuration)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't load candles")
		return
	}
	if files := r.URL.Query().Get("files"); files != "" {
		filesN, err := strconv.Atoi(files)
		if err != nil {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't parse 'files' field")
			return
		}
		candles = limitCandleFiles(candles, filesN)
	}

	rest.RenderJSON(w, candles)
}

// POST /api/insert
func (s *Server) insert(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var l store.LogRecord
	err := decoder.Decode(&l)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "Problem decoding JSON")
		return
	}
	if l.Date.Equal(time.Time{}) {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("missing field in JSON"), "missing field in JSON: ts")
		return
	}
	if l.DestHost == "" {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("missing field in JSON"), "missing field in JSON: dest")
		return
	}
	if l.FileName == "" {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("missing field in JSON"), "missing field in JSON: file_name")
		return
	}
	if l.FromIP == "" {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("missing field in JSON"), "missing field in JSON: from_ip")
		return
	}
	err = saveLogRecord(s.Engine, s.Aggregator, l)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "Problem saving LogRecord")
		return
	}

	rest.RenderJSON(w, rest.JSON{"result": "ok"})
}
