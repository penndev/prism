package proxy

import (
	"desktop/internal"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// netsh interface ipv4 set address name="PrismTUN" source=static addr=172.19.0.1 mask=255.255.255.255
const TUN_NAME = "utun"
const TUN_MTU = 1500
const TUN_OFFSET = 4

var TUN_IP netip.Prefix
var Routes []netip.Prefix

// 自定义网卡GUID 方便wintun复用
func init() {
	TUN_IP = netip.MustParsePrefix("172.19.0.1/32")
	Routes = []netip.Prefix{
		netip.MustParsePrefix("1.0.0.0/8"),
		netip.MustParsePrefix("2.0.0.0/7"),
		netip.MustParsePrefix("4.0.0.0/6"),
		netip.MustParsePrefix("8.0.0.0/5"),
		netip.MustParsePrefix("16.0.0.0/4"),
		netip.MustParsePrefix("32.0.0.0/3"),
		netip.MustParsePrefix("64.0.0.0/2"),
		netip.MustParsePrefix("128.0.0.0/1"),
		netip.MustParsePrefix("198.18.0.0/15"),
	}
	initElevate()
}

const sudoFile = "/etc/sudoers.d/prism-desktop"
const elevateFailAfter = 3 * time.Second

var elevateMarkerFile = "/tmp/prism-desktop-elevate.marker"

// initElevate 检查上次提权重启是否超时失败，超时则清理 sudoers
func initElevate() {
	defer os.Remove(elevateMarkerFile)
	if os.Geteuid() == 0 {
		return
	}
	data, err := os.ReadFile(elevateMarkerFile)
	if err != nil {
		return
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return
	}
	if time.Since(time.Unix(ts, 0)) <= elevateFailAfter {
		return
	}
	apple := fmt.Sprintf(`do shell script "rm -f %s" with administrator privileges`, sudoFile)
	exec.Command("osascript", "-e", apple).Run()
}

func writeElevateMarker() {
	os.WriteFile(elevateMarkerFile, []byte(fmt.Sprintf("%d", time.Now().Unix())), 0644)
}

func tunPermission() bool {
	// 已经是 root
	if os.Geteuid() == 0 {
		return true
	}

	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	u, err := user.Current()
	if err != nil {
		return false
	}

	// sudoers 已存在时先尝试提权启动；失败则继续走下方修复流程，避免误退出
	if _, err := os.Stat(sudoFile); err == nil {
		if cmd := exec.Command("sudo", "-n", exePath); cmd.Start() == nil {
			writeElevateMarker()
			time.Sleep(100 * time.Millisecond)
			internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
			return false
		}
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "sudo launch failed, repairing sudoers")
	}

	// sudoers 内容（⚠️ 不要转义路径）
	line := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: %s\n", u.Username, exePath)

	// 单行脚本（无换行）
	script := fmt.Sprintf(
		`echo '%s' > %s && chmod 440 %s`,
		line, sudoFile, sudoFile,
	)
	apple := fmt.Sprintf(`do shell script %q with administrator privileges`, script)
	cmd := exec.Command("osascript", "-e", apple)

	go func() {
		// 第一次尝试
		err := cmd.Run()

		if err != nil {
			// 修复（再写一次）
			fixScript := fmt.Sprintf(
				`echo '%s' > %s && chmod 440 %s`,
				line, sudoFile, sudoFile,
			)
			fixApple := fmt.Sprintf(`do shell script %q with administrator privileges`, fixScript)
			_ = exec.Command("osascript", "-e", fixApple).Run()
		}

		if cmd := exec.Command("sudo", "-n", exePath); cmd.Start() == nil {
			writeElevateMarker()
			time.Sleep(100 * time.Millisecond)
			internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
		} else {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "elevated launch failed")
		}
	}()

	return false
}
