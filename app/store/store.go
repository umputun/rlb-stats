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

// Engine defines interface to save log entries and load candels
type Engine interface {
	Save(msg parse.LogEntry) (err error)
	Load(periodStart, periodEnd time.Time) (result []Candle, err error)
}

func infoFromLog(l parse.LogEntry) Info {
	return Info{
		Volume:         1,
		MinAnswerTime:  l.AnswerTime,
		MeanAnswerTime: l.AnswerTime,
		MaxAnswerTime:  l.AnswerTime,
		Files:          map[string]int{l.FileName: 1},
	}
}

func (n Info) appendLog(l parse.LogEntry) Info {
	count, fileInNode := n.Files[l.FileName]
	if fileInNode {
		n.Files[l.FileName] = count + 1
	} else {
		n.Files[l.FileName] = 1
	}
	if n.MinAnswerTime > l.AnswerTime {
		n.MinAnswerTime = l.AnswerTime
	}
	if n.MaxAnswerTime < l.AnswerTime {
		n.MaxAnswerTime = l.AnswerTime
	}
	// TODO get average of durations
	//n.MeanAnswerTime = (n.MeanAnswerTime * n.Volume) + n.MeanAnswerTime) / (n.Volume + 1)
	n.Volume++
	return n
}

func createCandle(l parse.LogEntry) (c Candle) {
	node := infoFromLog(l)
	c.Nodes = map[string]Info{
		l.DestinationNode: node,
		"all":             node,
	}
	return
}

func appendToCandle(c Candle, l parse.LogEntry) Candle {
	node, nodeInCandle := c.Nodes[l.DestinationNode]
	if nodeInCandle {
		node = node.appendLog(l)
	} else {
		node = infoFromLog(l)
	}
	c.Nodes[l.DestinationNode] = node
	c.Nodes["all"] = c.Nodes["all"].appendLog(l)
	return c
}
