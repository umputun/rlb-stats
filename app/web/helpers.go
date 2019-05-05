package web

import (
	"log"
	"time"

	"github.com/umputun/rlb-stats/app/store"
	"github.com/wcharczuk/go-chart"
)

// calculateTimePeriod waits for from and to time.Duration in human-readable form
// and returns time.Time in relevance with current time, and 1\10th of this period
// as time.Duration.
func calculateTimePeriod(from, to string) (time.Time, time.Time, time.Duration) {
	if from == "" {
		from = "168h"
	}
	fromDuration, err := time.ParseDuration(from)
	if err != nil {
		// TODO write a warning about being unable to parse from field
		// TODO handle negative duration
		log.Print("[WARN] dashboard: can't parse from field")
		fromDuration = time.Hour * 24 * 7
	}
	fromTime := time.Now().Add(-fromDuration)
	toTime := time.Now()
	if to != "" {
		t, terr := time.ParseDuration(to)
		if terr != nil {
			log.Print("[WARN] dashboard: can't parse to field")
			//	TODO write a warning about being unable to parse to field
			//	TODO handle negative duration
		}
		toTime = toTime.Add(-t)
	}
	return fromTime, toTime, toTime.Sub(fromTime).Truncate(time.Second) / 10
}

// prepareSeries require candles and request duration\step data and returns
// a chart.Series from given candles with given params
func prepareSeries(candles []store.Candle, fromTime time.Time, toTime time.Time, aggDuration time.Duration, qType string) (series []chart.Series) {
	tempSeries := map[string]chart.TimeSeries{}
	for _, candle := range candles {
		switch qType {
		case "by_file":
			for filename, count := range candle.Nodes["all"].Files {
				tempSeries[filename] = chart.TimeSeries{Name: filename,
					XValues: append(tempSeries[filename].XValues, candle.StartMinute),
					YValues: append(tempSeries[filename].YValues, float64(count)),
				}
				delete(tempSeries, "all")
			}
		default:
			for node, count := range candle.Nodes {
				tempSeries[node] = chart.TimeSeries{Name: node,
					XValues: append(tempSeries[node].XValues, candle.StartMinute),
					YValues: append(tempSeries[node].YValues, float64(count.Volume)),
				}
			}
		}
	}
	for _, c := range tempSeries {
		series = append(series, c)
	}
	return series
}
