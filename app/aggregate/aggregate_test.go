package aggregate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
)

var testCandles = []candle.Candle{
	{Nodes: map[string]candle.Info{
		"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 6, MeanAnswerTime: time.Second * 6, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 6, MeanAnswerTime: time.Second * 6, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}},
	{Nodes: map[string]candle.Info{
		"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute)},
	{Nodes: map[string]candle.Info{
		"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 2)},
	{Nodes: map[string]candle.Info{
		"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 3)},
	{Nodes: map[string]candle.Info{
		"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 4)},
	{Nodes: map[string]candle.Info{
		"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 5)},
	{Nodes: map[string]candle.Info{
		"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		StartMinute: time.Time{}.Add(time.Minute * 10)},
}

var resultCandles = map[int][]candle.Candle{
	1: testCandles,
	2: {
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3 + 6) / 2, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3 + 6) / 2, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
		},
			StartMinute: time.Time{}},

		{Nodes: map[string]candle.Info{
			"n7.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 2)},

		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 4)},

		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 10)},
	},
	3: {
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3 + 6) / 2, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {Volume: 3, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3 + 6 + 3) / 3, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
		},
			StartMinute: time.Time{}},
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"n7.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {Volume: 3, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 3)},
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 9)},
	},
	5: {
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 3, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3*2 + 6) / 3, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 3}},
			"n7.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {Volume: 5, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3*2 + 6 + 3*2) / 5, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 3, "/rtfiles/rt_podcast562.mp3": 2}},
		},
			StartMinute: time.Time{}},
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 5)},
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 10)}},
	10: {
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 4, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3*3 + 6) / 4, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 4}},
			"n7.radio-t.com": {Volume: 2, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {Volume: 6, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * (3*3 + 6 + 3*2) / 6, MaxAnswerTime: time.Second * 6, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 4, "/rtfiles/rt_podcast562.mp3": 2}},
		},
			StartMinute: time.Time{}},
		{Nodes: map[string]candle.Info{
			"n6.radio-t.com": {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {Volume: 1, MinAnswerTime: time.Second * 3, MeanAnswerTime: time.Second * 3, MaxAnswerTime: time.Second * 3, Files: map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			StartMinute: time.Time{}.Add(time.Minute * 10)}},
}

func TestAggregation(t *testing.T) {

	for _, i := range []int{1, 2, 3, 5, 10} {
		testSlice := Do(testCandles, time.Duration(i)*time.Minute)
		assert.EqualValues(t, resultCandles[i], testSlice, "candle aggregate for %v minutes match with expected output", i)
	}
}
