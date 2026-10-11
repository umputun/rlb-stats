package nginx

import (
	"errors"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/rlb-stats/app/store"
)

// Saver persists a Candle emitted by the aggregator. Mirrors the subset of store.Engine
// the consumer needs so tests can swap in a stub without the bolt dependency.
type Saver interface {
	Save(candle store.Candle) error
}

// Aggregator is the subset of store.Aggregator the consumer uses.
type Aggregator interface {
	Store(store.LogRecord) (store.Candle, bool)
}

// Consume reads lines from in, parses + filters each, and feeds the resulting records
// into agg. When agg emits a candle (on minute boundary), the candle is saved to engine.
// Returns when in is closed, so a Tailer that closes its channel on ctx cancel also
// drains this consumer cleanly.
//
// Parse errors are logged at DEBUG only: a line that fails parse is noise, not a bug
// (misconfigured nginx with the old format produces a steady stream of these until the
// log_format is updated).
func Consume(in <-chan string, filter Filter, agg Aggregator, engine Saver) {
	for line := range in {
		p, err := Parse(line)
		if err != nil {
			if errors.Is(err, ErrBadLine) {
				log.Printf("[DEBUG] skip unparseable line: %s", err)
			} else {
				log.Printf("[DEBUG] parse error: %s", err)
			}
			continue
		}
		if !filter.Keep(p) {
			continue
		}
		rec := p.ToRecord()
		candle, ok := agg.Store(rec)
		if !ok {
			continue
		}
		if err := engine.Save(candle); err != nil {
			log.Printf("[WARN] failed to save candle: %s", err)
		}
	}
}
