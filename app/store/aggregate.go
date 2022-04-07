package store

import (
	"fmt"
	"time"
)

// Aggregator stores single log records into minute candles, returning candle for previous minute when
// first log entry for new minute appears
type Aggregator struct {
	entries []LogRecord // used to store entries which are not yet dumped into candles
}

// Store LogRecord into temp storage and return Candle when minute change,
// counting multiple entries with same FromIP and FileName as single data point
func (p *Aggregator) Store(entry LogRecord) (minuteCandle Candle, ok bool) {

	// drop seconds and nanoseconds from log date to match candle's 1min resolution
	entry.Date = time.Date(entry.Date.Year(), entry.Date.Month(), entry.Date.Day(), entry.Date.Hour(), entry.Date.Minute(),
		0, 0, entry.Date.Location())

	// if there are existing entries and date changed
	if len(p.entries) != 0 && !entry.Date.Equal(p.entries[len(p.entries)-1].Date) {
		// then all previous entries have same date precise to the minute and will be written to single candle
		minuteCandle = NewCandle()
		var deduplicate = map[string]struct{}{} // deduplicate store ip-file map
		for _, entry := range p.entries {
			if _, dup := deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.FromIP)]; dup {
				continue
			}
			minuteCandle.Update(entry)
			deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.FromIP)] = struct{}{}
		}
		ok = true                 // candle is ready to be written
		p.entries = []LogRecord{} // clean written entries
	}

	p.entries = append(p.entries, entry)
	return minuteCandle, ok
}
