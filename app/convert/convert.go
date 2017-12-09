package convert

import (
	"fmt"
	"time"

	"sort"

	"github.com/umputun/rlb-stats/app/candle"
	"github.com/umputun/rlb-stats/app/parse"
)

// do convert LogEntries into Candles,
// dropping duplicate IP-filename pairs each minute
func do(entries []parse.LogEntry) []candle.Candle {
	c := make(map[time.Time]candle.Candle)
	deduplicate := make(map[string]struct{})
	for _, entry := range entries {
		// drop seconds and nanoseconds from log date
		entry.Date = time.Date(
			entry.Date.Year(),
			entry.Date.Month(),
			entry.Date.Day(),
			entry.Date.Hour(),
			entry.Date.Minute(),
			0,
			0,
			entry.Date.Location())
		_, duplicate := deduplicate[fmt.Sprintf("%d-%s-%s", entry.Date.Unix(), entry.FileName, entry.SourceIP)]
		if !duplicate {
			newCandle, exists := c[entry.Date]
			if !exists {
				newCandle = candle.NewCandle()
			}
			newCandle.Update(entry)
			c[entry.Date] = newCandle
			deduplicate[fmt.Sprintf("%d-%s-%s", entry.Date.Unix(), entry.FileName, entry.SourceIP)] = struct{}{}
		}
	}

	var candles []candle.Candle

	for _, value := range c {
		candles = append(candles, value)
	}
	// sort the slice to make sure we return test data in same order
	sort.Slice(candles, func(i, j int) bool { return candles[i].StartMinute.Before(candles[j].StartMinute) })
	return candles
}
