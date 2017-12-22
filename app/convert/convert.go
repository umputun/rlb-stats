package convert

import (
	"fmt"
	"time"

	"github.com/umputun/rlb-stats/app/candle"
	"github.com/umputun/rlb-stats/app/parse"
)

// entries contain previous entries
var entries []parse.LogEntry

// Submit store LogEntry and return Candle when minute change
func Submit(newEntry parse.LogEntry) (candle.Candle, bool) {
	minuteCandle := candle.Candle{}
	ok := false
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

	if len(entries) > 0 && !newEntry.Date.Equal(entries[len(entries)-1].Date) { // if there are existing entries and date changed
		minuteCandle = candle.NewCandle()           // then all previous entries have same date precise to the minute and will be written to single candle
		var deduplicate = make(map[string]struct{}) // deduplicate store ip-file map
		for _, entry := range entries {
			_, duplicate := deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.SourceIP)]
			if !duplicate {
				minuteCandle.Update(entry)
				deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.SourceIP)] = struct{}{}
			}
		}
		ok = true                    // candle is ready to be written
		entries = []parse.LogEntry{} // clean written entries
	}
	entries = append(entries, newEntry)

	return minuteCandle, ok
}
