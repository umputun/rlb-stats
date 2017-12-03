package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	// create DB
	s, err := NewBolt("/tmp/test.bd", time.Second*3)
	assert.Nil(t, err, "engine created")

	// save first log entry
	logEntry := parse.LogEntry{
		SourceIP:        "127.0.0.1",
		FileName:        "rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second,
		Date:            time.Now().Round(0), // Round to drop monotonic clock, https://tip.golang.org/pkg/time/#hdr-Monotonic_Clocks
	}
	assert.Nil(t, s.Save(logEntry), "saved fine")
	savedEntry, err := s.loadLogEntry(time.Now(), time.Now().Add(time.Minute))
	assert.Nil(t, err, "key found")
	assert.EqualValues(t, logEntry, savedEntry[0], "matches loaded msg")
	// save second log entry
	secondLogEntry := parse.LogEntry{
		SourceIP:        "127.0.0.2",
		FileName:        "rtfiles/rt_podcast561.mp3",
		DestinationNode: "n7.radio-t.com",
		AnswerTime:      time.Second / 2,
		Date:            time.Now().Round(0), // Round to drop monotonic clock, https://tip.golang.org/pkg/time/#hdr-Monotonic_Clocks
	}
	assert.Nil(t, s.Save(secondLogEntry), "saved second log entry")
	thirdLogEntry := parse.LogEntry{
		SourceIP:        "127.0.0.3",
		FileName:        "rtfiles/rt_podcast563.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Second * 4,
		Date:            time.Now().Round(0), // Round to drop monotonic clock, https://tip.golang.org/pkg/time/#hdr-Monotonic_Clocks
	}
	assert.Nil(t, s.Save(thirdLogEntry), "saved third log entry")
	// wait for aggregation to work
	time.Sleep(time.Second * 5)
	nodeFromTwoEntries := Info{
		Volume:         2,
		MinAnswerTime:  logEntry.AnswerTime,
		MeanAnswerTime: (thirdLogEntry.AnswerTime + logEntry.AnswerTime) / 2,
		MaxAnswerTime:  thirdLogEntry.AnswerTime,
		Files: map[string]int{
			thirdLogEntry.FileName: 1,
			logEntry.FileName:      1,
		},
	}
	secondNode := Info{
		Volume:         1,
		MinAnswerTime:  secondLogEntry.AnswerTime,
		MeanAnswerTime: secondLogEntry.AnswerTime,
		MaxAnswerTime:  secondLogEntry.AnswerTime,
		Files:          map[string]int{secondLogEntry.FileName: 1},
	}
	sumNode := Info{
		Volume:         3,
		MinAnswerTime:  secondLogEntry.AnswerTime,
		MeanAnswerTime: (thirdLogEntry.AnswerTime + secondLogEntry.AnswerTime + logEntry.AnswerTime) / 3,
		MaxAnswerTime:  thirdLogEntry.AnswerTime,
		Files: map[string]int{
			logEntry.FileName:      2,
			thirdLogEntry.FileName: 1,
		},
	}
	candle := []Candle{Candle{Nodes: map[string]Info{
		"all":            sumNode,
		"n7.radio-t.com": secondNode,
		"n6.radio-t.com": nodeFromTwoEntries,
	},
		StartMinute: time.Date(
			logEntry.Date.Year(),
			logEntry.Date.Month(),
			logEntry.Date.Day(),
			logEntry.Date.Hour(),
			logEntry.Date.Minute(),
			0,
			0,
			logEntry.Date.Location())}}
	// load aggregated candle
	aggregatedCandles, err := s.Load(time.Now().Add(-time.Minute), time.Now())

	assert.Nil(t, err, "aggregated candle found")
	assert.NotNil(t, aggregatedCandles, "candle is created")
	assert.EqualValues(t, candle, aggregatedCandles, "candle is equal to expected")
	// check log entries were removed after aggregation
	savedEntry, err = s.loadLogEntry(time.Now(), time.Now().Add(time.Minute))
	assert.Nil(t, err, "log load successful")
	assert.Nil(t, savedEntry, "logs were removed after aggregation")
	os.Remove("/tmp/test.bd")
}
