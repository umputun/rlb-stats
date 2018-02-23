package web

import (
	"sort"

	"github.com/umputun/rlb-stats/app/store"
)

// fileInfo contain single node download statistics
type volumeStats struct {
	name   string
	volume int
}

func getTop(aggType string, candles []store.Candle, amount int) []volumeStats {
	agg := map[string]int{}
	for _, candle := range candles {
		switch aggType {
		case "files":
			{
				for filename, count := range candle.Nodes["all"].Files {
					agg[filename] += count
				}
			}
		case "nodes":
			{
				for node, count := range candle.Nodes {
					agg[node] += count.Volume
				}
			}
		}
	}

	var result []volumeStats

	for k, v := range agg {
		result = append(result, volumeStats{k, v})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].volume > result[j].volume
	})

	return result[:amount]
}
