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

const TUN_NAME = "utun"
const TUN_MTU = 1500
const TUN_OFFSET = 4

var TUN_IP netip.Prefix
var TUN_IP6 netip.Prefix
var Routes []netip.Prefix

const sudoFile = "/etc/sudoers.d/prism-desktop"
const elevateMarkerFile = "/tmp/prism-desktop-elevate.marker"

// 提权进程超过这个时间还没接上，视为失败，清掉免密 sudoers。
const elevateFailAfter = 3 * time.Second

func init() {
	TUN_IP = netip.MustParsePrefix("172.19.0.1/32")
	TUN_IP6 = netip.MustParsePrefix("fd19::1/128")
	Routes = []netip.Prefix{
		// 0.0.0.0/8 不收不然就环路了。
		// netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("1.0.0.0/8"),
		netip.MustParsePrefix("2.0.0.0/7"),
		netip.MustParsePrefix("4.0.0.0/6"),
		netip.MustParsePrefix("8.0.0.0/5"),
		netip.MustParsePrefix("16.0.0.0/4"),
		netip.MustParsePrefix("32.0.0.0/3"),
		netip.MustParsePrefix("64.0.0.0/2"),
		netip.MustParsePrefix("128.0.0.0/1"),
		// ::/8 不收不然就环路了。
		// netip.MustParsePrefix("::/8"),
		netip.MustParsePrefix("100::/8"),
		netip.MustParsePrefix("200::/7"),
		netip.MustParsePrefix("400::/6"),
		netip.MustParsePrefix("800::/5"),
		netip.MustParsePrefix("1000::/4"),
		netip.MustParsePrefix("2000::/3"),
		netip.MustParsePrefix("4000::/2"),
		netip.MustParsePrefix("8000::/1"),
	}

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
	exec.Command("osascript", "-e", fmt.Sprintf(
		`do shell script "rm -f %s" with administrator privileges`, sudoFile,
	)).Run()
}

func startElevated(exe string) bool {
	if exec.Command("sudo", "-n", exe).Start() != nil {
		return false
	}
	os.WriteFile(elevateMarkerFile, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
	time.Sleep(100 * time.Millisecond)
	internal.App.Event.Emit(internal.AppConfig.EventNameServiceAppQuit, true)
	return true
}

func tunPermission() bool {
	if os.Geteuid() == 0 {
		return true
	}

	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	// 免密规则要写 sudo 真正执行的路径，不能写符号链接。
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	u, err := user.Current()
	if err != nil {
		return false
	}
	// sudoers 用户名里这些字符有语法含义，写进去规则会错。
	if strings.ContainsAny(u.Username, " \t\n#,:=\\") {
		return false
	}

	// 规则已在时先直接提权；失败再往下重写，避免半截文件导致进程直接退出。
	if _, err := os.Stat(sudoFile); err == nil {
		if startElevated(exePath) {
			return false
		}
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "sudo launch failed, repairing sudoers")
	}

	// osascript 以 root 跑这段脚本。整行用单引号包住，路径里的单引号才不会跳出引号。
	line := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: %s", u.Username, exePath)
	quoted := "'" + strings.ReplaceAll(line, "'", `'\''`) + "'"
	script := fmt.Sprintf("echo %s > %s && chmod 440 %s", quoted, sudoFile, sudoFile)
	apple := fmt.Sprintf(`do shell script %q with administrator privileges`, script)

	go func() {
		// osascript 偶发失败，同样的脚本再试一次。
		for range 2 {
			if exec.Command("osascript", "-e", apple).Run() == nil {
				break
			}
		}
		if !startElevated(exePath) {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "elevated launch failed")
		}
	}()
	return false
}
