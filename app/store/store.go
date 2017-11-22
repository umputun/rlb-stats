package store

import (
	"time"

	"github.com/umputun/rlb-stats/app/parse"
)

// Candle contains one minute candle from log entries for that period
type Candle struct {
	Volume          int
	UniqIPsNum      int
	FileName        string
	DestinationNode string
	MinAnswerTime   time.Duration
	MeanAnswerTime  time.Duration
	MaxAnswerTime   time.Duration
	MinuteStart     time.Time
}

// Engine defines interface to save log entries and load candels
type Engine interface {
	Save(msg *parse.LogEntry) (err error)
	Load(periodStart, periodEnd time.Time) (result []Candle, err error)
}
