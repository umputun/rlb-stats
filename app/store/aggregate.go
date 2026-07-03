package store

import (
	"sync"
	"time"
)

// Aggregator stores single log records into minute candles, returning candle for previous minute when
// first log entry for new minute appears
type Aggregator struct {
	mu      sync.Mutex
	entries []LogRecord // used to store entries which are not yet dumped into candles
}

// Store LogRecord into temp storage and return Candle when minute change,
// counting multiple entries with same FromIP and FileName as single data point
func (p *Aggregator) Store(entry LogRecord) (minuteCandle Candle, ok bool) {

	// drop seconds and nanoseconds from log date to match candle's 1min resolution
	entry.Date = time.Date(entry.Date.Year(), entry.Date.Month(), entry.Date.Day(), entry.Date.Hour(), entry.Date.Minute(),
		0, 0, entry.Date.Location())

	p.mu.Lock()
	defer p.mu.Unlock()

	// if there are existing entries and date changed, all previous entries share the same minute
	// and collapse into a single candle
	if len(p.entries) != 0 && !entry.Date.Equal(p.entries[len(p.entries)-1].Date) {
		minuteCandle = buildCandle(p.entries)
		ok = true                 // candle is ready to be written
		p.entries = []LogRecord{} // clean written entries
	}

	p.entries = append(p.entries, entry)
	return minuteCandle, ok
}

// Flush emits a candle from any buffered entries without waiting for a minute boundary.
// returns false if no entries are buffered.
func (p *Aggregator) Flush() (minuteCandle Candle, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.entries) == 0 {
		return Candle{}, false
	}

	minuteCandle = buildCandle(p.entries)
	p.entries = nil
	return minuteCandle, true
}

// fileIP identifies a unique file+ip pair for deduplication; a struct key avoids
// the collisions a concatenated string key would allow (e.g. "a-b"+"c" vs "a"+"b-c")
type fileIP struct {
	file string
	ip   string
}

// buildCandle collapses buffered entries into a single candle, counting each
// unique file+ip pair once
func buildCandle(entries []LogRecord) Candle {
	minuteCandle := NewCandle()
	deduplicate := map[fileIP]struct{}{}
	for _, entry := range entries {
		key := fileIP{file: entry.FileName, ip: entry.FromIP}
		if _, dup := deduplicate[key]; dup {
			continue
		}
		minuteCandle.Update(entry)
		deduplicate[key] = struct{}{}
	}
	return minuteCandle
}
