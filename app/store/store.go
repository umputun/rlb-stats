package store

import (
	"time"
)

// Engine defines interface to save log entries and load candles
type Engine interface {
	Save(candle Candle) (err error)
	Load(periodStart, periodEnd time.Time) (result []Candle, err error)
}
