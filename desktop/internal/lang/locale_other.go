//go:build !windows

package lang

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func osLanguageTag() string {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
		if err == nil {
			s := string(out)
			if i := strings.Index(s, `"`); i >= 0 {
				s = s[i+1:]
				if j := strings.Index(s, `"`); j >= 0 {
					return s[:j]
				}
			}
		}
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(os.Getenv(key))
		if v != "" && v != "C" && v != "POSIX" {
			return v
		}
	}
	return ""
}
