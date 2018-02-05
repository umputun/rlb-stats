package convert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
	"github.com/umputun/rlb-stats/app/parse"
)

var testsTable = []struct {
	in     parse.LogEntry
	out    candle.Candle
	dumped bool
}{
	{parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	},
		candle.Candle{}, // empty, not yet dumped
		false},
	{parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to second file
		FileName:        "/rtfiles/rt_podcast562.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	},
		candle.Candle{}, // empty, not yet dumped
		false},
	{parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file, other node
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	},
		candle.Candle{}, // empty, not yet dumped
		false},
	{parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file, other minute
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{}.Add(time.Minute),
	},
		candle.Candle{ // from first 3 entries
			Nodes: map[string]candle.Info{
				"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
				"all":            {Volume: 2, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
		true},
	{parse.LogEntry{
		SourceIP:        "127.0.0.1", // access in third minute, will not be flushed into resultCandle
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{}.Add(time.Minute * 2),
	},
		candle.Candle{ // from 4th entry
			Nodes: map[string]candle.Info{
				"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
				"all":            {Volume: 1, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			},
			StartMinute: time.Time{}.Add(time.Minute),
		},
		true},
}

func TestLogConversion(t *testing.T) {
	for _, testPair := range testsTable {
		resultCandle, ok := Submit(testPair.in)
		assert.EqualValues(t, testPair.out, resultCandle, "candle match with expected output")
		assert.EqualValues(t, testPair.dumped, ok, "entry (not) dumped")
	}
}
