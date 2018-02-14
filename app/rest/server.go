package rest

import (
	"errors"
	"fmt"
	"net/http"

	"log"

	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/umputun/rlb-stats/app/aggregate"
	"github.com/umputun/rlb-stats/app/store"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// Server is a rest interface to storage
type Server struct {
	Engine store.Engine
	Port   int
}

func sendErrorJSON(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	log.Printf("[WARN] %s", details)
	render.Status(r, code)
	render.JSON(w, r, JSON{"error": err.Error(), "details": details})
}

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate rest server on port %v", s.Port)
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/candle", s.getCandle)
		r.Get("/grafana/", s.root) // https://github.com/grafana/simple-json-datasource implementation
		r.Get("/grafana/search", s.root)
		r.Get("/grafana/query", s.root)
		r.Get("/grafana/annotations", s.root)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", s.Port), r))
}

// GET /grafana/
// dummy answer
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v: %v", r.URL.Path, r.Method)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"status": "ok"})
}

// GET /candle
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
	candles, err := s.Engine.Load(fromTime, toTime)
	if err != nil {
		sendErrorJSON(w, r, http.StatusBadRequest, err, "can't load candles")
		return
	}
	if a := r.URL.Query().Get("aggregate"); a != "" {
		duration, err := time.ParseDuration(a)
		if err != nil {
			sendErrorJSON(w, r, http.StatusExpectationFailed, err, "can't parse 'aggregate' field")
			return
		}
		aggregate.Do(&candles, duration)
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, candles)
}
