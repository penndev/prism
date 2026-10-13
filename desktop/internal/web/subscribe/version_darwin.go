//go:build darwin

package subscribe

import (
	"os"
	"path/filepath"
	"strings"
)

func appVersion() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "..", "Info.plist"))
	if err != nil {
		return ""
	}
	const key = "<key>CFBundleShortVersionString</key>"
	text := string(data)
	i := strings.Index(text, key)
	if i < 0 {
		return ""
	}
	rest := text[i+len(key):]
	start := strings.Index(rest, "<string>")
	end := strings.Index(rest, "</string>")
	if start < 0 || end < start {
		return ""
	}
	return strings.TrimSpace(rest[start+len("<string>") : end])
}
