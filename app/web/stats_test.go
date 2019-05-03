package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/store"
)

var results = map[int]struct{ topFiles, topNodes []volumeStats }{
	1: {topFiles: []volumeStats{{"/rtfiles/rt_podcast561.mp3", 4}},
		topNodes: []volumeStats{{"n6.radio-t.com", 3}}},
	2: {topFiles: []volumeStats{{"/rtfiles/rt_podcast561.mp3", 4}, {"/rtfiles/rt_podcast562.mp3", 2}},
		topNodes: []volumeStats{{"n6.radio-t.com", 3}, {"n7.radio-t.com", 2}}},
	3: {topFiles: []volumeStats{{"/rtfiles/rt_podcast561.mp3", 4}, {"/rtfiles/rt_podcast562.mp3", 2}},
		topNodes: []volumeStats{{"n6.radio-t.com", 3}, {"n7.radio-t.com", 2}, {"n8.radio-t.com", 1}}},
}

func TestStats(t *testing.T) {

	candles := []store.Candle{
		{Nodes: map[string]store.Info{
			"n6.radio-t.com": {Volume: 2},
			"all":            {Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1}},
		}},
		{Nodes: map[string]store.Info{
			"n8.radio-t.com": {Volume: 1},
			"n7.radio-t.com": {Volume: 2},
			"n6.radio-t.com": {Volume: 1},
			"all":            {Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 3, "/rtfiles/rt_podcast562.mp3": 1}},
		}},
	}

	for _, i := range []int{1, 2, 3} {
		assert.EqualValues(t, results[i].topFiles, getTop("files", candles, i), "candle match with expected output")
		assert.EqualValues(t, results[i].topNodes, getTop("nodes", candles, i), "candle match with expected output")
	}
	//	big number request should return all results
	assert.EqualValues(t, results[3].topFiles, getTop("files", candles, 100), "candle match with expected output")
	assert.EqualValues(t, results[3].topNodes, getTop("nodes", candles, 100), "candle match with expected output")

}
