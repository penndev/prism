package route

import (
	"desktop/internal"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}

func prefixMask(p netip.Prefix) net.IP {
	if !p.IsValid() {
		return nil
	}
	if p.Addr().Is4() {
		return net.IP(net.CIDRMask(p.Bits(), 32))
	}
	return net.IP(net.CIDRMask(p.Bits(), 128))
}

// 给设备设置静态IP
func SetDevAddr(tunName string, prefix netip.Prefix) error {
	if !prefix.IsValid() {
		return fmt.Errorf("invalid prefix")
	}
	tunIP := prefix.Addr().String()
	var args []string
	switch {
	case prefix.Addr().Is4():
		args = []string{
			"interface", "ipv4", "set", "address",
			fmt.Sprintf("name=%s", tunName),
			"source=static",
			fmt.Sprintf("addr=%s", tunIP),
			fmt.Sprintf("mask=%s", prefixMask(prefix).String()),
		}
	case prefix.Addr().Is6():
		// TUN 上 DAD 收不到 NA，Windows 会丢掉刚加上的 IPv6 地址。
		prep := []string{
			"interface", "ipv6", "set", "interface",
			"interface=" + tunName,
			"dadtransmits=0",
			"forwarding=enabled",
		}
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "netsh "+strings.Join(prep, " "))
		cmd := exec.Command("netsh", prep...)
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf("set ipv6 interface failed: %v, %s", err, out))
		}
		args = []string{
			"interface", "ipv6", "add", "address",
			"interface=" + tunName,
			"address=" + tunIP,
		}
	default:
		return fmt.Errorf("unsupported prefix: %v", prefix)
	}
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "netsh "+strings.Join(args, " "))

	cmd := exec.Command("netsh", args...)
	hideConsole(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil && !alreadyExists(string(out)) {
		return fmt.Errorf("netsh failed: %v, %s", err, string(out))
	}

	if err := waitDevAddr(tunName, prefix.Addr()); err != nil {
		return err
	}
	if prefix.Addr().Is4() {
		metric := []string{"interface", "ipv4", "set", "interface", "interface=" + tunName, "metric=1"}
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "netsh "+strings.Join(metric, " "))
		cmd := exec.Command("netsh", metric...)
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf("set ipv4 metric failed: %v, %s", err, out))
		}
	}
	return nil
}

func alreadyExists(out string) bool {
	low := strings.ToLower(out)
	return strings.Contains(low, "exists") || strings.Contains(out, "已存在")
}

func waitDevAddr(tunName string, addr netip.Addr) error {
	want := net.IP(addr.AsSlice())
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			if ifaceHasUnicast(tunName, want) {
				return nil
			}
		case <-timeout:
			return errors.New("set dev static ip timeout")
		}
	}
}

func ifaceHasUnicast(tunName string, want net.IP) bool {
	if want == nil {
		return false
	}
	var size uint32 = 15000
	buf := make([]byte, size)
	for {
		err := windows.GetAdaptersAddresses(
			windows.AF_UNSPEC,
			windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST|windows.GAA_FLAG_SKIP_DNS_SERVER,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])),
			&size,
		)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return false
		}
		buf = make([]byte, size)
	}
	wantName := strings.ToLower(tunName)
	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); aa != nil; aa = aa.Next {
		name := strings.ToLower(windows.UTF16PtrToString(aa.FriendlyName))
		if name != wantName {
			continue
		}
		for ua := aa.FirstUnicastAddress; ua != nil; ua = ua.Next {
			ip := sockaddrIP(ua.Address)
			if ip != nil && ip.Equal(want) {
				return true
			}
		}
	}
	return false
}

func sockaddrIP(sa windows.SocketAddress) net.IP {
	if sa.Sockaddr == nil {
		return nil
	}
	switch sa.Sockaddr.Addr.Family {
	case windows.AF_INET:
		raw := (*windows.RawSockaddrInet4)(unsafe.Pointer(sa.Sockaddr))
		return net.IPv4(raw.Addr[0], raw.Addr[1], raw.Addr[2], raw.Addr[3])
	case windows.AF_INET6:
		raw := (*windows.RawSockaddrInet6)(unsafe.Pointer(sa.Sockaddr))
		ip := make(net.IP, net.IPv6len)
		copy(ip, raw.Addr[:])
		return ip
	default:
		return nil
	}
}

func setDevDNS(tunName string) {
	servers := CurrentDNS(tunName)
	if tunName == "" || len(servers) == 0 {
		return
	}
	args := []string{
		"interface", "ipv4", "set", "dnsservers",
		"name=" + tunName,
		"source=static",
		"address=" + servers[0],
		"register=none",
		"validate=no",
	}
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "netsh "+strings.Join(args, " "))
	cmd := exec.Command("netsh", args...)
	hideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf("set dns failed: %v, %s", err, out))
		return
	}
	for i, s := range servers[1:] {
		args := []string{
			"interface", "ipv4", "add", "dnsservers",
			"name=" + tunName,
			"address=" + s,
			fmt.Sprintf("index=%d", i+2),
			"validate=no",
		}
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "netsh "+strings.Join(args, " "))
		cmd := exec.Command("netsh", args...)
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf("add dns failed: %v, %s", err, out))
		}
	}
}

func systemDNSIPv4(skipIface string) []string {
	var size uint32 = 15000
	buf := make([]byte, size)
	for {
		err := windows.GetAdaptersAddresses(
			windows.AF_INET,
			windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])),
			&size,
		)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			internal.App.Event.Emit(
				internal.AppConfig.LogTypeName_STATUS,
				"GetAdaptersAddresses failed: "+err.Error(),
			)
			return nil
		}
		buf = make([]byte, size)
	}

	skip := strings.ToLower(skipIface)
	seen := map[string]struct{}{}
	var out []string
	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); aa != nil; aa = aa.Next {
		name := strings.ToLower(windows.UTF16PtrToString(aa.FriendlyName))
		if name == skip || strings.Contains(name, "wintun") {
			continue
		}
		if aa.IfType == windows.IF_TYPE_SOFTWARE_LOOPBACK {
			continue
		}
		if aa.OperStatus != windows.IfOperStatusUp {
			continue
		}
		for dns := aa.FirstDnsServerAddress; dns != nil; dns = dns.Next {
			ip := sockaddrIPv4(dns.Address)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
				continue
			}
			s := ip.String()
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// CurrentDNS 返回当前系统正在使用的 IPv4 DNS（跳过 skipIface、回环与未启用网卡）。
func CurrentDNS(skipIface string) []string {
	return systemDNSIPv4(skipIface)
}

// RestoreDNS Windows 上 DNS 绑在 TUN 网卡，设备关闭后自动失效。
func RestoreDNS() {}

func sockaddrIPv4(sa windows.SocketAddress) net.IP {
	if sa.Sockaddr == nil {
		return nil
	}
	if sa.Sockaddr.Addr.Family != windows.AF_INET {
		return nil
	}
	raw := (*windows.RawSockaddrInet4)(unsafe.Pointer(sa.Sockaddr))
	return net.IPv4(raw.Addr[0], raw.Addr[1], raw.Addr[2], raw.Addr[3])
}

// 设置路由表
func SetRouteAddr(addr netip.Prefix, gateway net.IP) error {
	if !addr.IsValid() {
		return fmt.Errorf("invalid route prefix")
	}
	if gateway == nil {
		return fmt.Errorf("gateway is nil")
	}

	if addr.Addr().Is6() {
		args := []string{"-6", "add", addr.String(), gateway.String()}
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "route "+strings.Join(args, " "))
		rc := exec.Command("route", args...)
		hideConsole(rc)
		out, err := rc.CombinedOutput()
		if err != nil && !alreadyExists(string(out)) {
			return fmt.Errorf("route failed: %v, output: %s", err, string(out))
		}
		return nil
	}

	tunName := ""
	if ifis, err := net.Interfaces(); err == nil {
		for _, ifi := range ifis {
			addrs, err := ifi.Addrs()
			if err != nil {
				continue
			}
			for _, a := range addrs {
				ipnet, ok := a.(*net.IPNet)
				if !ok || !ipnet.IP.Equal(gateway) {
					continue
				}
				tunName = ifi.Name
				break
			}
			if tunName != "" {
				break
			}
		}
	}
	if tunName == "" {
		return fmt.Errorf("tun iface not found for gateway %s", gateway)
	}

	args := []string{
		"interface", "ipv4", "add", "route",
		"prefix=" + addr.String(),
		"interface=" + tunName,
		"metric=1",
		"store=active",
	}
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "netsh "+strings.Join(args, " "))
	cmd := exec.Command("netsh", args...)
	hideConsole(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil && !alreadyExists(string(out)) {
		return fmt.Errorf("netsh route failed: %v, output: %s", err, string(out))
	}
	return nil
}
