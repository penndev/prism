//go:build darwin

package route

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

func waitDevAddrReady(tunName string, prefix netip.Prefix) error {
	tunIP := prefix.Addr().String()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			out, err := exec.Command("ifconfig", tunName).CombinedOutput()
			if err != nil {
				continue
			}
			if strings.Contains(string(out), tunIP) {
				return nil
			}
		case <-timeout:
			return errors.New("cant set dev static ip")
		}
	}
}

func SetDevAddr(tunName string, prefix netip.Prefix) error {
	ip := prefix.Addr().String()
	var commands [][]string
	if prefix.Addr().Is4() {
		mask := prefixMask(prefix).String()
		commands = [][]string{
			{"ifconfig", tunName, "inet", ip, ip, "netmask", mask, "up"},
			{"ifconfig", tunName, "inet", ip, "netmask", mask, "up"},
		}
	} else {
		commands = [][]string{
			{"ifconfig", tunName, "inet6", ip, "prefixlen", "128"},
		}
	}

	var lastErr error
	for _, args := range commands {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err == nil {
			return waitDevAddrReady(tunName, prefix)
		}
		lastErr = fmt.Errorf("ifconfig failed: %v, output: %s", err, out)
	}
	return lastErr
}

func SetRouteAddr(prefix netip.Prefix, gateway net.IP) error {
	gw := gateway.String()
	var commands [][]string
	if prefix.Addr().Is6() {
		commands = [][]string{
			{"route", "-n", "add", "-inet6", prefix.String(), gw},
			{"route", "-n", "add", "-inet6", "-net", prefix.String(), gw},
		}
	} else {
		netStr := prefix.Addr().String()
		mask := prefixMask(prefix).String()
		commands = [][]string{
			{"route", "-n", "add", "-net", netStr, mask, gw},
			{"route", "-n", "add", "-net", netStr, "-netmask", mask, gw},
		}
	}

	var lastErr error
	for _, args := range commands {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err == nil {
			return nil
		}
		// 路由已经在，重复添加不算失败。
		if strings.Contains(strings.ToLower(string(out)), "exists") {
			return nil
		}
		lastErr = fmt.Errorf("route failed: %v, output: %s", err, out)
	}
	return lastErr
}

// dnsSetup 是改掉的那次系统 DNS。restore 交给 networksetup 写回去，active 给上游用。
type dnsSetup struct {
	service string
	restore []string
	active  []string
}

var savedDNS atomic.Pointer[dnsSetup]

func addServer(servers []string, raw string) []string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || ip.IsLoopback() || ip.String() == tunDNS {
		return servers
	}
	s := ip.String()
	if slices.Contains(servers, s) {
		return servers
	}
	return append(servers, s)
}

// CurrentDNS 返回系统正在用的 DNS。改过之后 scutil 里只剩 tunDNS，用改之前记下的那组。
func CurrentDNS(string) []string {
	if saved := savedDNS.Load(); saved != nil {
		return saved.active
	}
	shown, err := exec.Command("scutil", "--dns").CombinedOutput()
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(shown), "\n") {
		if strings.Contains(line, "scoped queries") {
			break
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "nameserver[") {
			continue
		}
		_, addr, ok := strings.Cut(line, " : ")
		if !ok {
			continue
		}
		out = addServer(out, addr)
	}
	return out
}

// macOS 上发往局域网 DNS 的查询绑着物理网卡，主机路由盖不住。
// 改默认出口那张网卡的 DNS。DHCP 没手写服务器时，恢复用 Empty，把租约交回去。
func setDevDNS(tunName string) {
	if savedDNS.Load() != nil {
		return
	}
	out, err := exec.Command("route", "-n", "get", "-inet", "default").CombinedOutput()
	if err != nil {
		status("dns: %v, %s", err, out)
		return
	}
	var iface string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "interface:" {
			iface = fields[1]
			break
		}
	}
	if iface == "" || iface == tunName {
		status("dns: no physical default route")
		return
	}
	order, err := exec.Command("networksetup", "-listnetworkserviceorder").CombinedOutput()
	if err != nil {
		status("dns: %v, %s", err, order)
		return
	}
	var service, pending string
	for _, line := range strings.Split(string(order), "\n") {
		line = strings.TrimSpace(line)
		// "(1) Wi-Fi" 是服务名，下一行 "Device: en0)" 才对得上接口。
		if strings.HasPrefix(line, "(") && strings.Contains(line, ") ") && !strings.Contains(line, "Hardware Port") {
			_, name, ok := strings.Cut(line, ") ")
			if ok {
				pending = strings.TrimSpace(name)
			}
			continue
		}
		if pending != "" && strings.Contains(line, "Device: "+iface+")") {
			service = pending
			break
		}
	}
	if service == "" {
		status("dns: no service for %s", iface)
		return
	}
	got, err := exec.Command("networksetup", "-getdnsservers", service).CombinedOutput()
	if err != nil {
		status("dns: %v, %s", err, got)
		return
	}
	restore := []string{"Empty"}
	if !strings.Contains(string(got), "aren't any DNS") {
		restore = nil
		for _, line := range strings.Split(string(got), "\n") {
			ip := net.ParseIP(strings.TrimSpace(line))
			if ip == nil || ip.String() == tunDNS {
				continue
			}
			restore = append(restore, ip.String())
		}
		if len(restore) == 0 {
			restore = []string{"Empty"}
		}
	}
	active := CurrentDNS("")
	// 上次退出没恢复时，scutil 里只剩 tunDNS，上游改从 DHCP 租约读。
	if len(active) == 0 {
		if opt, err := exec.Command("ipconfig", "getoption", iface, "domain_name_server").CombinedOutput(); err == nil {
			for _, field := range strings.Fields(string(opt)) {
				ip := net.ParseIP(field)
				if ip == nil || ip.String() == tunDNS {
					continue
				}
				if s := ip.String(); !slices.Contains(active, s) {
					active = append(active, s)
				}
			}
		}
	}
	savedDNS.Store(&dnsSetup{service: service, restore: restore, active: active})
	if out, err = exec.Command("networksetup", "-setdnsservers", service, tunDNS).CombinedOutput(); err != nil {
		status("set dns failed: %v, %s", err, out)
	}
}

func RestoreDNS() {
	saved := savedDNS.Swap(nil)
	if saved == nil {
		return
	}
	args := append([]string{"-setdnsservers", saved.service}, saved.restore...)
	out, err := exec.Command("networksetup", args...).CombinedOutput()
	if err != nil {
		status("restore dns failed: %v, %s", err, out)
	}
}
