//go:build !windows && !darwin

package autostart

func enable() error {
	return errUnsupported
}

func disable() error {
	return errUnsupported
}
