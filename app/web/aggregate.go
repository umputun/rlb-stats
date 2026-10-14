package web

import (
	"context"
	"sort"
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

// aggregateCandles buckets candles into aggInterval-wide windows aligned to the earliest
// candle, summing each window into a single candle. aggInterval is truncated to whole
// minutes with a one-minute floor. empty windows are omitted; output is ordered by time.
// a cancelled ctx returns the complete windows built so far, with no error.
func aggregateCandles(ctx context.Context, candles []store.Candle, aggInterval time.Duration) []store.Candle {
	// return empty slice instead of nil for empty result
	result := []store.Candle{}
	if len(candles) == 0 {
		return result
	}

	// protect against sub-minute intervals truncating to zero
	if aggInterval < time.Minute {
		aggInterval = time.Minute
	}
	aggInterval = aggInterval.Truncate(time.Minute)

	// work on a time-ordered copy. store.Engine returns candles sorted by StartMinute (bolt
	// key order, which firstIndexSince in dashboard.go also relies on); the sort here is
	// defensive, so the bucket origin and boundary cancellation depend neither on the caller
	// nor on the key range where lexicographic order stops matching numeric order (store/bolt.go)
	ordered := make([]store.Candle, len(candles))
	copy(ordered, candles)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].StartMinute.Before(ordered[j].StartMinute) })

	// bucket origin is the earliest candle; window k spans [origin+k*interval, origin+(k+1)*interval)
	origin := ordered[0].StartMinute

	// whole minutes per window and the origin as whole seconds: offsets are counted in seconds
	// rather than through time.Sub, whose Duration saturates at ~292 years and would collapse
	// candles that far apart into one window. aggInterval is minute-truncated above, so a window
	// offset is always a whole number of minutes and reconstructs exactly from the origin second.
	intervalMin := int64(aggInterval / time.Minute)
	originSec, originNsec := origin.Unix(), int64(origin.Nanosecond())

	// ordered is time-sorted, so the window index never decreases and every candle for a window
	// arrives in one run. one window is therefore open at a time: it closes when the index moves
	// on, which is also where cancellation is honoured, so an aborted request never yields a
	// partially-filled window. empty windows are dropped as they close.
	var open store.Candle
	var openIdx int64
	started := false
	closeWindow := func() {
		if started && len(open.Nodes) != 0 {
			result = append(result, open)
		}
	}

	for _, c := range ordered {
		// seconds from origin, floored: ordered is sorted so this is never negative, and a
		// sub-second remainder cannot move the minute it falls in
		sec := c.StartMinute.Unix() - originSec
		if int64(c.StartMinute.Nanosecond()) < originNsec {
			sec--
		}
		idx := sec / 60 / intervalMin
		if !started || idx != openIdx {
			closeWindow()
			select {
			case <-ctx.Done():
				return result
			default:
			}
			open = store.NewCandle()
			open.StartMinute = time.Unix(originSec+idx*intervalMin*60, originNsec).In(origin.Location())
			openIdx, started = idx, true
		}
		open = updateCandleAndDiscardTime(open, c)
	}
	closeWindow()

	return result
}

func updateCandleAndDiscardTime(source store.Candle, appendix store.Candle) store.Candle {
	for n := range appendix.Nodes {
		m, ok := source.Nodes[n]
		if !ok {
			m = store.NewInfo()
		}
		if m.Files == nil {
			m.Files = map[string]int{}
		}
		if m.Countries == nil {
			m.Countries = map[string]int{}
		}
		for file := range appendix.Nodes[n].Files {
			m.Files[file] += appendix.Nodes[n].Files[file]
		}
		for country, cnt := range appendix.Nodes[n].Countries {
			m.Countries[country] += cnt
		}
		m.Volume += appendix.Nodes[n].Volume
		m.Bytes += appendix.Nodes[n].Bytes
		source.Nodes[n] = m
	}
	return source
}
