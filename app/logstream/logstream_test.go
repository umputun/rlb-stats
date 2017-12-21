package logstream

import (
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/parse"
)

func TestLogExtraction(t *testing.T) {
	regEx := `^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$`
	parser, _ := parse.New(regEx)
	lineExtractor := NewLineExtractor()
	var entries []parse.LogEntry

	go func() {
		lineExtractor.Write([]byte(fmt.Sprint("2017/09/17 12:54:54.095329 - GET - /api/v1/jump/fil")))
		lineExtractor.Write([]byte(fmt.Sprint("es?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3\n")))
		lineExtractor.Write([]byte(fmt.Sprint("2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3\n2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3")))
		close(lineExtractor.ch)
	}()

	for line := range lineExtractor.ch {
		entry, err := parser.Do(line)
		assert.Nil(t, err, "single line parsed")
		entries = append(entries, entry)
	}

	entryParsed := parse.LogEntry{
		SourceIP:        "213.87.120.120",
		FileName:        "/api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3",
		DestinationNode: "n6.radio-t.com",
		AnswerTime:      time.Nanosecond * 710679,
		Date:            time.Date(2017, 9, 17, 12, 54, 54, 95329000, time.Time{}.Location()),
	}
	entriesParsed := []parse.LogEntry{entryParsed, entryParsed, entryParsed}
	assert.Equal(t, entriesParsed, entries, "entries parsed")

}
