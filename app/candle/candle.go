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

// Aggregate candles from input, aggInterval truncated to minutes
func Aggregate(candles []Candle, aggInterval time.Duration) (result []Candle) {

	aggInterval = aggInterval.Truncate(time.Minute)
	var firstDate, lastDate = time.Now(), time.Time{}
	for _, c := range candles {
		if c.StartMinute.Before(firstDate) {
			firstDate = c.StartMinute
		}
		if c.StartMinute.After(lastDate) {
			lastDate = c.StartMinute
		}
	}

	for aggTime := firstDate; aggTime.Before(lastDate.Add(aggInterval)); aggTime = aggTime.Add(aggInterval) {
		minuteCandle := NewCandle()
		minuteCandle.StartMinute = aggTime
		for _, c := range candles {
			if c.StartMinute == aggTime || c.StartMinute.After(aggTime) && c.StartMinute.Before(aggTime.Add(aggInterval)) {
				c = updateAndDiscardTime(minuteCandle, c)
			}
		}
		if len(minuteCandle.Nodes) != 0 {
			result = append(result, minuteCandle)
		}
	}
	return result
}

func updateAndDiscardTime(source Candle, appendix Candle) Candle {
	for n := range appendix.Nodes {
		m, ok := source.Nodes[n]
		if !ok {
			m = NewInfo()
		}
		// to calculate mean time multiply source and appendix by their volume and divide everything by total volume
		m.MeanAnswerTime = (m.MeanAnswerTime*time.Duration(m.Volume) + appendix.Nodes[n].MeanAnswerTime*time.Duration(appendix.Nodes[n].Volume)) /
			time.Duration(m.Volume+appendix.Nodes[n].Volume)
		if m.MinAnswerTime > appendix.Nodes[n].MinAnswerTime {
			m.MinAnswerTime = appendix.Nodes[n].MinAnswerTime
		}
		if m.MaxAnswerTime < appendix.Nodes[n].MaxAnswerTime {
			m.MaxAnswerTime = appendix.Nodes[n].MaxAnswerTime
		}
		for file := range appendix.Nodes[n].Files {
			m.Files[file] += appendix.Nodes[n].Files[file]
		}
		m.Volume += appendix.Nodes[n].Volume
		source.Nodes[n] = m
	}
	return source
}
