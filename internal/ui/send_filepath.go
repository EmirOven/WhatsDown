package ui

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// parseFilePathFromText checks if text contains an existing file path or URI.
func parseFilePathFromText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") || line == "copy" || line == "cut" {
			continue
		}
		if (strings.HasPrefix(line, "'") && strings.HasSuffix(line, "'")) ||
			(strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"")) {
			if len(line) >= 2 {
				line = line[1 : len(line)-1]
			}
		}

		targetPath := ""
		if strings.HasPrefix(line, "file://") {
			if u, err := url.Parse(line); err == nil {
				targetPath = u.Path
				if unescaped, err := url.PathUnescape(targetPath); err == nil {
					targetPath = unescaped
				}
			}
		} else if strings.HasPrefix(line, "/") {
			targetPath = line
		} else if strings.HasPrefix(line, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				targetPath = filepath.Join(home, line[2:])
			}
		}

		if targetPath != "" {
			if fi, err := os.Stat(targetPath); err == nil && !fi.IsDir() {
				return targetPath, true
			}
		}
	}
	return "", false
}
