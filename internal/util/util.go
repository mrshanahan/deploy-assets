package util

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func GetTimestampedFileName(prefix string) string {
	timestamp := strings.Replace(time.Now().Format(time.RFC3339Nano), ":", "", -1)
	name := fmt.Sprintf("%s-%s", prefix, timestamp)
	return name
}

func GetTempFilePath(prefix string) string {
	name := GetTimestampedFileName(prefix)
	path := filepath.Join("/tmp", name)
	return path
}

func Keys[K comparable, V any](m map[K]V) []K {
	keys := []K{}
	for k, _ := range m {
		keys = append(keys, k)
	}
	return keys
}

func Values[K comparable, V any](m map[K]V) []V {
	values := []V{}
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

type Set[T comparable] struct {
	m map[T]bool
}

func NewSet[T comparable](xs ...T) Set[T] {
	s := Set[T]{make(map[T]bool)}
	for _, x := range xs {
		s.m[x] = true
	}
	return s
}

func (s Set[T]) Contains(x T) bool {
	_, prs := s.m[x]
	return prs
}

// func (s Set[T]) Union(xs ...T)

func Map[S any, T any](xs []S, f func(S) T) []T {
	ys := []T{}
	for _, x := range xs {
		ys = append(ys, f(x))
	}
	return ys
}

func Filter[T any](xs []T, p func(T) bool) []T {
	ys := []T{}
	for _, x := range xs {
		if p(x) {
			ys = append(ys, x)
		}
	}
	return ys
}

func All[T any](xs []T, p func(T) bool) bool {
	for _, x := range xs {
		if !p(x) {
			return false
		}
	}
	return true
}

func HumanReadableSize(bytes int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	unitIdx := 0
	coeff := float64(bytes)
	for coeff >= 1024 && unitIdx < len(units)-1 {
		coeff = coeff / 1024
		unitIdx += 1
	}
	if unitIdx == 0 {
		return fmt.Sprintf("%d%s", bytes, units[unitIdx])
	}
	if coeff >= 10 {
		return fmt.Sprintf("%.0f%s", coeff, units[unitIdx])

	}
	return fmt.Sprintf("%.1f%s", coeff, units[unitIdx])
}

// dropCR & the definition of ScanLinesOrUntil were basically copied verbatim
// from the golang source: https://github.com/golang/go/blob/master/src/bufio/scan.go#L341-L369

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func ScanUntil(bs ...byte) func([]byte, bool) (int, []byte, error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		for _, b := range bs {
			if i := bytes.IndexByte(data, b); i >= 0 {
				// We have a full newline-terminated line.
				// MRS: OR terminated by whatever was passed in!
				return i + 1, dropCR(data[0:i]), nil
			}
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), dropCR(data), nil
		}
		// Request more data.
		return 0, nil, nil
	}
}
