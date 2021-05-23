package store

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	// normal flow
	file, err := ioutil.TempFile("/tmp/", "bolt_test.bd.")
	assert.Nil(t, err, "created temp file")

	s, err := NewBolt(file.Name())
	assert.Nil(t, err, "engine created")

	testCandle := NewCandle()
	testCandle.StartMinute = time.Unix(0, 0)

	assert.Nil(t, s.Save(testCandle), "saved fine")
	savedCandle, err := s.Load(time.Unix(0, 0), time.Unix(0, 0).Add(time.Hour))
	assert.Nil(t, err, "key found")
	require.NotEqual(t, []Candle{}, savedCandle, "key found")
	assert.EqualValues(t, testCandle, savedCandle[0], "matches loaded msg")

	assert.Nil(t, os.Remove(file.Name()), "removed fine")

	// broken DB file
	_, err = NewBolt("/dev/null")
	assert.NotNil(t, err, "engine not created")
}

func TestBolt_LoadStream(t *testing.T) {
	file, err := ioutil.TempFile("/tmp/", "bolt_test.bd.")
	assert.Nil(t, err, "created temp file")

	s, err := NewBolt(file.Name())
	assert.Nil(t, err, "engine created")

	// save 3 candles
	testCandle := NewCandle()
	testCandle.StartMinute = time.Unix(0, 0)
	assert.Nil(t, s.Save(testCandle), "saved fine")

	testCandle = NewCandle()
	testCandle.StartMinute = time.Unix(100, 0)
	assert.Nil(t, s.Save(testCandle), "saved fine")

	testCandle = NewCandle()
	testCandle.StartMinute = time.Unix(200, 0)
	assert.Nil(t, s.Save(testCandle), "saved fine")

	ch := s.LoadStream(time.Unix(0, 0), time.Unix(0, 0).Add(time.Hour))
	res := []Candle{}
	for c := range ch {
		res = append(res, c)
	}
	assert.Equal(t, 3, len(res), "all 3 candles loaded")
}
