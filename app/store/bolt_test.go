package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	s, err := NewBolt("/tmp/test.bd")
	assert.Nil(t, err, "engine created")

	candle := candle.NewCandle()

	assert.Nil(t, s.Save(candle), "saved fine")
	savedCandle, err := s.Load(time.Time{}, time.Time{})
	assert.Nil(t, err, "key found")
	assert.EqualValues(t, candle, savedCandle[0], "matches loaded msg")

	os.Remove("/tmp/test.bd")
}
