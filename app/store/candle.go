package store

import (
	"time"
)

// Candle contain one minute candle from log entries for that period
type Candle struct {
	Nodes       map[string]Info
	StartMinute time.Time
}

// Info contain single node download statistics
type Info struct {
	Volume int
	Files  map[string]int
}

// NewInfo create empty node information
func NewInfo() Info {
	return Info{
		Volume: 0,
		Files:  map[string]int{},
	}
}

// LogRecord contains meaningful subset of data from rlb LogRecord
type LogRecord struct {
	FromIP   string    `json:"from_ip"`
	FileName string    `json:"file_name"`
	DestHost string    `json:"dest"`
	Date     time.Time `json:"ts"`
}

// NewCandle create empty candle
func NewCandle() (c Candle) {
	c.Nodes = map[string]Info{}
	c.StartMinute = time.Time{}
	return c
}

// Update log destination node and add same stats to "all" node
func (c *Candle) Update(l LogRecord) {
	for _, nodeName := range []string{l.DestHost, "all"} {
		node, ok := c.Nodes[nodeName]
		if !ok {
			node = NewInfo()
		}
		if nodeName == "all" { // we keep all files in "all" node only
			node.Files[l.FileName]++
		}
		node.Volume++
		c.Nodes[nodeName] = node
	}
	c.StartMinute = l.Date
}
