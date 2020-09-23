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

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"

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
	err := http.ListenAndServe(fmt.Sprintf("%v:%v", s.address, s.Port), s.routes())
	log.Printf("[WARN] http server terminated, %s", err)
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(5 * 60 * time.Second))
	r.Use(rest.AppInfo("rlb-stats", "umputun", s.Version), rest.Ping)

	r.Group(func(rUI chi.Router) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[INFO]"))
		rUI.Use(l.Handler)
		rUI.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
		workDir, _ := os.Getwd()
		filesDir := filepath.Join(workDir, s.webappPrefix+"webapp")
		fileServer(rUI, "/", http.Dir(filesDir))
	})

	r.Group(func(rAPI chi.Router) {
		l := logger.New(logger.Log(log.Default()), logger.Prefix("[DEBUG]"))
		rAPI.Use(l.Handler)
		rAPI.Route("/api", func(r chi.Router) {
			r.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil))).Get("/candle", s.getCandle)
			r.With(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(100, nil))).Post("/insert", s.insert)
		})
	})

	return r
}

func sendErrorJSON(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	log.Printf("[DEBUG] %s", details)
	render.Status(r, code)
	render.JSON(w, r, JSON{"error": err.Error(), "details": details})
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
	aggDuration := toTime.Sub(fromTime).Truncate(time.Second) / 100
	if a := r.URL.Query().Get("aggregate"); a != "" {
		aggDuration, err = time.ParseDuration(a)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'aggregate' field")
			return
		}
	}
	if n := r.URL.Query().Get("max_points"); n != "" {
		i, err := strconv.ParseUint(n, 10, 8)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'max_points' field")
			return
		}
		aggDuration = toTime.Sub(fromTime).Truncate(time.Second) / time.Duration(i)
	}

	candles, err := loadCandles(s.Engine, fromTime, toTime, aggDuration)
	if err != nil {
		sendErrorJSON(w, r, http.StatusBadRequest, err, "can't load candles")
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, candles)
}

// POST /api/insert
func (s Server) insert(w http.ResponseWriter, r *http.Request) {
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

	render.JSON(w, r, struct {
		Result string `json:"result"`
	}{"ok"})
}

// fileServer conveniently sets up a http.fileServer handler to serve
// static files from a http.FileSystem.
func fileServer(r chi.Router, path string, root http.FileSystem) {
	fs := http.StripPrefix(path, http.FileServer(root))
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
