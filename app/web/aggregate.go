package web

import (
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

// aggregateCandles takes candles from input, and aggregate them by aggInterval truncated to minutes
func aggregateCandles(candles []store.Candle, aggInterval time.Duration) []store.Candle {
	// initialize result in this way to return empty slice instead of nil for empty result
	result := []store.Candle{}

	// protect against less than 1m interval truncated to zero
	if aggInterval < time.Minute {
		aggInterval = time.Minute
	}
	aggInterval = aggInterval.Truncate(time.Minute)

	var firstDate, lastDate = time.Now(), time.Time{}
	for _, c := range candles {
		if c.StartMinute.Before(firstDate) {
			firstDate = c.StartMinute
		}
		if c.StartMinute.After(lastDate) {
			lastDate = c.StartMinute
		}
	}

	for aggTime := firstDate; aggTime.Before(lastDate.Add(aggInterval)); aggTime = aggTime.Add(aggInterval) {
		minuteCandle := store.NewCandle()
		minuteCandle.StartMinute = aggTime
		for _, c := range candles {
			if c.StartMinute == aggTime || c.StartMinute.After(aggTime) && c.StartMinute.Before(aggTime.Add(aggInterval)) {
				c = updateCandleAndDiscardTime(minuteCandle, c)
			}
		}
		if len(minuteCandle.Nodes) != 0 {
			result = append(result, minuteCandle)
		}
	}
	return result
}

func updateCandleAndDiscardTime(source store.Candle, appendix store.Candle) store.Candle {
	for n := range appendix.Nodes {
		m, ok := source.Nodes[n]
		if !ok {
			m = store.NewInfo()
		}
		for file := range appendix.Nodes[n].Files {
			m.Files[file] += appendix.Nodes[n].Files[file]
		}
		m.Volume += appendix.Nodes[n].Volume
		source.Nodes[n] = m
	}
	return source
}
