package web

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/rlb-stats/app/store"
)

func TestLoadCandles(t *testing.T) {
	e, teardown := startupEngine(t, false)
	defer teardown()

	// load empty results
	result, err := loadCandles(e, time.Time{}, time.Time{}.Add(time.Minute), time.Nanosecond)
	assert.Nil(t, err)
	assert.Equal(t, []store.Candle{}, result)

	// load non-empty results
	result, err = loadCandles(e, time.Unix(0, 0), time.Unix(0, 0).Add(time.Minute), time.Nanosecond)
	assert.Nil(t, err)
	assert.Equal(t, []store.Candle{storedCandle}, result)

	badE, _ := startupEngine(t, true)
	// load from non-existent files
	result, err = loadCandles(badE, time.Unix(0, 0), time.Unix(0, 0).Add(time.Minute), time.Nanosecond)
	assert.Nil(t, result)
	assert.EqualError(t, err, "test error")
}

func Test_limitCandleFiles(t *testing.T) {
	candle1 := store.Candle{
		Nodes: map[string]store.Info{
			"node1": {Volume: 123, Files: map[string]int{
				"/rtfiles/rt_podcast561.mp3": 2,
				"/rtfiles/rt_podcast562.mp3": 5,
				"/rtfiles/rt_podcast563.mp3": 1,
				"/rtfiles/rt_podcast564.mp3": 4},
			},
			"node2": {Volume: 45, Files: map[string]int{
				"/rtfiles/rt_podcast561.mp3": 1,
				"/rtfiles/rt_podcast562.mp3": 2,
				"/rtfiles/rt_podcast563.mp3": 10,
				"/rtfiles/rt_podcast564.mp3": 20},
			},
			"all": {Volume: 12, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
		StartMinute: time.Date(2022, time.April, 9, 20, 0, 0, 0, time.UTC),
	}

	candle2 := store.Candle{
		Nodes: map[string]store.Info{
			"node1": {Volume: 333, Files: map[string]int{
				"/rtfiles/rt_podcast561.mp3": 1,
				"/rtfiles/rt_podcast562.mp3": 2,
				"/rtfiles/rt_podcast563.mp3": 1,
				"/rtfiles/rt_podcast564.mp3": 4},
			},
			"node2": {Volume: 77, Files: map[string]int{
				"/rtfiles/rt_podcast561.mp3": 1,
				"/rtfiles/rt_podcast562.mp3": 2},
			},
			"all": {Volume: 11, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
		StartMinute: time.Date(2022, time.April, 9, 30, 0, 0, 0, time.UTC),
	}

	result := limitCandleFiles([]store.Candle{candle1, candle2}, 2)
	assert.Equal(t, 2, len(result))

	assert.Equal(t, time.Date(2022, time.April, 9, 20, 0, 0, 0, time.UTC), result[0].StartMinute)
	assert.EqualValues(t,
		store.Info{Volume: 123, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 5, "/rtfiles/rt_podcast564.mp3": 4}},
		result[0].Nodes["node1"])
	assert.EqualValues(t,
		store.Info{Volume: 45, Files: map[string]int{"/rtfiles/rt_podcast563.mp3": 10, "/rtfiles/rt_podcast564.mp3": 20}},
		result[0].Nodes["node2"])

	assert.EqualValues(t,
		store.Info{Volume: 333, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2, "/rtfiles/rt_podcast564.mp3": 4}},
		result[1].Nodes["node1"])
	assert.EqualValues(t,
		store.Info{Volume: 77, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 2}},
		result[1].Nodes["node2"])
}

var storedCandle = store.Candle{
	Nodes: map[string]store.Info{
		"n6.radio-t.com": {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
	StartMinute: time.Unix(0, 0),
}

func startupEngine(t *testing.T, badEngine bool) (engine store.Engine, teardown func()) {
	if badEngine {
		return MockDB{}, func() {}
	}
	file, err := ioutil.TempFile("/tmp/", "bolt_test.bd.")
	assert.Nil(t, err, "created temp file")
	engine, err = store.NewBolt(file.Name())
	assert.Nil(t, err, "engine created")
	assert.Nil(t, engine.Save(storedCandle), "saved fine")

	teardown = func() {
		_ = os.Remove(file.Name())
	}

	return engine, teardown
}

// MockDB implements store.Engine
type MockDB struct {
}

func (m MockDB) Save(candle store.Candle) error {
	return errors.New("test error")
}

func (m MockDB) Load(periodStart, periodEnd time.Time) ([]store.Candle, error) {
	return nil, errors.New("test error")
}
