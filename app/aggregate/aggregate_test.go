package aggregate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb-stats/app/candle"
)

var testCandles = []candle.Candle{
	{map[string]candle.Info{
		"n6.radio-t.com": {1, time.Second * 6, time.Second * 6, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {1, time.Second * 6, time.Second * 6, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		time.Time{}},
	{map[string]candle.Info{
		"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		time.Time{}.Add(time.Minute)},
	{map[string]candle.Info{
		"n7.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
	},
		time.Time{}.Add(time.Minute * 2)},
	{map[string]candle.Info{
		"n7.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
	},
		time.Time{}.Add(time.Minute * 3)},
	{map[string]candle.Info{
		"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		time.Time{}.Add(time.Minute * 4)},
	{map[string]candle.Info{
		"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		time.Time{}.Add(time.Minute * 5)},
	{map[string]candle.Info{
		"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
	},
		time.Time{}.Add(time.Minute * 10)},
}

var resultCandles = map[int][]candle.Candle{
	1: append([]candle.Candle(nil), testCandles...),
	2: {
		{map[string]candle.Info{
			"n6.radio-t.com": {2, time.Second * 3, time.Second * (3 + 6) / 2, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {2, time.Second * 3, time.Second * (3 + 6) / 2, time.Second * 5, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
		},
			time.Time{}},

		{map[string]candle.Info{
			"n7.radio-t.com": {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
		},
			time.Time{}.Add(time.Minute * 2)},

		{map[string]candle.Info{
			"n6.radio-t.com": {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
		},
			time.Time{}.Add(time.Minute * 4)},

		{map[string]candle.Info{
			"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			time.Time{}.Add(time.Minute * 10)},
	},
	3: {
		{map[string]candle.Info{
			"n6.radio-t.com": {2, time.Second * 3, time.Second * (3 + 6) / 2, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"n7.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {3, time.Second * 3, time.Second * (3 + 6 + 3) / 3, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
		},
			time.Time{}},
		{map[string]candle.Info{
			"n6.radio-t.com": {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"n7.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 1}},
			"all":            {3, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1}},
		},
			time.Time{}.Add(time.Minute * 3)},
		{map[string]candle.Info{
			"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			time.Time{}.Add(time.Minute * 9)},
	},
	5: {
		{map[string]candle.Info{
			"n6.radio-t.com": {3, time.Second * 3, time.Second * (3*2 + 6) / 3, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 3}},
			"n7.radio-t.com": {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 2}},
			"all":            {5, time.Second * 3, time.Second * (3*2 + 6 + 3*2) / 5, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 3, "/rtfiles/rt_podcast562.mp3": 2}},
		},
			time.Time{}},
		{map[string]candle.Info{
			"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			time.Time{}.Add(time.Minute * 5)},
		{map[string]candle.Info{
			"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			time.Time{}.Add(time.Minute * 10)}},
	10: {
		{map[string]candle.Info{
			"n6.radio-t.com": {4, time.Second * 3, time.Second * (3*3 + 6) / 4, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 4}},
			"n7.radio-t.com": {2, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast562.mp3": 2}},
			"all":            {6, time.Second * 3, time.Second * (3*3 + 6 + 3*2) / 6, time.Second * 6, map[string]int{"/rtfiles/rt_podcast561.mp3": 4, "/rtfiles/rt_podcast562.mp3": 2}},
		},
			time.Time{}},
		{map[string]candle.Info{
			"n6.radio-t.com": {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
			"all":            {1, time.Second * 3, time.Second * 3, time.Second * 3, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}},
		},
			time.Time{}.Add(time.Minute * 10)}},
}

func TestAggregation(t *testing.T) {

	for _, i := range []int{1, 2, 3, 5, 10} {
		testCandles := append([]candle.Candle(nil), testCandles...)
		Do(&testCandles, time.Duration(i)*time.Minute)
		assert.EqualValues(t, resultCandles[i], testCandles, "candle match with expected output")
	}
}
