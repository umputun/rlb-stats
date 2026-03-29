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

func TestFlush(t *testing.T) {
	t.Run("empty aggregator", func(t *testing.T) {
		parser := &Aggregator{}
		candle, ok := parser.Flush()
		assert.False(t, ok)
		assert.Equal(t, Candle{}, candle)
	})

	t.Run("buffered entries without minute boundary", func(t *testing.T) {
		parser := &Aggregator{}
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		// store entries within the same minute — no candle emitted
		_, ok := parser.Store(LogRecord{
			FromIP: "127.0.0.1", FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n6.radio-t.com", Date: baseTime,
		})
		assert.False(t, ok)

		_, ok = parser.Store(LogRecord{
			FromIP: "10.0.0.1", FileName: "/rtfiles/rt_podcast562.mp3",
			DestHost: "n7.radio-t.com", Date: baseTime.Add(30 * time.Second),
		})
		assert.False(t, ok)

		// flush should emit a candle with the buffered entries
		candle, ok := parser.Flush()
		assert.True(t, ok)
		assert.Equal(t, 3, len(candle.Nodes), "should have n6, n7, and all")
		assert.Equal(t, 1, candle.Nodes["n6.radio-t.com"].Volume)
		assert.Equal(t, 1, candle.Nodes["n7.radio-t.com"].Volume)
		assert.Equal(t, 2, candle.Nodes["all"].Volume)
		assert.Equal(t, 1, candle.Nodes["all"].Files["/rtfiles/rt_podcast561.mp3"])
		assert.Equal(t, 1, candle.Nodes["all"].Files["/rtfiles/rt_podcast562.mp3"])
	})

	t.Run("flush after store emitted candle", func(t *testing.T) {
		parser := &Aggregator{}
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		// first minute entry
		_, ok := parser.Store(LogRecord{
			FromIP: "127.0.0.1", FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n6.radio-t.com", Date: baseTime,
		})
		assert.False(t, ok)

		// second minute entry triggers candle for first minute
		_, ok = parser.Store(LogRecord{
			FromIP: "10.0.0.1", FileName: "/rtfiles/rt_podcast562.mp3",
			DestHost: "n7.radio-t.com", Date: baseTime.Add(time.Minute),
		})
		assert.True(t, ok)

		// flush should only emit the trailing entry from the second minute
		candle, ok := parser.Flush()
		assert.True(t, ok)
		assert.Equal(t, 2, len(candle.Nodes), "should have n7 and all")
		assert.Equal(t, 1, candle.Nodes["n7.radio-t.com"].Volume)
		assert.Equal(t, 1, candle.Nodes["all"].Volume)
	})

	t.Run("double flush returns false", func(t *testing.T) {
		parser := &Aggregator{}
		baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		parser.Store(LogRecord{
			FromIP: "127.0.0.1", FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n6.radio-t.com", Date: baseTime,
		})

		_, ok := parser.Flush()
		assert.True(t, ok)

		// second flush on empty buffer
		candle, ok := parser.Flush()
		assert.False(t, ok)
		assert.Equal(t, Candle{}, candle)
	})
}
