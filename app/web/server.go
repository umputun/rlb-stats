package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/routegroup"

	"github.com/umputun/rlb-stats/app/store"
)

// Server is a web-server for rlb-stats REST API and UI
type Server struct {
	Engine       store.Engine
	Aggregator   *store.Aggregator
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
	srv := http.Server{
		Addr:              fmt.Sprintf("%v:%v", s.address, s.Port),
		Handler:           s.routes(),
		ReadHeaderTimeout: time.Second * 5,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
	}
	log.Printf("[WARN] http server terminated, %s", srv.ListenAndServe())
}

func (s *Server) routes() http.Handler {
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

		workDir, _ := os.Getwd()
		filesDir := filepath.Join(workDir, s.webappPrefix+"webapp")
		rUI.HandleFiles("/", http.Dir(filesDir))
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

func sendErrorJSON(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	log.Printf("[DEBUG] %s", details)
	render.Status(r, code)
	render.JSON(w, r, JSON{"error": err.Error(), "details": details})
}

// GET /api/candle?from=2022-04-06T05:06:17.041Z&to=2022-04-06T06:06:17.041Z&max_points=100&files=10
func (s *Server) getCandle(w http.ResponseWriter, r *http.Request) {
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
	aggDuration := toTime.Sub(fromTime).Truncate(time.Second) / 100
	if a := r.URL.Query().Get("aggregate"); a != "" {
		aggDuration, err = time.ParseDuration(a)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'aggregate' field")
			return
		}
	}
	if n := r.URL.Query().Get("max_points"); n != "" {
		i, err := strconv.ParseInt(n, 10, 8)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'max_points' field")
			return
		}
		aggDuration = toTime.Sub(fromTime).Truncate(time.Second) / time.Duration(i)
	}

	candles, err := loadCandles(r.Context(), s.Engine, fromTime, toTime, aggDuration)
	if err != nil {
		sendErrorJSON(w, r, http.StatusBadRequest, err, "can't load candles")
		return
	}
	if files := r.URL.Query().Get("files"); files != "" {
		filesN, err := strconv.Atoi(files)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'files' field")
			return
		}
		candles = limitCandleFiles(candles, filesN)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, candles)
}

// POST /api/insert
func (s *Server) insert(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var l store.LogRecord
	err := decoder.Decode(&l)
	if err != nil {
		sendErrorJSON(w, r, http.StatusBadRequest, err, "Problem decoding JSON")
		return
	}
	if l.Date.Equal(time.Time{}) {
		sendErrorJSON(w, r, http.StatusBadRequest, errors.New("missing field in JSON"), "ts")
		return
	}
	if l.DestHost == "" {
		sendErrorJSON(w, r, http.StatusBadRequest, errors.New("missing field in JSON"), "dest")
		return
	}
	if l.FileName == "" {
		sendErrorJSON(w, r, http.StatusBadRequest, errors.New("missing field in JSON"), "file_name")
		return
	}
	if l.FromIP == "" {
		sendErrorJSON(w, r, http.StatusBadRequest, errors.New("missing field in JSON"), "from_ip")
		return
	}
	err = saveLogRecord(s.Engine, s.Aggregator, l)
	if err != nil {
		sendErrorJSON(w, r, http.StatusInternalServerError, err, "Problem saving LogRecord")
		return
	}

	render.JSON(w, r, rest.JSON{"result": "ok"})
}
