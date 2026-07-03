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

	// work on a time-ordered copy: bucket-boundary cancellation relies on candles being
	// sorted, and store.Engine does not guarantee ordering. sorting here keeps the result
	// (and the cancellation contract) correct regardless of the engine's iteration order
	ordered := make([]store.Candle, len(candles))
	copy(ordered, candles)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].StartMinute.Before(ordered[j].StartMinute) })

	// bucket origin is the earliest candle; window k spans [origin+k*interval, origin+(k+1)*interval)
	origin := ordered[0].StartMinute

	buckets := map[time.Time]store.Candle{}
	order := make([]time.Time, 0, len(ordered))

	// emit returns the aggregated buckets ordered by time, skipping empty ones.
	// used both on normal completion and on cancellation, so an aborted request
	// still yields the buckets aggregated so far rather than an empty result.
	emit := func() []store.Candle {
		sort.Slice(order, func(i, j int) bool { return order[i].Before(order[j]) })
		for _, t := range order {
			if len(buckets[t].Nodes) != 0 {
				result = append(result, buckets[t])
			}
		}
		return result
	}

	for _, c := range ordered {
		// integer window index; the Duration holds the count, so idx*aggInterval is the offset
		idx := c.StartMinute.Sub(origin) / aggInterval
		bucketTime := origin.Add(idx * aggInterval)
		agg, ok := buckets[bucketTime]
		if !ok {
			// honour cancellation only at bucket boundaries so a cancelled request never
			// yields a partially-filled bucket; candles are processed in time order (sorted
			// above), so every earlier bucket is already complete here
			select {
			case <-ctx.Done():
				return emit()
			default:
			}
			agg = store.NewCandle()
			agg.StartMinute = bucketTime
			order = append(order, bucketTime)
		}
		buckets[bucketTime] = updateCandleAndDiscardTime(agg, c)
	}

	return emit()
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
