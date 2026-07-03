//go:build windows

package autostart

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func openRunKey() (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
}

func enable() error {
	k, err := openRunKey()
	if err != nil {
		return err
	}
	defer k.Close()

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return k.SetStringValue(appName, exe)
}

func disable() error {
	k, err := openRunKey()
	if err != nil {
		return err
	}
	defer k.Close()

	err = k.DeleteValue(appName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
