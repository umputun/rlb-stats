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

func TestBolt_Close(t *testing.T) {
	file, err := os.CreateTemp("/tmp/", "bolt_test.bd.")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	s, err := NewBolt(file.Name())
	require.NoError(t, err)
	assert.NoError(t, s.Close())
}

