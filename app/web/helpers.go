package web

import (
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

// loadCandles loads candles for given period of time aggregated by given duration
func loadCandles(engine store.Engine, from time.Time, to time.Time, aggDuration time.Duration) ([]store.Candle, error) {
	candles, err := engine.Load(from, to)
	if err != nil {
		return nil, err
	}
	if aggDuration != time.Minute {
		candles = aggregateCandles(candles, aggDuration)
	}
	return candles, nil
}

// saveLogRecord saves a log record to candle
func saveLogRecord(engine store.Engine, parser *store.Aggregator, l store.LogRecord) error {
	if candle, ok := parser.Store(l); ok { // Store returns ok in case candle is ready
		return engine.Save(candle)
	}
	return nil
}
