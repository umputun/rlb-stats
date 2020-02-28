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
