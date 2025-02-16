package utils

import (
	"fmt"
	"time"
)

func ParseTimeFlag(input string) (time.Time, error) {
	if input == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		time.RFC3339,          // e.g. "2006-01-02T15:04:05Z07:00"
		"2006-01-02",          // e.g. "2006-01-02"
		"2006/01/02",          // e.g. "2006/01/02"
		"2006-01-02 15:04:05", // e.g. "2006-01-02 15:04:05"
		"01/02/2006 15:04",    // e.g. "01/02/2006 15:04"
	}

	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, input)
		if err == nil {
			return t, nil
		}
	}

	// If none of the date layouts match, try to parse it as a duration.
	d, err := time.ParseDuration(input)
	if err == nil {
		return time.Now().Add(-d), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %q", input)
}
