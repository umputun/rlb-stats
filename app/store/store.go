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
	Save(candle Candle) (err error)
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
	if n.MinAnswerTime > l.AnswerTime {
		n.MinAnswerTime = l.AnswerTime
	}
	n.MeanAnswerTime = (n.MeanAnswerTime*time.Duration(n.Volume) + l.AnswerTime) / time.Duration(n.Volume+1)
	if n.MaxAnswerTime < l.AnswerTime {
		n.MaxAnswerTime = l.AnswerTime
	}
	n.Files[l.FileName]++
	n.Volume++
}

func newCandle() (c Candle) {
	c.Nodes = map[string]Info{}
	c.StartMinute = time.Time{}
	return
}

func (c *Candle) update(l parse.LogEntry) {
	node, ok := c.Nodes[l.DestinationNode]
	if !ok {
		node = newInfo()
	}
	node.update(l)
	c.Nodes[l.DestinationNode] = node
	nodeAll, allOk := c.Nodes["all"]
	if !allOk {
		nodeAll = newInfo()
	}
	nodeAll.update(l)
	c.Nodes["all"] = nodeAll
	c.StartMinute = l.Date
}
