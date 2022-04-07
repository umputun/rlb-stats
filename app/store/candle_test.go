package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var logsTestsTable = []struct {
	in  LogRecord
	out Candle
}{
	{
		LogRecord{
			FromIP:   "127.0.0.1",
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n6.radio-t.com",
			Date:     time.Time{},
		},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": {1, map[string]int{}},
				"all":            {1, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
	},
	{
		LogRecord{
			FromIP:   "127.0.0.3",
			FileName: "/rtfiles/rt_podcast562.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{},
		},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": {1, map[string]int{}},
				"n7.radio-t.com": {1, map[string]int{}},
				"all":            {2, map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
	},
	{
		LogRecord{
			FromIP:   "127.0.0.2",
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{},
		},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": {1, map[string]int{}},
				"n7.radio-t.com": {2, map[string]int{}},
				"all":            {3, map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
	},
}

func TestNewAndUpdateCandle(t *testing.T) {
	candle := NewCandle()
	for _, testPair := range logsTestsTable {
		candle.Update(testPair.in)
		assert.EqualValues(t, testPair.out, candle, "candle match with expected output")
	}
}
