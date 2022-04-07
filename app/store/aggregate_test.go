package store

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testsTable = []struct {
	in     LogRecord
	out    Candle
	dumped bool
}{
	{ // 0
		LogRecord{
			FromIP:   "127.0.0.1", // access to first file
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n6.radio-t.com",
			Date:     time.Time{},
		},
		Candle{}, // empty, not yet dumped
		false,
	},

	{ // 1
		LogRecord{
			FromIP:   "127.0.0.1", // access to second file
			FileName: "/rtfiles/rt_podcast562.mp3",
			DestHost: "n6.radio-t.com",
			Date:     time.Time{},
		},
		Candle{}, // empty, not yet dumped
		false,
	},

	{ // 2
		LogRecord{
			FromIP:   "127.0.0.1", // access to first file, other node
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{},
		},
		Candle{}, // empty, not yet dumped
		false,
	},

	{ // 3
		LogRecord{
			FromIP:   "127.0.0.1", // access to first file, other minute
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{}.Add(time.Minute),
		},
		Candle{ // from first 3 entries
			Nodes: map[string]Info{
				"n6.radio-t.com": {Volume: 2, Files: map[string]int{}},
				"all":            {Volume: 2, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
		true,
	},

	{ // 4
		LogRecord{
			FromIP:   "127.0.0.1", // access in third minute, will not be flushed into resultCandle
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{}.Add(time.Minute * 2),
		},
		Candle{ // from 4th entry
			Nodes: map[string]Info{
				"n7.radio-t.com": {Volume: 1, Files: map[string]int{}},
				"all":            {Volume: 1, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			},
			StartMinute: time.Time{}.Add(time.Minute),
		},
		true,
	},
}

func TestParsing(t *testing.T) {
	parser := &Aggregator{}

	// test LogRecord conversion to Candle
	for i, tt := range testsTable {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			resultCandle, ok := parser.Store(tt.in)
			assert.EqualValues(t, tt.out, resultCandle, "candle match with expected output")
			assert.EqualValues(t, tt.dumped, ok, "entry (not) dumped")
		})
	}
}
