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
