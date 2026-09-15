//go:build windows

package lang

import "syscall"

var procGetUserDefaultUILanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

func osLanguageTag() string {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	if uint16(r)&0x3FF == 0x04 {
		return "zh"
	}
	return "en"
}
