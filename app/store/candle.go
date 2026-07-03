package store

import (
	"time"
)

// AllNode is the synthetic node name aggregating stats across every destination node
const AllNode = "all"

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

// mergeCandles combines the per-node volumes and file counts of base and add into a
// single candle, keeping add's StartMinute. used to accumulate multiple writes for the
// same minute instead of overwriting one with another.
//
// this runs only on the abnormal path where a minute is saved twice (flush-on-shutdown
// then restart, or a minute split across concurrent inserts); normal operation saves
// each minute once and never merges. counts are summed, so an ip+file pair that appears
// in both fragments is counted more than once - accepted as a best-effort recovery,
// since the alternative (overwrite) drops the earlier fragment entirely. exact
// cross-fragment de-duplication would require persisting the dedup keys.
func mergeCandles(base, add Candle) Candle {
	merged := NewCandle()
	merged.StartMinute = add.StartMinute
	for _, src := range []Candle{base, add} {
		for name, info := range src.Nodes {
			node, ok := merged.Nodes[name]
			if !ok {
				node = NewInfo()
			}
			node.Volume += info.Volume
			for file, count := range info.Files {
				node.Files[file] += count
			}
			merged.Nodes[name] = node
		}
	}
	return merged
}

// Update log destination node and add same stats to "all" node
func (c *Candle) Update(l LogRecord) {
	for _, nodeName := range []string{l.DestHost, AllNode} {
		node, ok := c.Nodes[nodeName]
		if !ok {
			node = NewInfo()
		}
		if nodeName == AllNode { // we keep all files in AllNode only
			node.Files[l.FileName]++
		}
		node.Volume++
		c.Nodes[nodeName] = node
	}
	c.StartMinute = l.Date
}
