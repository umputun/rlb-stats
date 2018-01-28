package parse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	const testString = `2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3`
	const badString = `gabbish`
	const defaultRegEx = `^(?P<Date>.+) - (?:.+) - (?P<FileName>.+) - (?P<SourceIP>.+) - (?:.+) - (?P<AnswerTime>.+) - https?://(?P<DestinationNode>.+?)/.+$`
	const badRegEx = `([`
	const wrongRegEx = `^(?P<FileName>.+)$`
	const defaultDateFormat = `2006/01/02 15:04:05`
	const badDateFormat = `gabbish`
	// normal flow
	parser, err := New(defaultRegEx, defaultDateFormat)
	assert.Nil(t, err, "parser created")
	assert.NotNil(t, parser.pattern, "parser pattern is present")

	entry, err := parser.Do(testString)
	assert.Nil(t, err, "string parsed")

	entryParsed := LogEntry{
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

	parser, err = New(defaultRegEx, badDateFormat)
	assert.Nil(t, err, "parser created")
	_, err = parser.Do(testString)
	assert.NotNil(t, err, "string not passed due to bad date")
	_, err = New(badRegEx, defaultDateFormat)
	assert.NotNil(t, err, "parser failed to be created due to bad regexp")
	_, err = New(wrongRegEx, defaultDateFormat)
	assert.NotNil(t, err, "parser failed to be created due to missing fields")
}
