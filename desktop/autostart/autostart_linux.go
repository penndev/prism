//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopFileName = "prism.desktop"

func desktopEntryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "autostart")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, desktopFileName), nil
}

func enable() error {
	path, err := desktopEntryPath()
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Socks5 proxy desktop app
Exec=%s
Terminal=false
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
`, appName, exe)

	return os.WriteFile(path, []byte(content), 0644)
}

func disable() error {
	path, err := desktopEntryPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
