package store

import (
	"time"

	"fmt"

	"github.com/umputun/rlb-stats/app/parse"
)

// Candle contain one minute candle from log entries for that period
type Candle struct {
	Nodes       map[string]Info
	StartMinute time.Time
}

// Info contain single node download statistics
type Info struct {
	Volume         int
	MinAnswerTime  time.Duration
	MeanAnswerTime time.Duration
	MaxAnswerTime  time.Duration
	Files          map[string]int
}

// Engine defines interface to save log entries and load candles
type Engine interface {
	Save(msg parse.LogEntry) (err error)
	Load(periodStart, periodEnd time.Time) (result []Candle, err error)
}

func newInfo(l parse.LogEntry) Info {
	return Info{
		Volume:         1,
		MinAnswerTime:  l.AnswerTime,
		MeanAnswerTime: l.AnswerTime,
		MaxAnswerTime:  l.AnswerTime,
		Files:          map[string]int{l.FileName: 1},
	}
}

func (n *Info) update(l parse.LogEntry) {
	// int is 0 if not defined, OK to use it
	n.Files[l.FileName] += 1
	if n.MinAnswerTime > l.AnswerTime {
		n.MinAnswerTime = l.AnswerTime
	}
	if n.MaxAnswerTime < l.AnswerTime {
		n.MaxAnswerTime = l.AnswerTime
	}
	n.MeanAnswerTime = (n.MeanAnswerTime*time.Duration(n.Volume) + l.AnswerTime) / time.Duration(n.Volume+1)
	n.Volume += 1
}

func newCandle(l parse.LogEntry) (c Candle) {
	node := newInfo(l)
	c.Nodes = map[string]Info{
		l.DestinationNode: node,
		"all":             node,
	}
	c.StartMinute = l.Date
	return
}

func (c *Candle) update(l parse.LogEntry) {
	node, nodeInCandle := c.Nodes[l.DestinationNode]
	if nodeInCandle {
		node.update(l)
	} else {
		node = newInfo(l)
	}
	c.Nodes[l.DestinationNode] = node
	nodeAll := c.Nodes["all"]
	nodeAll.update(l)
	c.Nodes["all"] = nodeAll
}

// entriesToCandles convert LogEntries to candles,
// dropping duplicate IP-filename pairs each minute
func entriesToCandles(entries []parse.LogEntry) map[time.Time]Candle {
	c := make(map[time.Time]Candle)
	deduplicate := make(map[string]bool)
	for _, entry := range entries {
		// drop seconds and nanoseconds from log date
		entry.Date = time.Date(
			entry.Date.Year(),
			entry.Date.Month(),
			entry.Date.Day(),
			entry.Date.Hour(),
			entry.Date.Minute(),
			0,
			0,
			entry.Date.Location())
		_, duplicate := deduplicate[fmt.Sprintf("%d-%s-%s", entry.Date.Unix(), entry.FileName, entry.SourceIP)]
		if !duplicate {
			candle, exists := c[entry.Date]
			if exists {
				candle.update(entry)
				c[entry.Date] = candle
			} else {
				c[entry.Date] = newCandle(entry)
			}
			deduplicate[fmt.Sprintf("%d-%s-%s", entry.Date.Unix(), entry.FileName, entry.SourceIP)] = true
		}
	}
	return c
}
