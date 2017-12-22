package convert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestLogConversion(t *testing.T) {
	logEntry1 := parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	}
	logEntry2 := parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to second file
		FileName:        "/rtfiles/rt_podcast562.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	}

	logEntry3 := parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file, other node
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	}
	logEntry4 := parse.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file, other minute
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{}.Add(time.Minute),
	}
	logEntry5 := parse.LogEntry{
		SourceIP:        "127.0.0.1", // access in third minute, will not be flushed into resultCandle
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{}.Add(time.Minute * 2),
	}
	testCandle1 := candle.Candle{ // from first 3 entries
		Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {Volume: 2, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
		},
		StartMinute: time.Time{},
	}
	testCandle2 := candle.Candle{ // from 4th entry
		Nodes: map[string]candle.Info{
			"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
		StartMinute: time.Time{}.Add(time.Minute),
	}

	resultCandle, ok := Submit(logEntry1)
	assert.EqualValues(t, candle.Candle{}, resultCandle, "empty resultCandle in output")
	assert.False(t, ok, "resultCandle not dumped")
	resultCandle, ok = Submit(logEntry2)
	assert.EqualValues(t, candle.Candle{}, resultCandle, "empty resultCandle in output")
	assert.False(t, ok, "resultCandle not dumped")
	resultCandle, ok = Submit(logEntry3)
	assert.EqualValues(t, candle.Candle{}, resultCandle, "empty resultCandle in output")
	assert.False(t, ok, "resultCandle not dumped")
	resultCandle, ok = Submit(logEntry4)
	assert.EqualValues(t, testCandle1, resultCandle, "resultCandle from first 3 records in output")
	assert.True(t, ok, "resultCandle dumped")
	resultCandle, ok = Submit(logEntry5)
	assert.EqualValues(t, testCandle2, resultCandle, "resultCandle from 4th record in output")
	assert.True(t, ok, "resultCandle dumped")

}
