package proxy

import (
	"desktop/internal"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"syscall"

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
	exePath = strings.ReplaceAll(exePath, `"`, "`\"")
	cmd = exec.Command("powershell",
		"-Command",
		`Start-Process "`+exePath+`" -Verb RunAs`,
	)
	hideConsole(cmd)
	// Run 会等到 UAC 结束：取消就返回错误，当前进程继续；同意才退出让新实例接手。
	if err := cmd.Run(); err != nil {
		return false
	}
	internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
	return false
}
