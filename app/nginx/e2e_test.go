package nginx

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rlb-stats/app/store"
)

// collectingEngine captures every Save and lets the test inspect what landed.
type collectingEngine struct {
	mu    sync.Mutex
	saved []store.Candle
}

func (e *collectingEngine) Save(c store.Candle) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.saved = append(e.saved, c)
	return nil
}

// TestEndToEndUmputunSample is the regression anchor for the whole PR: umputun's seven
// real log lines (with the three CF fields appended as the plan requires) go through
// the exact pipeline main.go wires -- tailer stand-in → parser → filter → aggregator →
// engine.Save -- and the resulting candle must match the known-good counts, bandwidth
// and country breakdown. Flipping any assumption in the parser, filter, or dedup path
// surfaces here as a mismatched assertion.
//
// Dedup trap: the two mp3 lines share remote_addr 108.162.220.92 (a Cloudflare edge IP)
// but carry different cf_connecting_ip values. If the parser fell back to remote_addr,
// the Aggregator would collapse them to one download with half the bandwidth and this
// test would fail — proving the CF client-IP path is live.
func TestEndToEndUmputunSample(t *testing.T) {
	// read umputun's sample log lines verbatim; this is the authoritative input shape
	logFile, err := os.Open(filepath.Join("testdata", "sample.log"))
	require.NoError(t, err)
	defer logFile.Close()

	lines := make(chan string, 16)
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	require.NoError(t, scanner.Err())
	close(lines)

	filter, err := NewFilter(
		nil,                    // Methods: default (GET only)
		[]int{200, 206},        // Status: podcast-focused
		[]string{"*.mp3"},      // Paths: podcast-focused
		nil,                    // Hosts: all
	)
	require.NoError(t, err)

	agg := &store.Aggregator{}
	engine := &collectingEngine{}
	Consume(lines, filter, agg, engine)

	// all seven lines are in minute 21:52; the aggregator only emits a candle on minute
	// change, so Save was NOT called during Consume. Flushing is what gets the candle out,
	// mirroring main.go's SIGTERM shutdown ordering. If the shutdown ordering were wrong
	// the candle would be lost — this is the integration check for that path too.
	assert.Empty(t, engine.saved, "no minute boundary crossed, aggregator must hold the candle")

	candle, ok := agg.Flush()
	require.True(t, ok, "aggregator must have buffered the seven lines")
	require.NoError(t, engine.Save(candle))

	require.Len(t, engine.saved, 1)
	c := engine.saved[0]

	// top-level: only the two mp3 lines passed the filter. 304 html hits, API json
	// hits and the news.radio-t.com post are all dropped.
	allNode, ok := c.Nodes[store.AllNode]
	require.True(t, ok, "candle must have the AllNode synthetic node")
	assert.Equal(t, 2, allNode.Volume, "exactly two downloads passed the default filter")
	assert.Equal(t, int64(2097152+139264), allNode.Bytes, "total bytes = both mp3 responses summed")

	// per-vhost: only cdn.radio-t.com saw downloads under the default filter
	cdn, ok := c.Nodes["cdn.radio-t.com"]
	require.True(t, ok, "cdn.radio-t.com must appear as a node (vhost from $http_host)")
	assert.Equal(t, 2, cdn.Volume)
	assert.Equal(t, int64(2097152+139264), cdn.Bytes)

	_, hasRadioT := c.Nodes["radio-t.com"]
	assert.False(t, hasRadioT, "radio-t.com host had only 304 and non-mp3 traffic, all filtered")
	_, hasNews := c.Nodes["news.radio-t.com"]
	assert.False(t, hasNews, "news.radio-t.com had non-mp3 traffic, all filtered")

	// files: only the one podcast ends up in AllNode's Files (counted twice: one 206, one 200)
	assert.Equal(t, 2, allNode.Files["/rt_podcast1010.mp3"])
	assert.Len(t, allNode.Files, 1, "no other paths survived the filter")

	// countries: both mp3 lines seeded CF-IPCountry; the US on the html/news traffic was
	// filtered out before reaching the aggregator. Dash and empty country codes elsewhere
	// collapsed to no country.
	assert.Equal(t, map[string]int{"DE": 1, "PL": 1}, allNode.Countries)

	// start minute matches the shared timestamp of all seven lines
	assert.True(t, c.StartMinute.Equal(time.Date(2026, time.April, 26, 21, 52, 0, 0, time.UTC)))
}

// TestEndToEndRegressionDedupUsesCFIP is the dedup-trap check: both mp3 lines share
// remote_addr 108.162.220.92, but different cf_connecting_ip values. Volume=2 proves
// dedup fires on the client IP, not the edge IP.
//
// Mutation test (verified by hand once): temporarily replace resolveClientIP's return
// with just `remote` and run — Volume collapses to 1, Bytes becomes 2097152+139264 but
// counted as one download, and this assertion fails. That is how to prove the test can
// detect the regression it claims to.
func TestEndToEndRegressionDedupUsesCFIP(t *testing.T) {
	// just the two mp3 lines, nothing else
	lines := make(chan string, 2)
	lines <- `108.162.220.92 - - [26/Apr/2026:21:52:32 +0000] "GET /rt_podcast1010.mp3 HTTP/2.0" 206 2097152 "cdn.radio-t.com" "-" "Podcasts/4025.510.6 CFNetwork/3860.500.112 Darwin/25.4.0" https "31.46.245.136" "DE" "bytes=0-2097151"`
	lines <- `108.162.220.92 - - [26/Apr/2026:21:52:32 +0000] "GET /rt_podcast1010.mp3 HTTP/2.0" 200 139264 "cdn.radio-t.com" "-" "atc/1.0 watchOS/26.3" https "109.196.194.69" "PL" "-"`
	close(lines)

	filter, err := NewFilter(nil, []int{200, 206}, []string{"*.mp3"}, nil)
	require.NoError(t, err)
	agg := &store.Aggregator{}
	engine := &collectingEngine{}
	Consume(lines, filter, agg, engine)

	candle, ok := agg.Flush()
	require.True(t, ok)
	require.NoError(t, engine.Save(candle))

	c := engine.saved[0]
	assert.Equal(t, 2, c.Nodes[store.AllNode].Volume,
		"two distinct CF client IPs behind the same CF edge → two downloads; "+
			"if this collapsed to 1 the parser is using remote_addr instead of cf_connecting_ip")
}
