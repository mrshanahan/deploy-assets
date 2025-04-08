package util

import (
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
