//go:build darwin
// +build darwin

package route

import (
	"desktop/internal"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strings"
	"time"
)

func prefixMask(p netip.Prefix) net.IP {
	if !p.IsValid() {
		return nil
	}
	if p.Addr().Is4() {
		return net.IP(net.CIDRMask(p.Bits(), 32))
	}
	return net.IP(net.CIDRMask(p.Bits(), 128))
}

func waitDevAddrReady(tunName string, tunPrefix netip.Prefix) error {
	tunIP := tunPrefix.Addr().String()
	ticker := time.NewTicker(200 * time.Millisecond)
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

// 给设备设置静态 IP (macOS)
func SetDevAddr(tunName string, prefix netip.Prefix) error {
	if !prefix.IsValid() {
		return fmt.Errorf("invalid prefix")
	}

	ipStr := prefix.Addr().String()
	var tries [][]string
	if prefix.Addr().Is4() {
		maskStr := prefixMask(prefix).String()
		tries = [][]string{
			{"ifconfig", tunName, "inet", ipStr, ipStr, "netmask", maskStr, "up"},
			{"ifconfig", tunName, "inet", ipStr, "netmask", maskStr, "up"},
		}
	} else if prefix.Addr().Is6() {
		tries = [][]string{
			{"ifconfig", tunName, "inet6", ipStr, "prefixlen", "128"},
		}
	} else {
		return fmt.Errorf("unsupported prefix: %v", prefix)
	}

	var lastErr error
	for _, t := range tries {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "ifconfig "+strings.Join(t[1:], " "))
		out, err := exec.Command(t[0], t[1:]...).CombinedOutput()
		if err == nil {
			return waitDevAddrReady(tunName, prefix)
		}
		lastErr = fmt.Errorf("ifconfig failed: %v, output: %s", err, string(out))
	}
	return lastErr
}

// 设置路由表 (macOS)
func SetRouteAddr(addr netip.Prefix, gateway net.IP) error {
	if !addr.IsValid() {
		return fmt.Errorf("invalid route prefix")
	}
	if gateway == nil {
		return fmt.Errorf("gateway is nil")
	}

	var tries [][]string
	if addr.Addr().Is6() {
		gw := gateway.String()
		tries = [][]string{
			{"route", "-n", "add", "-inet6", addr.String(), gw},
			{"route", "-n", "add", "-inet6", "-net", addr.String(), gw},
		}
	} else {
		gw4 := gateway.To4()
		if gw4 == nil {
			return fmt.Errorf("gateway must be ipv4, got: %v", gateway)
		}
		netStr := addr.Addr().String()
		maskStr := prefixMask(addr).String()
		gwStr := gw4.String()
		tries = [][]string{
			{"route", "-n", "add", "-net", netStr, maskStr, gwStr},
			{"route", "-n", "add", "-net", netStr, "-netmask", maskStr, gwStr},
		}
	}

	var lastErr error
	for _, t := range tries {
		out, err := exec.Command(t[0], t[1:]...).CombinedOutput()
		if err == nil {
			return nil
		}

		low := strings.ToLower(string(out))
		// 避免重复添加导致的报错中断；具体返回值是否要忽略由上层决定
		if strings.Contains(low, "file exists") || strings.Contains(low, "exists") {
			return nil
		}
		lastErr = fmt.Errorf("route failed: %v, output: %s", err, string(out))
	}
	return lastErr
}

// CurrentDNS 读取系统当前使用的 DNS。
func CurrentDNS(skipIface string) []string {
	return parseResolvConf("/etc/resolv.conf")
}

// 给当前 DNS 加主机路由，压过网卡上的直连路由。网关用 utun 上的地址，网卡消失时路由一起删除。
func setDevDNS(tunName string) {
	iface, err := net.InterfaceByName(tunName)
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "dns route: "+err.Error())
		return
	}
	addrs, err := iface.Addrs()
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "dns route: "+err.Error())
		return
	}
	var gw4, gw6 net.IP
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if v4 := ipnet.IP.To4(); v4 != nil {
			gw4 = v4
			continue
		}
		if ipnet.IP.IsGlobalUnicast() {
			gw6 = ipnet.IP
		}
	}
	cmd := exec.Command("scutil")
	cmd.Stdin = strings.NewReader("open\nget State:/Network/Global/DNS\nd.show\nclose\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf("dns route: %v, %s", err, out))
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		ip := net.ParseIP(fields[len(fields)-1])
		if ip == nil {
			continue
		}
		var prefix netip.Prefix
		var gw net.IP
		if v4 := ip.To4(); v4 != nil {
			if gw4 == nil || v4.Equal(gw4) {
				continue
			}
			prefix = netip.PrefixFrom(netip.MustParseAddr(v4.String()), 32)
			gw = gw4
		} else {
			if gw6 == nil || ip.Equal(gw6) {
				continue
			}
			addr, ok := netip.AddrFromSlice(ip)
			if !ok {
				continue
			}
			prefix = netip.PrefixFrom(addr, 128)
			gw = gw6
		}
		if err := SetRouteAddr(prefix, gw); err != nil {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf("dns route %s: %v", ip, err))
		}
	}
}
