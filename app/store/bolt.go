package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	bolt "go.etcd.io/bbolt"
)

var bucket = []byte("stats")

// Bolt implements store.Engine with boltdb
type Bolt struct {
	db *bolt.DB
}

// NewBolt makes persistent boltdb based store
func NewBolt(dbFile string) (*Bolt, error) {
	log.Printf("[INFO] bolt (persistent) store, %s", dbFile)
	result := Bolt{}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return &result, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bucket)
		return e
	})
	if err != nil {
		return &result, err
	}
	result.db = db
	return &result, err
}

// Save Candles with starting minute time.Unix() as a key for bolt range query
func (s *Bolt) Save(candle Candle) (err error) {
	key := fmt.Sprintf("%d", candle.StartMinute.Unix())
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		jdata, jerr := json.Marshal(candle)
		if jerr != nil {
			return jerr
		}
		return b.Put([]byte(key), jdata)
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] saved candle, StartMinute=%v", candle.StartMinute.Unix())
	return nil
}

// Load Candles by period
func (s *Bolt) Load(periodStart, periodEnd time.Time) (result []Candle, err error) {
	result = []Candle{}
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		c := b.Cursor()

		min := []byte(fmt.Sprintf("%d", periodStart.Unix()))
		max := []byte(fmt.Sprintf("%d", periodEnd.Unix()))

		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			newCandle := Candle{}
			err = json.Unmarshal(v, &newCandle)
			if err != nil {
				return err
			}
			result = append(result, newCandle)
			_ = v
		}
		return nil
	})
	return result, err
}

// LoadStream get Candles by period and returns as async stream
func (s *Bolt) LoadStream(periodStart, periodEnd time.Time) chan Candle {
	result := make(chan Candle)

	go func() {
		defer close(result)
		err := s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(bucket)
			c := b.Cursor()

			min := []byte(fmt.Sprintf("%d", periodStart.Unix()))
			max := []byte(fmt.Sprintf("%d", periodEnd.Unix()))

			for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
				newCandle := Candle{}
				if err := json.Unmarshal(v, &newCandle); err != nil {
					return err
				}
				result <- newCandle
			}
			return nil
		})
		if err != nil {
			log.Printf("[WARN] can't get candles, %v", err)
		}
	}()

	return result
}
