package nginx

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

type recordingEngine struct {
	saved     []store.Candle
	failNext  bool
	failedOne bool
}

func (r *recordingEngine) Save(c store.Candle) error {
	if r.failNext && !r.failedOne {
		r.failedOne = true
		return errors.New("boom")
	}
	r.saved = append(r.saved, c)
	return nil
}

func TestConsumeFiltersAndSaves(t *testing.T) {
	// three valid lines: two different minutes. default filter (GET-only, no status/path
	// restriction here) accepts all three. The aggregator emits a candle at the minute
	// boundary, which must land in the engine.
	line1 := `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"`
	line2 := `10.0.0.2 - - [26/Apr/2026:21:52:45 +0000] "GET /a.mp3 HTTP/2.0" 200 200 "cdn" "-" "x" https "-" "-" "-"`
	line3 := `10.0.0.3 - - [26/Apr/2026:21:53:01 +0000] "GET /b.mp3 HTTP/2.0" 200 300 "cdn" "-" "x" https "-" "-" "-"`

	in := make(chan string, 4)
	in <- line1
	in <- line2
	in <- line3
	close(in)

	agg := &store.Aggregator{}
	engine := &recordingEngine{}
	filter, err := NewFilter(nil, nil, nil, nil)
	require.NoError(t, err)

	Consume(in, filter, agg, engine)

	// after the minute changed from 21:52 → 21:53 the aggregator emits the 21:52 candle
	require.Len(t, engine.saved, 1, "one minute-boundary candle landed in the engine")
	c := engine.saved[0]
	assert.True(t, c.StartMinute.Equal(time.Date(2026, time.April, 26, 21, 52, 0, 0, time.UTC)),
		"candle minute is 21:52 UTC")
	assert.Equal(t, 2, c.Nodes["all"].Volume, "two 21:52 downloads")
	assert.Equal(t, int64(300), c.Nodes["all"].Bytes, "100+200 bytes")
}

func TestConsumeDropsBadLines(t *testing.T) {
	// a mix of good and bad lines: bad ones are logged + skipped, good ones are counted
	in := make(chan string, 3)
	in <- "not an nginx line"
	in <- `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"`
	in <- `10.0.0.2 - - [26/Apr/2026:21:53:30 +0000] "GET /a.mp3 HTTP/2.0" 200 50 "cdn" "-" "x" https "-" "-" "-"`
	close(in)

	agg := &store.Aggregator{}
	engine := &recordingEngine{}
	filter, err := NewFilter(nil, nil, nil, nil)
	require.NoError(t, err)
	Consume(in, filter, agg, engine)

	require.Len(t, engine.saved, 1, "the bad line is skipped, only the good 21:52 line landed in the emitted candle")
	assert.Equal(t, 1, engine.saved[0].Nodes["all"].Volume)
}

func TestConsumeAppliesFilter(t *testing.T) {
	// default filter: GET only, status 200,206, *.mp3. A 304 html hit and an api json
	// hit must be dropped; the one mp3 line is kept.
	in := make(chan string, 4)
	in <- `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /p/foo HTTP/2.0" 304 0 "radio-t.com" "-" "x" https "-" "-" "-"`
	in <- `10.0.0.2 - - [26/Apr/2026:21:52:31 +0000] "GET /site-api/last/1 HTTP/2.0" 200 100 "radio-t.com" "-" "x" https "-" "-" "-"`
	in <- `10.0.0.3 - - [26/Apr/2026:21:52:32 +0000] "GET /rt.mp3 HTTP/2.0" 206 999 "cdn" "-" "x" https "-" "-" "-"`
	in <- `10.0.0.4 - - [26/Apr/2026:21:53:00 +0000] "GET /other.mp3 HTTP/2.0" 200 123 "cdn" "-" "x" https "-" "-" "-"` // triggers minute flush
	close(in)

	agg := &store.Aggregator{}
	engine := &recordingEngine{}
	filter, err := NewFilter(nil, []int{200, 206}, []string{"*.mp3"}, nil)
	require.NoError(t, err)
	Consume(in, filter, agg, engine)

	require.Len(t, engine.saved, 1)
	c := engine.saved[0]
	assert.Equal(t, 1, c.Nodes["all"].Volume, "only the mp3 206 line passed the filter")
	assert.Equal(t, int64(999), c.Nodes["all"].Bytes)
}

func TestConsumeContinuesOnSaveError(t *testing.T) {
	in := make(chan string, 3)
	// two minute-change triggers so Save is called twice; the first fails, the second succeeds
	in <- `10.0.0.1 - - [26/Apr/2026:21:52:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"`
	in <- `10.0.0.2 - - [26/Apr/2026:21:53:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"` // flush 21:52, first Save fails
	in <- `10.0.0.3 - - [26/Apr/2026:21:54:30 +0000] "GET /a.mp3 HTTP/2.0" 200 100 "cdn" "-" "x" https "-" "-" "-"` // flush 21:53, Save ok
	close(in)

	agg := &store.Aggregator{}
	engine := &recordingEngine{failNext: true}
	filter, err := NewFilter(nil, nil, nil, nil)
	require.NoError(t, err)
	Consume(in, filter, agg, engine)

	require.Len(t, engine.saved, 1, "one save failed, one succeeded; consumer kept running")
	assert.True(t, engine.saved[0].StartMinute.Equal(time.Date(2026, time.April, 26, 21, 53, 0, 0, time.UTC)),
		"the saved candle is the 21:53 one (the 21:52 Save failed)")
}
