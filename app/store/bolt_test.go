package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBolt_TimeRange(t *testing.T) {
	t.Run("non-empty DB returns correct oldest and newest", func(t *testing.T) {
		file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		s, err := NewBolt(file.Name())
		require.NoError(t, err)
		defer s.Close()

		t1 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 6, 20, 14, 30, 0, 0, time.UTC)
		t3 := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)

		for _, ts := range []time.Time{t1, t2, t3} {
			c := NewCandle()
			c.StartMinute = ts
			require.NoError(t, s.Save(c))
		}

		oldest, newest, err := s.TimeRange(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, t1.Unix(), oldest.Unix(), "oldest should be earliest saved candle")
		assert.Equal(t, t2.Unix(), newest.Unix(), "newest should be latest saved candle")
	})

	t.Run("empty DB returns zero times", func(t *testing.T) {
		file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		s, err := NewBolt(file.Name())
		require.NoError(t, err)
		defer s.Close()

		oldest, newest, err := s.TimeRange(context.Background())
		assert.NoError(t, err)
		assert.True(t, oldest.IsZero(), "oldest should be zero for empty DB")
		assert.True(t, newest.IsZero(), "newest should be zero for empty DB")
	})

	t.Run("single entry returns same oldest and newest", func(t *testing.T) {
		file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		s, err := NewBolt(file.Name())
		require.NoError(t, err)
		defer s.Close()

		ts := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
		c := NewCandle()
		c.StartMinute = ts
		require.NoError(t, s.Save(c))

		oldest, newest, err := s.TimeRange(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, ts.Unix(), oldest.Unix())
		assert.Equal(t, ts.Unix(), newest.Unix())
	})
}

func TestBolt_Close(t *testing.T) {
	file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	s, err := NewBolt(file.Name())
	require.NoError(t, err)
	assert.NoError(t, s.Close())
}
