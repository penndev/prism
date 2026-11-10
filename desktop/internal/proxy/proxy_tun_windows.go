package proxy

import (
	"desktop/internal"
	"net/netip"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/tun"
)

const TUN_NAME = "prise-tun"
const TUN_MTU = 0
const TUN_OFFSET = 0

var TUN_IP netip.Prefix
var TUN_IP6 netip.Prefix
var Routes []netip.Prefix

// 自定义网卡 GUID，方便 wintun 复用同一块网卡。
func init() {
	TUN_IP = netip.MustParsePrefix("172.19.0.1/32")
	TUN_IP6 = netip.MustParsePrefix("fd19::1/128")
	Routes = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/0"),
		netip.MustParsePrefix("::/0"),
	}
	tun.WintunTunnelType = TUN_NAME
	tun.WintunStaticRequestedGUID = &windows.GUID{
		Data1: 0x8ceeab57,
		Data2: 0x7cb2,
		Data3: 0x469f,
		Data4: [8]byte{0x91, 0x3b, 0xea, 0xeb, 0x22, 0xe2, 0x28, 0x24},
	}
}

func runElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	// 字段顺序与 SHELLEXECUTEINFOW 一致。
	info := struct {
		cbSize       uint32
		fMask        uint32
		hwnd         uintptr
		lpVerb       *uint16
		lpFile       *uint16
		lpParameters *uint16
		lpDirectory  *uint16
		nShow        int32
		hInstApp     uintptr
		lpIDList     uintptr
		lpClass      *uint16
		hkeyClass    uintptr
		dwHotKey     uint32
		hMonitor     uintptr
		hProcess     uintptr
	}{
		fMask:  0x00000040 | 0x00000100, // SEE_MASK_NOCLOSEPROCESS | SEE_MASK_NOASYNC
		lpVerb: verb,
		lpFile: file,
		nShow:  windows.SW_SHOWNORMAL,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))
	r, _, callErr := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	if info.hProcess != 0 {
		windows.CloseHandle(windows.Handle(info.hProcess))
	}
	if r == 0 {
		return callErr
	}
	return nil
}

func tunPermission() bool {
	var token windows.Token
	if windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token) == nil {
		ok := token.IsElevated()
		token.Close()
		if ok {
			return true
		}
	}
	// 取消 UAC 时返回错误，当前进程继续；同意才退出，让提权后的实例接手。
	if runElevated() != nil {
		return false
	}
	internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
	return false
}
