package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestSaveAndLoadLogEntryBolt(t *testing.T) {
	// normal flow
	file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
	assert.Nil(t, err, "created temp file")

	s, err := NewBolt(file.Name())
	assert.Nil(t, err, "engine created")

	testCandle := NewCandle()
	testCandle.StartMinute = time.Unix(0, 0)

	assert.Nil(t, s.Save(testCandle), "saved fine")
	savedCandle, err := s.Load(context.Background(), time.Unix(0, 0), time.Unix(0, 0).Add(time.Hour))
	assert.Nil(t, err, "key found")
	require.NotEqual(t, []Candle{}, savedCandle, "key found")
	assert.EqualValues(t, testCandle, savedCandle[0], "matches loaded msg")

	assert.Nil(t, os.Remove(file.Name()), "removed fine")

	// broken DB file
	badBolt, err := NewBolt("/dev/null")
	assert.Nil(t, badBolt, "nil returned on error")
	assert.NotNil(t, err, "engine not created")
}

func TestBolt_SaveMergesSameMinute(t *testing.T) {
	file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	defer os.Remove(file.Name())

	s, err := NewBolt(file.Name())
	require.NoError(t, err)
	defer s.Close()

	minute := time.Unix(0, 0)

	first := Candle{
		StartMinute: minute,
		Nodes: map[string]Info{
			"n1":  {Volume: 2, Files: map[string]int{"a.mp3": 2}},
			"all": {Volume: 2, Files: map[string]int{"a.mp3": 2}},
		},
	}
	second := Candle{
		StartMinute: minute,
		Nodes: map[string]Info{
			"n1":  {Volume: 1, Files: map[string]int{"a.mp3": 1}},
			"n2":  {Volume: 3, Files: map[string]int{"b.mp3": 3}},
			"all": {Volume: 4, Files: map[string]int{"a.mp3": 1, "b.mp3": 3}},
		},
	}

	require.NoError(t, s.Save(first))
	require.NoError(t, s.Save(second), "second write for the same minute must merge, not overwrite")

	loaded, err := s.Load(context.Background(), minute, minute.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, loaded, 1, "same minute stays a single candle")

	got := loaded[0]
	assert.Equal(t, 3, got.Nodes["n1"].Volume, "n1 volume summed across both writes")
	assert.Equal(t, 3, got.Nodes["n1"].Files["a.mp3"])
	assert.Equal(t, 3, got.Nodes["n2"].Volume, "node only in second write is kept")
	assert.Equal(t, 6, got.Nodes["all"].Volume, "all volume summed")
	assert.Equal(t, 3, got.Nodes["all"].Files["a.mp3"])
	assert.Equal(t, 3, got.Nodes["all"].Files["b.mp3"])
}

func TestBolt_SaveOverwritesCorruptEntry(t *testing.T) {
	file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	defer os.Remove(file.Name())

	s, err := NewBolt(file.Name())
	require.NoError(t, err)
	defer s.Close()

	minute := time.Unix(0, 0)
	key := fmt.Appendf(nil, "%d", minute.Unix())

	// plant a corrupt (non-JSON) value at the key
	require.NoError(t, s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucket).Put(key, []byte("not json"))
	}))

	candle := Candle{StartMinute: minute, Nodes: map[string]Info{
		"all": {Volume: 5, Files: map[string]int{"a.mp3": 5}},
	}}
	// save must overwrite the unreadable entry rather than fail
	require.NoError(t, s.Save(candle))

	loaded, err := s.Load(context.Background(), minute, minute.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, 5, loaded[0].Nodes["all"].Volume, "corrupt entry overwritten with the new candle")
}

func TestBolt_Close(t *testing.T) {
	file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	s, err := NewBolt(file.Name())
	require.NoError(t, err)
	assert.NoError(t, s.Close())
}
