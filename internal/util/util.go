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

func GetKeys[K comparable, V any](m map[K]V) []K {
	keys := []K{}
	for k, _ := range m {
		keys = append(keys, k)
	}
	return keys
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
