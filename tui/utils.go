package tui

import (
	"os"
	"strings"
)

func readFile(path string) []string {
	file, err := os.ReadFile(path)
	if err != nil {
		return []string { "failed to read file"}
	}

	content := strings.Split(string(file), "\n")
	return content
}
