package rest

import (
	"fmt"
	"net/http"

	"log"

	"time"

	"encoding/json"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/umputun/rlb-stats/app/store"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// Server is a rest interface to storage
type Server struct {
	Engine store.Engine
	Port   int
}

// Run starts a web-server
func (s *Server) Run() {
	log.Printf("[INFO] activate rest server on port %v", s.Port)
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/candle", s.getCandle)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", s.Port), r))
}

// GET /v1/candle
func (s Server) getCandle(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	if from == "" {
		log.Print("[WARN] no 'from' in get request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": "no 'from' field passed"})
		return
	}
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		log.Print("[WARN] can't parse 'from' field")
		render.Status(r, http.StatusExpectationFailed)
		render.JSON(w, r, JSON{"error": err.Error()})
		return
	}
	toTime := time.Now()
	if to := r.URL.Query().Get("to"); to != "" {
		t, terr := time.Parse(time.RFC3339, to)
		if terr != nil {
			log.Print("[WARN] can't parse 'to' field")
			render.Status(r, http.StatusExpectationFailed)
			render.JSON(w, r, JSON{"error": terr.Error()})
			return
		}
		toTime = t
	}
	candles, err := s.Engine.Load(fromTime, toTime)
	if err != nil {
		log.Print("[WARN] can't load candles")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": err.Error()})
		return
	}
	candlesJSON, _ := json.Marshal(candles)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, candlesJSON)
}
