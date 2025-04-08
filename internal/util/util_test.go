package util

import (
	"strconv"
	"testing"
)

func TestHumanReadableSize(t *testing.T) {
	var tests = []struct {
		bytes    int64
		expected string
	}{
		{1, "1B"},
		{-1, "-1B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KiB"},
		{1025, "1.0KiB"},
		{1024 * 2, "2.0KiB"},
		{1024 * 9.5, "9.5KiB"},
		{1024 * 10, "10KiB"},
		{1024 * 100, "100KiB"},
		{1024 * 1000, "1000KiB"},
		{1024 * 1024, "1.0MiB"},
		{1024 * 1024 * 1024, "1.0GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0TiB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0PiB"},
		{1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1024PiB"},
	}

	for _, test := range tests {
		t.Run(strconv.FormatInt(test.bytes, 10), func(s *testing.T) {
			actual := HumanReadableSize(test.bytes)
			if actual != test.expected {
				s.Errorf("expected %s, got %s", test.expected, actual)
			}
		})
	}

}
