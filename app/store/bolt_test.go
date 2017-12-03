package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	s, err := NewBolt("/tmp/test.bd")
	assert.Nil(t, err, "engine created")

	candle := newCandle()
	candle.update(parse.LogEntry{
		SourceIP:        "127.0.0.1",
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Now().Round(0), // drop monotonic time by round
	})

	assert.Nil(t, s.Save(candle), "saved fine")
	savedCandle, err := s.Load(time.Now().Add(-time.Minute), time.Now())
	assert.Nil(t, err, "key found")
	assert.EqualValues(t, candle, savedCandle[0], "matches loaded msg")
	t.Logf("saved: %s\nloaded: %s", candle.StartMinute, savedCandle[0].StartMinute)

	os.Remove("/tmp/test.bd")
}
