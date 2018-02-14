package candle

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

// NewInfo create empty node information
func NewInfo() Info {
	return Info{
		Volume:         0,
		MinAnswerTime:  time.Hour,
		MeanAnswerTime: time.Duration(0),
		MaxAnswerTime:  time.Duration(0),
		Files:          map[string]int{},
	}
}

// update single node information
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

// NewCandle create empty candle
func NewCandle() (c Candle) {
	c.Nodes = map[string]Info{}
	c.StartMinute = time.Time{}
	return c
}

// Update log destination node and add same stats to "all" node
func (c *Candle) Update(l parse.LogEntry) {
	for _, nodeName := range []string{l.DestinationNode, "all"} {
		node, ok := c.Nodes[nodeName]
		if !ok {
			node = NewInfo()
		}
		node.update(l)
		c.Nodes[nodeName] = node
	}
	c.StartMinute = l.Date
}
