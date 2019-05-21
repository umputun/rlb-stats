package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wcharczuk/go-chart"

	"github.com/umputun/rlb-stats/app/store"
)

func TestTime(t *testing.T) {
	var testSet = map[int]struct {
		from, to         string
		fromTime, toTime time.Time
		fromDuration     time.Duration
	}{
		1: {"", "", time.Now().Add(time.Hour * -168), time.Now(), time.Hour * 168 / 10},
		2: {"10h", "", time.Now().Add(time.Hour * -10), time.Now(), time.Hour * 10 / 10},
		3: {"168h", "8h", time.Now().Add(time.Hour * -168), time.Now().Add(time.Hour * -8), time.Hour * 160 / 10},
		4: {"wrong", "wrong", time.Now().Add(time.Hour * -168), time.Now(), time.Hour * 168 / 10},
	}
	for _, data := range testSet {
		fromTime, toTime, fromDuration := calculateTimePeriod(data.from, data.to)
		assert.EqualValues(t, data.fromTime.Truncate(time.Minute), fromTime.Truncate(time.Minute), "fromTime match expected")
		assert.EqualValues(t, data.toTime.Truncate(time.Minute), toTime.Truncate(time.Minute), "toTime match expected")
		assert.EqualValues(t, data.fromDuration, fromDuration, "steps duration match expected")
	}
}

func TestSeriesGeneration(t *testing.T) {
	var testSet = map[int]struct {
		candles     []store.Candle
		fromTime    time.Time
		toTime      time.Time
		aggDuration time.Duration
		qType       string
		filename    string
		result      []chart.Series
	}{
		1: {[]store.Candle{}, time.Time{}, time.Time{}.Add(time.Hour), time.Hour, "by_server",
			"", nil,
		},
	}
	for i, data := range testSet {
		result := prepareSeries(data.candles, data.fromTime, data.toTime, data.aggDuration, data.qType, data.filename)
		assert.EqualValues(t, data.result, result, "generated series for set %v match expected", i)
	}
}

func TestLoadCandlesEmptyResponse(t *testing.T) {
	// Start a local HTTP server with valid response
	goodServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Test request parameters
		assert.Equal(t, "/api/candle?from=0001-01-01T00%3A00%3A00Z&to=0001-01-01T00%3A01%3A00Z&aggregate=1ns", r.URL.String())
		assert.Equal(t, "0001-01-01T00:00:00Z", r.URL.Query().Get("from"))
		assert.Equal(t, "0001-01-01T00:01:00Z", r.URL.Query().Get("to"))
		// Send response to be tested
		_, err := rw.Write([]byte(`[]`))
		assert.Nil(t, err)
	}))
	// Close the server when test finishes
	defer goodServer.Close()

	apiClient.apiURL = goodServer.URL
	apiClient.httpClient = goodServer.Client()

	// load empty results
	result, err := loadCandles(time.Time{}, time.Time{}.Add(time.Minute), time.Nanosecond)
	assert.Equal(t, []store.Candle{}, result)
	assert.Nil(t, err)
}

func TestLoadCandlesBadResponse(t *testing.T) {
	// Start a local HTTP server with valid response
	badServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Send response to be tested
		_, err := rw.Write([]byte(`{}`))
		assert.Nil(t, err)
	}))
	// Close the server when test finishes
	defer badServer.Close()

	apiClient.apiURL = badServer.URL
	apiClient.httpClient = badServer.Client()

	// load wrong JSON response
	result, err := loadCandles(time.Time{}, time.Time{}.Add(time.Minute), time.Nanosecond)
	assert.Equal(t, []store.Candle(nil), result)
	assert.IsType(t, &json.UnmarshalTypeError{}, err)

	// try to load from empty URL
	apiClient.apiURL = ""
	_, err = loadCandles(time.Time{}, time.Time{}.Add(time.Minute), time.Nanosecond)
	assert.NotNil(t, err)
}
