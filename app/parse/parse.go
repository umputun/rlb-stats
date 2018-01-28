package parse

import (
	"log"
	"regexp"
	"time"

	"fmt"
)

// Parser contain validated regular expression for parsing logs
type Parser struct {
	pattern    *regexp.Regexp
	dateFormat string
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
func New(regEx string, dateFormat string) (parser *Parser, err error) {
	parser = &Parser{}
	parser.dateFormat = dateFormat
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
	for _, name := range []string{"SourceIP", "FileName", "DestinationNode", "AnswerTime", "Date"} {
		if !contains(name, names) {
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
			entry.Date, err = time.Parse(p.dateFormat, m)
		}
		if err != nil {
			return entry, err
		}
	}
	return entry, err
}

// contains string in slice
func contains(src string, inSlice []string) bool {
	for _, a := range inSlice {
		if a == src {
			return true
		}
	}
	return false
}
