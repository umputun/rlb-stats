package store

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"
)

// Parser contain validated regular expression for parsing logs
type Parser struct {
	pattern    *regexp.Regexp
	dateFormat string
	entries    []LogRecord // used to store entries which are not yet dumped into candles
}

// NewLogService checks if regular expression valid
func newParser(regEx string, dateFormat string) (parser *Parser, err error) {
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
	for _, name := range []string{"FromIP", "FileName", "DestHost", "Date"} {
		if !contains(name, names) {
			log.Printf("[ERROR] '%v' field absent in regexp", name)
			err = fmt.Errorf("'%v' missing regexp fields", p.pattern)
		}
	}
	return err
}

// Do parse log line into LogRecord
func (p *Parser) Do(line string) (entry LogRecord, err error) {
	result := p.pattern.FindStringSubmatch(line)
	if result == nil {
		return entry, errors.New("can't match line against given regEx")
	}
	n := p.pattern.SubexpNames()
	for i, m := range result {
		switch n[i] {
		case "": // first result is full string which is unnamed, do nothing with it
		case "FromIP":
			entry.FromIP = m
		case "FileName":
			entry.FileName = m
		case "DestHost":
			entry.DestHost = m
		case "Date":
			entry.Date, err = time.ParseInLocation(p.dateFormat, m, time.Local)
		}
		if err != nil {
			return entry, err
		}
	}
	return entry, err
}

// Submit store LogRecord and return Candle when minute change
func (p *Parser) Submit(newEntry LogRecord) (Candle, bool) {
	minuteCandle := Candle{}
	ok := false
	// drop seconds and nanoseconds from log date
	newEntry.Date = time.Date(
		newEntry.Date.Year(),
		newEntry.Date.Month(),
		newEntry.Date.Day(),
		newEntry.Date.Hour(),
		newEntry.Date.Minute(),
		0,
		0,
		newEntry.Date.Location())

	if len(p.entries) > 0 && !newEntry.Date.Equal(p.entries[len(p.entries)-1].Date) { // if there are existing entries and date changed
		minuteCandle = NewCandle()                  // then all previous entries have same date precise to the minute and will be written to single candle
		var deduplicate = make(map[string]struct{}) // deduplicate store ip-file map
		for _, entry := range p.entries {
			_, duplicate := deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.FromIP)]
			if !duplicate {
				minuteCandle.Update(entry)
				deduplicate[fmt.Sprintf("%s-%s", entry.FileName, entry.FromIP)] = struct{}{}
			}
		}
		ok = true                 // candle is ready to be written
		p.entries = []LogRecord{} // clean written entries
	}
	p.entries = append(p.entries, newEntry)

	return minuteCandle, ok
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
