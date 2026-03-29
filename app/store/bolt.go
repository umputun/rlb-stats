package store

import (
	"bytes"
	"context"
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
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bucket)
		return e
	})
	if err != nil {
		return nil, err
	}
	return &Bolt{db: db}, nil
}

// Close closes the underlying boltdb
func (s *Bolt) Close() error {
	return s.db.Close()
}

// Save Candles with starting minute time.Unix() as a key for bolt range query.
// keys are decimal Unix timestamps; lexicographic ordering matches numeric ordering
// because all timestamps since 1973 are 10 digits (remains true until 2286).
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

// TimeRange returns the oldest and newest candle timestamps in the store.
// returns zero times with no error if the bucket is empty.
func (s *Bolt) TimeRange(ctx context.Context) (oldest, newest time.Time, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		c := b.Cursor()

		firstKey, _ := c.First()
		if firstKey == nil {
			return nil // empty bucket
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var firstUnix int64
		if _, scanErr := fmt.Sscanf(string(firstKey), "%d", &firstUnix); scanErr != nil {
			return fmt.Errorf("parse oldest key %q: %w", firstKey, scanErr)
		}
		oldest = time.Unix(firstUnix, 0)

		lastKey, _ := c.Last()
		var lastUnix int64
		if _, scanErr := fmt.Sscanf(string(lastKey), "%d", &lastUnix); scanErr != nil {
			return fmt.Errorf("parse newest key %q: %w", lastKey, scanErr)
		}
		newest = time.Unix(lastUnix, 0)

		return nil
	})
	return oldest, newest, err
}

// Load Candles by period
func (s *Bolt) Load(ctx context.Context, periodStart, periodEnd time.Time) (result []Candle, err error) {
	result = []Candle{}
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		c := b.Cursor()

		minimum := fmt.Appendf(nil, "%d", periodStart.Unix())
		maximum := fmt.Appendf(nil, "%d", periodEnd.Unix())

		for k, v := c.Seek(minimum); k != nil && bytes.Compare(k, maximum) <= 0; k, v = c.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			newCandle := Candle{}
			err = json.Unmarshal(v, &newCandle)
			if err != nil {
				return err
			}
			result = append(result, newCandle)
		}
		return nil
	})
	return result, err
}
