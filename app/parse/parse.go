package parse

import (
	"log"
	"regexp"
	"time"
)

const testString = "2017/09/17 12:54:54.095329 - GET - /api/v1/jump/files?url=/rtfiles/rt_podcast561.mp3 - 213.87.120.120 - 302 (70) - 710.679µs - http://n6.radio-t.com/rtfiles/rt_podcast561.mp3"

// LogEntry contains meaningful data extracted from single log line
type LogEntry struct {
	SourceIP        string
	FileName        string
	DestinationNode string
	AnswerTime      time.Duration
	Date            time.Time
}

// InitRegex checks if regular expression valid for parsing LogEntry
func InitRegex(line string) (regex *regexp.Regexp) {
	regex = regexp.MustCompile(line)
	// TODO: validate regex to make sure it doesn't contain wrong fields and contain right fields
	return
}

// Log parse log line into LogEntry
func Log(line string, regEx *regexp.Regexp) (entry LogEntry, err error) {
	result := regEx.FindStringSubmatch(line)
	n := regEx.SubexpNames()
	for i, m := range result {
		switch n[i] {
		case "": // first result is full string, do nothing with it
		case "SourceIP":
			entry.SourceIP = m
		case "FileName":
			entry.FileName = m
		case "DestinationNode":
			entry.DestinationNode = m
		case "AnswerTime":
			entry.AnswerTime, _ = time.ParseDuration(m)
		case "Date":
			entry.Date, _ = time.Parse("2006/01/02 15:04:05", m)
		default:
			log.Fatalf("[ERROR] unknown field '%s'", n[i])
		}
	}
	return
}
