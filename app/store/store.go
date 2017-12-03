package store

import (
	"time"

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
	Save(entries map[time.Time]Candle) (err error)
	Load(periodStart, periodEnd time.Time) (result []Candle, err error)
}

func newInfo() Info {
	return Info{
		Volume:         0,
		MinAnswerTime:  time.Hour,
		MeanAnswerTime: time.Duration(0),
		MaxAnswerTime:  time.Duration(0),
		Files:          map[string]int{},
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

func newCandle(StartMinute time.Time) (c Candle) {
	c.Nodes = map[string]Info{}
	c.StartMinute = StartMinute
	return
}

func (c *Candle) update(l parse.LogEntry) {
	node, ok := c.Nodes[l.DestinationNode]
	if !ok {
		node = newInfo()
	}
	node.update(l)
	c.Nodes[l.DestinationNode] = node
	nodeAll := c.Nodes["all"]
	nodeAll.update(l)
	c.Nodes["all"] = nodeAll
}
