package store

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
	"github.com/umputun/rlb-stats/app/parse"
)

var bucketCandles = []byte("stats")
var bucketLogs = []byte("raw_logs")

// Bolt implements store.Engine with boltdb
type Bolt struct {
	db *bolt.DB
}

// NewBolt makes persistent boltdb based store
func NewBolt(dbFile string, collectDuration time.Duration) (*Bolt, error) {
	log.Printf("[INFO] bolt (persitent) store, %s", dbFile)
	result := Bolt{}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bucketCandles)
		return e
	})
	db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bucketLogs)
		return e
	})
	result.db = db
	result.activateCollector(collectDuration)
	return &result, err
}

// Save with ts-ip as a key. ts prefix for bolt range query
func (s *Bolt) Save(entry *parse.LogEntry) (err error) {
	key := fmt.Sprintf("%d-%s", entry.Date.Unix(), entry.SourceIP)
	total := 0
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketLogs)
		total = b.Stats().KeyN
		jdata, jerr := json.Marshal(entry)
		if jerr != nil {
			return err
		}
		return b.Put([]byte(key), jdata)
	})

	log.Printf("[DEBUG] saved, time=%v, total=%d", entry.Date, total+1)
	return err
}

// Load by period
func (s *Bolt) loadLogEntry(periodStart, periodEnd time.Time) (result []parse.LogEntry, err error) {
	// TODO: collect data for period, return entries
	result = append(result, parse.LogEntry{})
	return
}

// Load by period
func (s *Bolt) Load(periodStart, periodEnd time.Time) (result []Candle, err error) {
	// TODO: collect data for period, return candles
	return
}

// TODO: write logEntry->candle function
// TODO: dedupe same ips in candle (by which rule?)

// activateCollector runs periodic cleanups to aggregate data into candles
// detection based on ts (unix time) prefix of the key.
func (s *Bolt) activateCollector(every time.Duration) {
	log.Printf("[INFO] collecter activated, every %v", every)

	ticker := time.NewTicker(every)
	go func() {
		for range ticker.C {

			// TODO: collect data for previous period and group into candles

		}
	}()
}
