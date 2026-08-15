package proxy

import (
	"desktop/internal"
	"net/netip"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/tun"
)

const TUN_NAME = "prise-tun"
const TUN_MTU = 0
const TUN_OFFSET = 0

var TUN_IP netip.Prefix
var Routes []netip.Prefix

// 自定义网卡GUID 方便wintun复用
func init() {
	TUN_IP = netip.MustParsePrefix("172.19.0.1/32")
	Routes = []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}
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
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}

func tunPermission() bool {
	cmd := exec.Command("net", "session")
	hideConsole(cmd)
	if cmd.Run() == nil {
		return true
	}
	exePath, _ := os.Executable()
	cmd = exec.Command("powershell",
		"-Command",
		`Start-Process "`+exePath+`" -Verb RunAs`,
	)
	hideConsole(cmd)
	_ = cmd.Start()
	// 退出当前进程
	go func() {
		time.Sleep(100 * time.Millisecond)
		internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
	}()
	return false
}
