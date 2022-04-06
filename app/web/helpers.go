package web

import (
	"sort"
	"time"

	"github.com/umputun/rlb-stats/app/store"
)

// loadCandles loads candles for given period of time aggregated by given duration
func loadCandles(engine store.Engine, from time.Time, to time.Time, aggDuration time.Duration) ([]store.Candle, error) {
	candles, err := engine.Load(from, to)
	if err != nil {
		return nil, err
	}
	if aggDuration != time.Minute {
		candles = aggregateCandles(candles, aggDuration)
	}
	return candles, nil
}

// saveLogRecord saves a log record to candle
func saveLogRecord(engine store.Engine, parser *store.Aggregator, l store.LogRecord) error {
	if candle, ok := parser.Store(l); ok { // Store returns ok in case candle is ready
		return engine.Save(candle)
	}
	return nil
}

// limitCandleFiles limit files in each node and keep only top N files
func limitCandleFiles(candles []store.Candle, filesLimit int) []store.Candle {

	type fileInfo struct {
		name  string
		count int
	}

	flatFiles := func(files map[string]int) []fileInfo {
		var res []fileInfo
		for k, v := range files {
			res = append(res, fileInfo{k, v})
		}
		return res
	}

	mapFiles := func(files []fileInfo) map[string]int {
		res := make(map[string]int)
		for _, v := range files {
			res[v.name] = v.count
		}
		return res
	}

	res := []store.Candle{}
	for _, c := range candles {

		candle := store.Candle{
			Nodes:       make(map[string]store.Info),
			StartMinute: c.StartMinute,
		}

		for name, node := range c.Nodes {
			files := flatFiles(node.Files)
			sort.Slice(files, func(i, j int) bool {
				return files[i].count > files[j].count
			})
			if len(files) > filesLimit {
				files = files[:filesLimit]
			}
			candle.Nodes[name] = store.Info{
				Volume: node.Volume,
				Files:  mapFiles(files),
			}
		}
		res = append(res, candle)
	}
	return res
}
