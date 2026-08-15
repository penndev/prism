//go:build darwin
// +build darwin

package route

import (
	"errors"
	"fmt"
	"log"
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
	if !prefix.IsValid() || !prefix.Addr().Is4() {
		return fmt.Errorf("only ipv4 prefix is supported: %v", prefix)
	}

	ipStr := prefix.Addr().String()
	maskStr := prefixMask(prefix).String()

	// utun 是点到点接口，ifconfig 通常需要提供 dstaddr（可以用同一个 ip 作为兜底）
	tries := [][]string{
		// ifconfig utunX inet <addr> <dstaddr> netmask <mask> up
		{"ifconfig", tunName, "inet", ipStr, ipStr, "netmask", maskStr, "up"},
		// ifconfig utunX inet <addr> netmask <mask> up（不同系统/场景可能能直接工作）
		{"ifconfig", tunName, "inet", ipStr, "netmask", maskStr, "up"},
	}

	var lastErr error
	for _, t := range tries {
		log.Println("ifconfig", strings.Join(t[1:], " "))
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

	gw4 := gateway.To4()
	if gw4 == nil {
		return fmt.Errorf("gateway must be ipv4, got: %v", gateway)
	}

	netStr := addr.Addr().String()
	maskStr := prefixMask(addr).String()
	gwStr := gw4.String()

	// 尽量覆盖 macOS / BSD route 的不同参数风格
	tries := [][]string{
		{"route", "-n", "add", "-net", netStr, maskStr, gwStr},
		{"route", "-n", "add", "-net", netStr, "-netmask", maskStr, gwStr},
	}

	var lastErr error
	for _, t := range tries {
		log.Println("route", strings.Join(t[1:], " "))
		out, err := exec.Command(t[0], t[1:]...).CombinedOutput()
		if err == nil {
			if len(out) > 0 {
				log.Println("route output:", string(out))
			}
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
