//go:build !windows && !darwin && !linux

package autostart

func enable() error {
	return errUnsupported
}

func disable() error {
	return errUnsupported
}
