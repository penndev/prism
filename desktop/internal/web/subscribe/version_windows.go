//go:build windows

package subscribe

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func appVersion() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size == 0 {
		return ""
	}
	buf := make([]byte, size)
	if err = windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])); err != nil {
		return ""
	}
	var info *windows.VS_FIXEDFILEINFO
	var length uint32
	if err = windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\`, unsafe.Pointer(&info), &length); err != nil || info == nil {
		return ""
	}
	parts := []uint32{
		info.ProductVersionMS >> 16,
		info.ProductVersionMS & 0xffff,
		info.ProductVersionLS >> 16,
		info.ProductVersionLS & 0xffff,
	}
	for len(parts) > 1 && parts[len(parts)-1] == 0 {
		parts = parts[:len(parts)-1]
	}
	text := fmt.Sprintf("%d", parts[0])
	for _, part := range parts[1:] {
		text += fmt.Sprintf(".%d", part)
	}
	return text
}
