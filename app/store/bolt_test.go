package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	s, err := NewBolt("/tmp/test.bd", time.Minute)
	assert.Nil(t, err, "engine created")

	logEntry := parse.LogEntry{SourceIP: "127.0.0.1", FileName: "rtfiles/rt_podcast561.mp3", DestinationNode: "n6.radio-t.com", AnswerTime: time.Second, Date: time.Now()}
	assert.Nil(t, s.Save(&logEntry), "saved fine")
	savedEntry, err := s.loadLogEntry(time.Now(), time.Now())
	assert.Nil(t, err, "key found")
	assert.EqualValues(t, logEntry, savedEntry[0], "matches loaded msg")

	os.Remove("/tmp/test.bd")
}
