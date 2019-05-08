package logservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/rlb-stats/app/store"
)

var testsTable = []struct {
	in     store.LogEntry
	out    store.Candle
	dumped bool
}{
	{store.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	},
		store.Candle{}, // empty, not yet dumped
		false},
	{store.LogEntry{
		SourceIP:        "127.0.0.1", // access to second file
		FileName:        "/rtfiles/rt_podcast562.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	},
		store.Candle{}, // empty, not yet dumped
		false},
	{store.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file, other node
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{},
	},
		store.Candle{}, // empty, not yet dumped
		false},
	{store.LogEntry{
		SourceIP:        "127.0.0.1", // access to first file, other minute
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{}.Add(time.Minute),
	},
		store.Candle{ // from first 3 entries
			Nodes: map[string]store.Info{
				"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
				"all":            {Volume: 2, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
			},
			StartMinute: time.Time{},
		},
		true},
	{store.LogEntry{
		SourceIP:        "127.0.0.1", // access in third minute, will not be flushed into resultCandle
		FileName:        "/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Time{}.Add(time.Minute * 2),
	},
		store.Candle{ // from 4th entry
			Nodes: map[string]store.Info{
				"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
				"all":            {Volume: 1, MinAnswerTime: time.Second, MeanAnswerTime: time.Second, MaxAnswerTime: time.Second, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			},
			StartMinute: time.Time{}.Add(time.Minute),
		},
		true},
}

func Test(t *testing.T) {
	const testString = `2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3`
	const badString = `gabbish`
	const defaultRegEx = `^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$`
	const badRegEx = `([`
	const wrongRegEx = `^(?P<FileName>.+)$`
	const defaultDateFormat = `2006/01/02 15:04:05`
	const badDateFormat = `gabbish`

	// normal flow
	parser, err := newParser(defaultRegEx, defaultDateFormat)
	assert.Nil(t, err, "parser created")
	assert.NotNil(t, parser.pattern, "parser pattern is present")

	entry, err := parser.Do(testString)
	assert.Nil(t, err, "string parsed")

	entryParsed := store.LogEntry{
		SourceIP:        "213.87.120.120",
		FileName:        "/api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Nanosecond * 710679,
		Date:            time.Date(2017, 9, 17, 12, 54, 54, 95329000, time.Time{}.Location()),
	}

	assert.EqualValues(t, entryParsed, entry, "matches loaded msg")

	// bad cases
	_, err = parser.Do(badString)
	assert.NotNil(t, err, "string not parsed")

	parser, err = newParser(defaultRegEx, badDateFormat)
	assert.Nil(t, err, "parser created")
	_, err = parser.Do(testString)
	assert.NotNil(t, err, "string not passed due to bad date")
	_, err = newParser(badRegEx, defaultDateFormat)
	assert.NotNil(t, err, "parser failed to be created due to bad regexp")
	_, err = newParser(wrongRegEx, defaultDateFormat)
	assert.NotNil(t, err, "parser failed to be created due to missing fields")

	// test LogEntry conversion to Candle
	for _, testPair := range testsTable {
		resultCandle, ok := parser.submit(testPair.in)
		assert.EqualValues(t, testPair.out, resultCandle, "candle match with expected output")
		assert.EqualValues(t, testPair.dumped, ok, "entry (not) dumped")
	}
}
