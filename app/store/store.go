package store

import (
	"log"
	"regexp"
	"time"
)

// LogEntry contains meaningful data extracted from single log line
type LogEntry struct {
	SourceIP        string
	FileName        string
	DestinationNode string
	AnswerTime      time.Duration
	Date            time.Time
}

// Candle contains one minute candle from log entries for that period
type Candle struct {
	UniqIPsNum      string
	FileName        string
	DestinationNode string
	MeanAnswerTime  time.Duration
	MinuteStart     time.Time
}

// ParseLine parse log line into LogEntry
func ParseLine(line string, regEx *regexp.Regexp) (entry LogEntry, err error) {
	result := regEx.FindStringSubmatch(line)
	n := regEx.SubexpNames()
	for i, m := range result {
		switch n[i] {
		case "": // Do nothing, first result is full string
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
			// TODO: validate regex to make sure it doesn't contain wrong fields and contain right fields
		default:
			log.Fatalf("[ERROR] unknown field '%s'", n[i])
		}
	}
	return
}

// TODO: write entries to boltDB

// Engine defines interface to save log entries and load candels
type Engine interface {
	Save(msg *LogEntry) (err error)
	Load(periodStart, periodEnd time.Time) (resutl *Candle, err error)
}
