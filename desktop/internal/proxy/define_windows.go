package proxy

import (
	"desktop/internal"
	"net/netip"
	"os"
	"os/exec"
	"syscall"
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

// 自定义网卡GUID 方便wintun复用
func init() {
	TUN_IP = netip.MustParsePrefix("172.19.0.1/32")
	TUN_IP6 = netip.MustParsePrefix("fd19::1/128")
	Routes = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/0"),
		netip.MustParsePrefix("::/0"),
	}
	// 设置tun设备名称标识和guid
	tun.WintunTunnelType = TUN_NAME
	tun.WintunStaticRequestedGUID = &windows.GUID{
		Data1: 0x8ceeab57,
		Data2: 0x7cb2,
		Data3: 0x469f,
		Data4: [8]byte{0x91, 0x3b, 0xea, 0xeb, 0x22, 0xe2, 0x28, 0x24},
	}
}

func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}

// runElevated 用 ShellExecute runas 拉起自己。不经过 powershell / cmd，避免弹出命令行窗口。
func runElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
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
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	r, _, callErr := shell32.NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	if info.hProcess != 0 {
		windows.CloseHandle(windows.Handle(info.hProcess))
	}
	if r == 0 {
		return callErr
	}
	return nil
}

func tunPermission() bool {
	cmd := exec.Command("net", "session")
	hideConsole(cmd)
	if cmd.Run() == nil {
		return true
	}
	// 取消 UAC 时返回错误，当前进程继续；同意才退出，让提权后的实例接手。
	if err := runElevated(); err != nil {
		return false
	}
	internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
	return false
}
