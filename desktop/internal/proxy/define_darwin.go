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
var TUN_IP6 netip.Prefix
var Routes []netip.Prefix

// 自定义网卡GUID 方便wintun复用
func init() {
	TUN_IP = netip.MustParsePrefix("172.19.0.1/32")
	TUN_IP6 = netip.MustParsePrefix("fd19::1/128")
	Routes = []netip.Prefix{
		netip.MustParsePrefix("1.0.0.0/8"),
		netip.MustParsePrefix("2.0.0.0/7"),
		netip.MustParsePrefix("4.0.0.0/6"),
		netip.MustParsePrefix("8.0.0.0/5"),
		netip.MustParsePrefix("16.0.0.0/4"),
		netip.MustParsePrefix("32.0.0.0/3"),
		netip.MustParsePrefix("64.0.0.0/2"),
		netip.MustParsePrefix("128.0.0.0/1"),
	}
	initElevate()
}

const sudoFile = "/etc/sudoers.d/prism-desktop"
const elevateFailAfter = 3 * time.Second

var elevateMarkerFile = "/tmp/prism-desktop-elevate.marker"

// shellQuote 用单引号包住字符串，内部出现的单引号按 shell 惯例拆成
// 「关引号 + 转义单引号 + 开引号」，避免跳出引用。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

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
	if strings.ContainsAny(u.Username, " \t\n#,:=\\") {
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

	// 这段会以 root 身份执行，exePath 和用户名必须做 shell 引用，
	// 否则路径里的单引号就能跳出引号拼进任意命令。
	line := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: %s", u.Username, exePath)
	script := fmt.Sprintf(
		"echo %s > %s && chmod 440 %s",
		shellQuote(line), sudoFile, sudoFile,
	)
	apple := fmt.Sprintf(`do shell script %q with administrator privileges`, script)

	go func() {
		// osascript 偶发失败，同样的脚本再试一次
		for i := 0; i < 2; i++ {
			if err := exec.Command("osascript", "-e", apple).Run(); err == nil {
				break
			}
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
