package web

import (
	"encoding/json"
	"net/http"
	"time"

	"fmt"

	"log"

	"net/url"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/umputun/rlb-stats/app/store"
)

// UIRouter handle routes for dashboard
func UIRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/", getDashboard)
	return r
}

// GET /dashboard
func getDashboard(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	if from == "" {
		from = "1w"
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

	topFiles := getTop("files", candles, 10)
	topNodes := getTop("nodes", candles, 10)

	result := struct {
		candles  []store.Candle
		topFiles []volumeStats
		topNodes []volumeStats
	}{candles, topFiles, topNodes}

	// TODO return HTML template
	render.JSON(w, r, result)
	render.Status(r, http.StatusOK)
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
