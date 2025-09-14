package utils

import (
	"fmt"
	"os"
	"sync"
)

var (
	TempDir string
	once    sync.Once
)

func GetTempDir() string {
	once.Do(func() {
		dir, err := os.MkdirTemp("", "supago-*")
		if err != nil {
			panic(fmt.Sprintf("failed to create temp dir: %v", err))
		}
		TempDir = dir
	})
	return TempDir
}
