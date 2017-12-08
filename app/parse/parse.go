package parse

import (
	"log"
	"regexp"
	"sort"
	"time"

	"fmt"
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
	if err != nil {
		log.Printf("[ERROR] regexp '%v' could not be compiled: '%v'", regEx, err)
		return parser, err
	}
	err = parser.validate()
	return parser, err
}

// validate regex to make sure it contain right named fields
func (p *Parser) validate() (err error) {
	names := p.pattern.SubexpNames()
	for _, name := range [5]string{"SourceIP", "FileName", "DestinationNode", "AnswerTime", "Date"} {
		i := sort.SearchStrings(names, name)
		if i == len(names) {
			log.Printf("[ERROR] '%v' field absent in regexp", name)
			err = fmt.Errorf("'%v' missing regexp fields", p.pattern)
		}
	}
	return err
}

// Do parse log line into LogEntry
func (p *Parser) Do(line string) (entry LogEntry, err error) {
	result := p.pattern.FindStringSubmatch(line)
	n := p.pattern.SubexpNames()
	for i, m := range result {
		switch n[i] {
		case "": // first result is full string which is unnamed, do nothing with it
		case "SourceIP":
			entry.SourceIP = m
		case "FileName":
			entry.FileName = m
		case "DestinationNode":
			entry.DestinationNode = m
		case "AnswerTime":
			entry.AnswerTime, err = time.ParseDuration(m)
		case "Date":
			entry.Date, err = time.Parse(`2006/01/02 15:04:05`, m)
		}
		if err != nil {
			return entry, err
		}
	}
	// FIXME what to return in case of error? Should we return entry, or only error?
	return entry, err
}
