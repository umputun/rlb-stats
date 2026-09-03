package web

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

var testCandles = []store.Candle{
	{Nodes: map[string]store.Info{
		"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}},
	{Nodes: map[string]store.Info{
		"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute)},
	{Nodes: map[string]store.Info{
		"n7.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 2)},
	{Nodes: map[string]store.Info{
		"n7.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 3)},
	{Nodes: map[string]store.Info{
		"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 4)},
	{Nodes: map[string]store.Info{
		"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 5)},
	{Nodes: map[string]store.Info{
		"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 10)},
}

var resultCandles = map[int][]store.Candle{
	1: testCandles,
	2: {
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
		},
			StartMinute: time.Time{}},

		{Nodes: map[string]store.Info{
			"n7.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 2)},

		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 4)},

		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 10)},
	},
	3: {
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"n7.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {Volume: 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
		},
			StartMinute: time.Time{}},
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"n7.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {Volume: 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 3)},
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 9)},
	},
	5: {
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 3}},
			"n7.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {Volume: 5, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 3, "/rtfiles/rt_podcast562.mp3": 2}},
		},
			StartMinute: time.Time{}},
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 5)},
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 10)}},
	10: {
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 4, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 4}},
			"n7.radio-t.com": {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {Volume: 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 4, "/rtfiles/rt_podcast562.mp3": 2}},
		},
			StartMinute: time.Time{}},
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 10)}},
}

func TestAggregation(t *testing.T) {
	for i, result := range resultCandles {
		testSlice := aggregateCandles(context.Background(), testCandles, time.Duration(i)*time.Minute)
		assert.EqualValues(t, result, testSlice, "candle aggregate for %v minutes match with expected output", i)
	}
	// test less than 1 minute period which should have same output as 1 minute aggregation
	testSlice := aggregateCandles(context.Background(), testCandles, time.Nanosecond)
	assert.EqualValues(t, testCandles, testSlice, "candle aggregate for 1 nanosecond match with expected output")
}

func TestAggregateCandlesEdgeCases(t *testing.T) {
	t.Run("cancelled context returns without processing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got := aggregateCandles(ctx, testCandles, time.Minute)
		assert.Equal(t, []store.Candle{}, got, "cancelled before any bucket is filled")
	})

	t.Run("candles with no nodes produce no bucket", func(t *testing.T) {
		candles := []store.Candle{
			{StartMinute: time.Time{}, Nodes: map[string]store.Info{}},
			{StartMinute: time.Time{}.Add(time.Minute), Nodes: map[string]store.Info{
				"all": {Volume: 1, Files: map[string]int{"a.mp3": 1}},
			}},
		}
		got := aggregateCandles(context.Background(), candles, time.Minute)
		require.Len(t, got, 1, "empty-node candle is skipped, only the real one remains")
		assert.Equal(t, time.Time{}.Add(time.Minute), got[0].StartMinute)
	})

	t.Run("candles more than 292 years apart keep separate buckets", func(t *testing.T) {
		// time.Sub saturates at the maximum Duration, so an offset taken through it would put
		// every candle beyond that distance in one window
		mk := func(ts time.Time, file string) store.Candle {
			c := store.NewCandle()
			c.StartMinute = ts
			c.Nodes["all"] = store.Info{Volume: 1, Files: map[string]int{file: 1}}
			return c
		}
		origin := time.Date(1700, 1, 1, 0, 0, 0, 0, time.UTC)
		mid := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		last := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		require.Equal(t, mid.Sub(origin), last.Sub(origin), "both offsets saturate to the same Duration")

		tenYears := 10 * 365 * 24 * time.Hour
		got := aggregateCandles(context.Background(), []store.Candle{mk(origin, "a"), mk(mid, "b"), mk(last, "c")}, tenYears)
		require.Len(t, got, 3, "one bucket per candle, none merged")
		assert.Equal(t, map[string]int{"a": 1}, got[0].Nodes["all"].Files)
		assert.Equal(t, map[string]int{"b": 1}, got[1].Nodes["all"].Files)
		assert.Equal(t, map[string]int{"c": 1}, got[2].Nodes["all"].Files)
	})

	t.Run("unsorted input aggregates the same as sorted input", func(t *testing.T) {
		// the sort is what keeps the origin at the earliest candle; without it a later
		// candle first would put earlier ones at a negative offset and into a window that
		// is never emitted, so the result would be short and out of time order
		reversed := make([]store.Candle, len(testCandles))
		for i, c := range testCandles {
			reversed[len(testCandles)-1-i] = c
		}
		for minutes, want := range resultCandles {
			got := aggregateCandles(context.Background(), reversed, time.Duration(minutes)*time.Minute)
			assert.EqualValues(t, want, got, "reversed input, %d minute windows", minutes)
		}
	})

	t.Run("cancellation mid-window returns only complete windows", func(t *testing.T) {
		// the context reports Done only on its second check. checks happen when a window is
		// opened, so the first window (two candles) completes and the second is never started.
		// a per-candle check instead would trip on the second candle and emit the first window
		// with half its volume
		mk := func(ts time.Time) store.Candle {
			c := store.NewCandle()
			c.StartMinute = ts
			c.Nodes["all"] = store.Info{Volume: 1, Files: map[string]int{"a.mp3": 1}}
			return c
		}
		t0 := time.Time{}
		candles := []store.Candle{mk(t0), mk(t0.Add(time.Minute)), mk(t0.Add(5 * time.Minute))}
		ctx := &doneOnSecondCheck{Context: context.Background(), done: make(chan struct{})}

		got := aggregateCandles(ctx, candles, 5*time.Minute)
		require.Len(t, got, 1, "only the first, complete window is returned")
		assert.Equal(t, t0, got[0].StartMinute)
		assert.Equal(t, 2, got[0].Nodes["all"].Volume, "both candles of the first window are summed")
	})
}

// doneOnSecondCheck is a context whose Done channel closes on the second call to Done,
// so a caller that checks once per window sees cancellation exactly when the second window opens
type doneOnSecondCheck struct {
	context.Context
	checks int
	done   chan struct{}
}

func (c *doneOnSecondCheck) Done() <-chan struct{} {
	c.checks++
	if c.checks == 2 {
		close(c.done)
	}
	return c.done
}

func (c *doneOnSecondCheck) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}
