package store

import (
	"time"

	"github.com/umputun/rlb-stats/app/candle"
)

// Engine defines interface to save log entries and load candles
type Engine interface {
	Save(candle candle.Candle) (err error)
	Load(periodStart, periodEnd time.Time) (result []candle.Candle, err error)
}
