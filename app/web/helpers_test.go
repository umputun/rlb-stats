package web

import (
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
