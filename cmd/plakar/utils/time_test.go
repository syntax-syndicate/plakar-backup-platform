package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimeFlag(t *testing.T) {
	// Test case: Empty input
	t1, err := ParseTimeFlag("")
	require.NoError(t, err)
	require.True(t, t1.IsZero())

	// Test case: RFC3339 format
	input := "2025-04-15T10:00:00Z"
	expected, _ := time.Parse(time.RFC3339, input)
	t2, err := ParseTimeFlag(input)
	require.NoError(t, err)
	require.Equal(t, expected, t2)

	// Test case: Date format "2006-01-02"
	input = "2025-04-15"
	expected, _ = time.Parse("2006-01-02", input)
	t3, err := ParseTimeFlag(input)
	require.NoError(t, err)
	require.Equal(t, expected, t3)

	// Test case: Date format "2006/01/02"
	input = "2025/04/15"
	expected, _ = time.Parse("2006/01/02", input)
	t4, err := ParseTimeFlag(input)
	require.NoError(t, err)
	require.Equal(t, expected, t4)

	// Test case: DateTime format "2006-01-02 15:04:05"
	input = "2025-04-15 10:00:00"
	expected, _ = time.Parse("2006-01-02 15:04:05", input)
	t5, err := ParseTimeFlag(input)
	require.NoError(t, err)
	require.Equal(t, expected, t5)

	// Test case: Duration format (e.g., "2h")
	input = "2h"
	now := time.Now()
	t6, err := ParseTimeFlag(input)
	require.NoError(t, err)
	require.WithinDuration(t, now.Add(-2*time.Hour), t6, time.Second)

	// Test case: Invalid format
	input = "invalid-time-format"
	t7, err := ParseTimeFlag(input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid time format")
	require.True(t, t7.IsZero())
}
