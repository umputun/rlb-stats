package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func info(volume int, files map[string]int, bytes int64, countries map[string]int) Info {
	if files == nil {
		files = map[string]int{}
	}
	if countries == nil {
		countries = map[string]int{}
	}
	return Info{Volume: volume, Files: files, Bytes: bytes, Countries: countries}
}

var logsTestsTable = []struct {
	in  LogRecord
	out Candle
}{
	{
		LogRecord{
			FromIP:   "127.0.0.1",
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n6.radio-t.com",
			Date:     time.Time{},
			Bytes:    1000,
			Country:  "US",
		},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": info(1, nil, 1000, nil),
				"all":            info(1, map[string]int{"/rtfiles/rt_podcast561.mp3": 1}, 1000, map[string]int{"US": 1}),
			},
			StartMinute: time.Time{},
		},
	},
	{
		LogRecord{
			FromIP:   "127.0.0.3",
			FileName: "/rtfiles/rt_podcast562.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{},
			Bytes:    500,
			Country:  "DE",
		},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": info(1, nil, 1000, nil),
				"n7.radio-t.com": info(1, nil, 500, nil),
				"all": info(2,
					map[string]int{"/rtfiles/rt_podcast561.mp3": 1, "/rtfiles/rt_podcast562.mp3": 1},
					1500,
					map[string]int{"US": 1, "DE": 1}),
			},
			StartMinute: time.Time{},
		},
	},
	{
		LogRecord{
			FromIP:   "127.0.0.2",
			FileName: "/rtfiles/rt_podcast561.mp3",
			DestHost: "n7.radio-t.com",
			Date:     time.Time{},
			Bytes:    2000,
			// Country empty: non-CF or geolocation disabled
		},
		Candle{
			Nodes: map[string]Info{
				"n6.radio-t.com": info(1, nil, 1000, nil),
				"n7.radio-t.com": info(2, nil, 2500, nil),
				"all": info(3,
					map[string]int{"/rtfiles/rt_podcast561.mp3": 2, "/rtfiles/rt_podcast562.mp3": 1},
					3500,
					map[string]int{"US": 1, "DE": 1}),
			},
			StartMinute: time.Time{},
		},
	},
}

func TestNewAndUpdateCandle(t *testing.T) {
	candle := NewCandle()
	for _, testPair := range logsTestsTable {
		candle.Update(testPair.in)
		assert.EqualValues(t, testPair.out, candle, "candle match with expected output")
	}
}

func TestAddBytes(t *testing.T) {
	candle := NewCandle()
	rec := LogRecord{FromIP: "10.0.0.1", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 100, Country: "FR"}
	candle.Update(rec)

	// dedup path: AddBytes must bump bytes on both nodes without touching Volume/Files/Countries
	candle.AddBytes(LogRecord{FromIP: "10.0.0.1", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 400, Country: "FR"})
	candle.AddBytes(LogRecord{FromIP: "10.0.0.1", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 300, Country: "FR"})

	assert.Equal(t, 1, candle.Nodes["cdn.x"].Volume, "Volume unchanged by AddBytes")
	assert.Equal(t, 1, candle.Nodes["all"].Volume, "AllNode Volume unchanged by AddBytes")
	assert.Equal(t, int64(800), candle.Nodes["cdn.x"].Bytes, "Bytes summed on dest node")
	assert.Equal(t, int64(800), candle.Nodes["all"].Bytes, "Bytes summed on AllNode")
	assert.Equal(t, 1, candle.Nodes["all"].Files["/a.mp3"], "Files unchanged by AddBytes")
	assert.Equal(t, 1, candle.Nodes["all"].Countries["FR"], "Countries unchanged by AddBytes")
}

func TestAddBytesZeroIsNoop(t *testing.T) {
	// a HEAD request that squeaks through filters would land with Bytes=0; AddBytes must
	// not create empty node entries just to add zero bytes.
	candle := NewCandle()
	candle.AddBytes(LogRecord{FromIP: "10.0.0.1", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 0})
	assert.Empty(t, candle.Nodes, "zero bytes must not create node entries")
}

func TestUpdateEmptyCountryIgnored(t *testing.T) {
	// non-CF deployment: Country stays empty, Countries map must not accumulate an "" key
	candle := NewCandle()
	candle.Update(LogRecord{FromIP: "10.0.0.1", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 10})
	assert.Empty(t, candle.Nodes["all"].Countries, "empty country must not land in map")
}

func TestMergeCandlesSumsBytesAndCountries(t *testing.T) {
	a := NewCandle()
	a.StartMinute = time.Time{}.Add(time.Minute)
	a.Update(LogRecord{FromIP: "1.1.1.1", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 100, Country: "US"})

	b := NewCandle()
	b.StartMinute = time.Time{}.Add(time.Minute)
	b.Update(LogRecord{FromIP: "2.2.2.2", FileName: "/a.mp3", DestHost: "cdn.x", Bytes: 250, Country: "DE"})
	b.Update(LogRecord{FromIP: "3.3.3.3", FileName: "/b.mp3", DestHost: "cdn.x", Bytes: 50, Country: "US"})

	merged := mergeCandles(a, b)

	assert.Equal(t, int64(400), merged.Nodes["cdn.x"].Bytes)
	assert.Equal(t, int64(400), merged.Nodes["all"].Bytes)
	assert.Equal(t, 3, merged.Nodes["all"].Volume)
	assert.Equal(t, 2, merged.Nodes["all"].Countries["US"])
	assert.Equal(t, 1, merged.Nodes["all"].Countries["DE"])
	assert.Equal(t, 2, merged.Nodes["all"].Files["/a.mp3"])
	assert.Equal(t, 1, merged.Nodes["all"].Files["/b.mp3"])
}

func TestInfoBackwardCompat(t *testing.T) {
	// old boltdb candles (JSON written before Bytes/Countries existed) must unmarshal into
	// the new Info shape with zero/nil for the new fields and no error.
	oldJSON := `{"Nodes":{"all":{"Volume":5,"Files":{"/a.mp3":5}}},"StartMinute":"2024-01-01T00:00:00Z"}`
	var c Candle
	require.NoError(t, json.Unmarshal([]byte(oldJSON), &c))
	assert.Equal(t, 5, c.Nodes["all"].Volume)
	assert.Equal(t, 5, c.Nodes["all"].Files["/a.mp3"])
	assert.Equal(t, int64(0), c.Nodes["all"].Bytes)
	assert.Nil(t, c.Nodes["all"].Countries, "missing field unmarshals as nil map")
}
