package candle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestNewAndUpdateCandle(t *testing.T) {

	candle := NewCandle()
	candle.update(parse.LogEntry{
		SourceIP:        "127.0.0.1",
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	})
	candleNew := Candle{
		Nodes: map[string]Info{
			"n6.radio-t.com": {1, time.Second, time.Second, time.Second, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {1, time.Second, time.Second, time.Second, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
		StartMinute: time.Time{},
	}

	assert.EqualValues(t, candle, candleNew, "matches loaded msg")

}
