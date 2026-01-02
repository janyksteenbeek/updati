package updater

import (
	"fmt"
	"os"
)

// fileHash returns a simple hash of a file for change detection
func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "empty", nil
	}

	start := data[:min(10, len(data))]
	end := data[max(0, len(data)-10):]

	return fmt.Sprintf("%d-%x-%x", len(data), start, end), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

