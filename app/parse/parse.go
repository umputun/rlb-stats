package parse

import (
	"log"
	"regexp"
	"time"
)

// Parser contain validated regular expression for parsing logs
type Parser struct {
	pattern *regexp.Regexp
}

// LogEntry contains meaningful data extracted from single log line
type LogEntry struct {
	SourceIP        string
	FileName        string
	DestinationNode string
	AnswerTime      time.Duration
	Date            time.Time
}

// New checks if regular expression valid for parsing LogEntry
func New(regEx string) (parser *Parser, err error) {
	parser = &Parser{}
	parser.pattern, err = regexp.Compile(regEx)
	// TODO: validate regex to make sure it doesn't contain wrong fields and contain right fields
	return parser, err
}

// Do parse log line into LogEntry
func (p *Parser) Do(line string) (entry LogEntry, err error) {
	result := p.pattern.FindStringSubmatch(line)
	n := p.pattern.SubexpNames()
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
			entry.AnswerTime, err = time.ParseDuration(m)
		case "Date":
			entry.Date, err = time.Parse("2006/01/02 15:04:05", m)
		default:
			log.Fatalf("[ERROR] unknown field '%s'", n[i])
		}
		if err != nil {
			return entry, err
		}
	}
	return entry, err
}
