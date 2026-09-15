//go:build windows

package lang

import (
	"fmt"
	"syscall"
	"testing"
	"unsafe"
)

func TestOSLanguageTag(t *testing.T) {
	r, _, callErr := procGetUserDefaultUILanguage.Call()
	langID := uint16(r)
	fmt.Printf("GetUserDefaultUILanguage raw=%d (0x%04X) primary=0x%03X err=%v\n", langID, langID, langID&0x3FF, callErr)
	fmt.Printf("osLanguageTag() = %q\n", osLanguageTag())
	fmt.Printf("Resolve(%q) = %q\n", LocaleSystem, Resolve(LocaleSystem))

	buf := make([]uint16, 85)
	n, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName").Call(
		uintptr(unsafe.Pointer(&buf[0])),
		85,
	)
	fmt.Printf("GetUserDefaultLocaleName n=%d err=%v name=%q\n", n, err, syscall.UTF16ToString(buf))
}
