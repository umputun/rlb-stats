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
func (p *Aggregator) Store(newEntry LogRecord) (minuteCandle Candle, ok bool) {
	// drop seconds and nanoseconds from log date
	newEntry.Date = time.Date(
		newEntry.Date.Year(),
		newEntry.Date.Month(),
		newEntry.Date.Day(),
		newEntry.Date.Hour(),
		newEntry.Date.Minute(),
		0,
		0,
		newEntry.Date.Location())

	if len(p.entries) > 0 && !newEntry.Date.Equal(p.entries[len(p.entries)-1].Date) { // if there are existing entries and date changed
		minuteCandle = NewCandle()                  // then all previous entries have same date precise to the minute and will be written to single candle
		var deduplicate = make(map[string]struct{}) // deduplicate store ip-file map
		for _, entry := range p.entries {
			_, duplicate := deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.FromIP)]
			if !duplicate {
				minuteCandle.Update(entry)
				deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.FromIP)] = struct{}{}
			}
		}
		ok = true                 // candle is ready to be written
		p.entries = []LogRecord{} // clean written entries
	}
	p.entries = append(p.entries, newEntry)

	return minuteCandle, ok
}
