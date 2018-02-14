package candle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/parse"
)

var testsTable = []struct {
	in  parse.LogEntry
	out Candle
}{
	{parse.LogEntry{
		SourceIP:        "127.0.0.1",
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second * 3,
		Date:            time.Time{},
	},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
				"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
	},
	{parse.LogEntry{
		SourceIP:        "127.0.0.3",
		FileName:        "/rtfiles/rt_podcast562.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second * 4,
		Date:            time.Time{},
	},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
				"n7.radio-t.com": {1, time.Second * 4, time.Second * 4, time.Second * 4, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
				"all":            {2, time.Second * 3, time.Second * (3 + 4) / 2, time.Second * 4, map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
	},
	{parse.LogEntry{
		SourceIP:        "127.0.0.2",
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second * 2,
		Date:            time.Time{},
	},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
				"n7.radio-t.com": {2, time.Second * 2, time.Second * 3, time.Second * 4, map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
				"all":            {3, time.Second * 2, time.Second * (3 + 4 + 2) / 3, time.Second * 4, map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
	},
}

func TestNewAndUpdateCandle(t *testing.T) {

	candle := NewCandle()
	for _, testPair := range testsTable {
		candle.Update(testPair.in)
		assert.EqualValues(t, testPair.out, candle, "candle match with expected output")
	}

}
