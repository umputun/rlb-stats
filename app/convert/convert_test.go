package convert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestLogConversion(t *testing.T) {
	logEntries := []parse.LogEntry{
		{
			SourceIP:        "127.0.0.1", // first ip to first file
			FileName:        "/rtfiles/rt_podcast561.mp3",
			DestinationNode: "n6.radio-t.com",
			AnswerTime:      time.Second,
			Date:            time.Time{},
		}, {
			SourceIP:        "127.0.0.1", // first ip to second file
			FileName:        "/rtfiles/rt_podcast562.mp3",
			DestinationNode: "n6.radio-t.com",
			AnswerTime:      time.Second,
			Date:            time.Time{},
		}, {
			SourceIP:        "127.0.0.1", // first ip to second file
			FileName:        "/rtfiles/rt_podcast561.mp3",
			DestinationNode: "n6.radio-t.com",
			AnswerTime:      time.Second,
			Date:            time.Time{},
		}, {
			SourceIP:        "127.0.0.2", // second ip to first file, other minute
			FileName:        "/rtfiles/rt_podcast561.mp3",
			DestinationNode: "n7.radio-t.com",
			AnswerTime:      time.Second,
			Date:            time.Time{}.Add(time.Minute), // other minute
		},
	}

	candles := do(logEntries)
	testCandles := []candle.Candle{
		{
			Nodes: map[string]candle.Info{
				"n6.radio-t.com": {2, time.Second, time.Second, time.Second, map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
				"all":            {2, time.Second, time.Second, time.Second, map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		}, {
			Nodes: map[string]candle.Info{
				"n7.radio-t.com": {1, time.Second, time.Second, time.Second, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
				"all":            {1, time.Second, time.Second, time.Second, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			},
			StartMinute: time.Time{}.Add(time.Minute),
		},
	}

	assert.EqualValues(t, candles, testCandles, "matches loaded msg")

}
