package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	// normal flow
	s, err := NewBolt("/tmp/test.bd")
	assert.Nil(t, err, "engine created")

	testCandle := candle.NewCandle()

	assert.Nil(t, s.Save(testCandle), "saved fine")
	savedCandle, err := s.Load(time.Time{}, time.Time{})
	assert.Nil(t, err, "key found")
	assert.EqualValues(t, testCandle, savedCandle[0], "matches loaded msg")

	assert.Nil(t, os.Remove("/tmp/test.bd"), "removed fine")

	// broken DB file
	s, err = NewBolt("/dev/null")
	assert.NotNil(t, err, "engine not created")
}
