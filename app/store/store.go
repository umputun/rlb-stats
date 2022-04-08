package store

import (
	"context"
	"time"
)

// Engine defines interface to save log entries and load candles
type Engine interface {
	Save(candle Candle) (err error)
	Load(ctx context.Context, periodStart, periodEnd time.Time) (result []Candle, err error)
}
