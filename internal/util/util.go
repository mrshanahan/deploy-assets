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
