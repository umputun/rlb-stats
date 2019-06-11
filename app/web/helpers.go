package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/wcharczuk/go-chart"

	"github.com/umputun/rlb-stats/app/store"
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

// loadCandles loads candles for given period of time aggregated by given duration
func loadCandles(from time.Time, to time.Time, duration time.Duration) ([]store.Candle, error) {
	var result []store.Candle
	candleGetURL := fmt.Sprintf("%v/api/candle?from=%v&to=%v&aggregate=%v",
		apiClient.apiURL,
		url.QueryEscape(from.Format(time.RFC3339)),
		url.QueryEscape(to.Format(time.RFC3339)),
		duration)
	r, err := apiClient.httpClient.Get(candleGetURL)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, r.Body.Close()
}

// prepareSeries require candles and request duration\step data and returns
// a chart.Series from given candles with given params
func prepareSeries(candles []store.Candle, qType string, filterFilename string) (series []chart.Series) {
	tempSeries := map[string]chart.TimeSeries{}
	for _, candle := range candles {
		switch qType {
		case "by_file":
			for filename, count := range candle.Nodes["all"].Files {
				if filename == "all" {
					continue
				}
				if filterFilename != "" && filename != filterFilename {
					continue
				}
				tempSeries[filename] = chart.TimeSeries{Name: filename,
					XValues: append(tempSeries[filename].XValues, candle.StartMinute),
					YValues: append(tempSeries[filename].YValues, float64(count)),
				}
			}
		case "by_node":
			for node, count := range candle.Nodes {
				if node == "all" {
					continue
				}
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

// valueFormatter formats float values without decimal part as integers,
// and don't return anything for any other input
func valueFormatter(v interface{}) string {
	switch v := v.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case float64:
		// print number if only have no decimal part
		if v == float64(int(v)) {
			return fmt.Sprintf("%.0f", v)
		}
	}
	return ""
}
