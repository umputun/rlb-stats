package logservice

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/rlb-stats/app/store"
)

func TestLogExtraction(t *testing.T) {
	const regEx = `^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$`
	const defaultDateFormat = `2006/01/02 15:04:05`
	parser, _ := newParser(regEx, defaultDateFormat)
	lineExtractor := newLineExtractor()
	var entries []store.LogEntry

	go func() {
		_, err := lineExtractor.Write([]byte(fmt.Sprint("2017/09/17 12:54:54.095329 - GET - /api/v1/jump/fil")))
		assert.Nil(t, err, "half of line written")
		_, err = lineExtractor.Write([]byte(fmt.Sprint("es?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3\n")))
		assert.Nil(t, err, "other half of line written")
		_, err = lineExtractor.Write([]byte(fmt.Sprint("2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3\n2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3\n")))
		assert.Nil(t, err, "two lines written")
		close(lineExtractor.ch)
	}()

	for line := range lineExtractor.Ch() {
		entry, err := parser.Do(line)
		assert.Nil(t, err, "single line parsed")
		entries = append(entries, entry)
	}

	entryParsed := store.LogEntry{
		SourceIP:        "213.87.120.120",
		FileName:        "/api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Nanosecond * 710679,
		Date:            time.Date(2017, 9, 17, 12, 54, 54, 95329000, time.Local),
	}
	entriesParsed := []store.LogEntry{entryParsed, entryParsed, entryParsed}
	assert.Equal(t, entriesParsed, entries, "entries parsed")

	// check what LogStreamer.Go is able to be run
	var streamer = logStreamer{}
	streamer.Go()

}
