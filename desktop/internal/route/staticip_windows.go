package route

import (
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

func runHidden(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return cmd.CombinedOutput()
}

func alreadyExists(out string) bool {
	return strings.Contains(strings.ToLower(out), "exists") || strings.Contains(out, "已存在")
}

// fn 返回 false 时停下。buf 在 fn 返回前一直有效。
func walkAdapters(family, flags uint32, fn func(*windows.IpAdapterAddresses) bool) error {
	size := uint32(15000)
	buf := make([]byte, size)
	for {
		err := windows.GetAdaptersAddresses(
			family,
			flags,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])),
			&size,
		)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return err
		}
		buf = make([]byte, size)
	}
	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); aa != nil; aa = aa.Next {
		if !fn(aa) {
			return nil
		}
	}
	return nil
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

func SetDevAddr(tunName string, prefix netip.Prefix) error {
	ip := prefix.Addr().String()
	var args []string
	switch {
	case prefix.Addr().Is4():
		args = []string{
			"interface", "ipv4", "set", "address",
			"name=" + tunName,
			"source=static",
			"addr=" + ip,
			"mask=" + prefixMask(prefix).String(),
		}
	case prefix.Addr().Is6():
		// TUN 上 DAD 收不到 NA，Windows 会丢掉刚加上的 IPv6 地址。
		out, err := runHidden(
			"netsh",
			"interface", "ipv6", "set", "interface",
			"interface="+tunName,
			"dadtransmits=0",
			"forwarding=enabled",
		)
		if err != nil {
			status("set ipv6 interface failed: %v, %s", err, out)
		}
		args = []string{
			"interface", "ipv6", "add", "address",
			"interface=" + tunName,
			"address=" + ip,
		}
	}

	out, err := runHidden("netsh", args...)
	if err != nil && !alreadyExists(string(out)) {
		return fmt.Errorf("netsh failed: %v, %s", err, out)
	}
	if err := waitDevAddr(tunName, prefix.Addr()); err != nil {
		return err
	}
	if !prefix.Addr().Is4() {
		return nil
	}
	out, err = runHidden(
		"netsh",
		"interface", "ipv4", "set", "interface",
		"interface="+tunName,
		"metric=1",
	)
	if err != nil {
		status("set ipv4 metric failed: %v, %s", err, out)
	}
	return nil
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
	wantName := strings.ToLower(tunName)
	found := false
	_ = walkAdapters(
		windows.AF_UNSPEC,
		windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST|windows.GAA_FLAG_SKIP_DNS_SERVER,
		func(aa *windows.IpAdapterAddresses) bool {
			name := strings.ToLower(windows.UTF16PtrToString(aa.FriendlyName))
			if name != wantName {
				return true
			}
			for ua := aa.FirstUnicastAddress; ua != nil; ua = ua.Next {
				ip := sockaddrIP(ua.Address)
				if ip != nil && ip.Equal(want) {
					found = true
					return false
				}
			}
			return true
		},
	)
	return found
}

func setDevDNS(tunName string) {
	out, err := runHidden(
		"netsh",
		"interface", "ipv4", "set", "dnsservers",
		"name="+tunName,
		"source=static",
		"address="+tunDNS,
		"register=none",
		"validate=no",
	)
	if err != nil {
		status("set dns failed: %v, %s", err, out)
	}
}

// DNS 写在 TUN 网卡上，网卡关掉就没了。
func RestoreDNS() {}

// CurrentDNS 返回当前系统正在使用的 IPv4 DNS（跳过 skipIface、回环与未启用网卡）。
func CurrentDNS(skipIface string) []string {
	skip := strings.ToLower(skipIface)
	seen := map[string]struct{}{}
	var out []string
	err := walkAdapters(
		windows.AF_INET,
		windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST,
		func(aa *windows.IpAdapterAddresses) bool {
			name := strings.ToLower(windows.UTF16PtrToString(aa.FriendlyName))
			if name == skip || strings.Contains(name, "wintun") {
				return true
			}
			if aa.IfType == windows.IF_TYPE_SOFTWARE_LOOPBACK || aa.OperStatus != windows.IfOperStatusUp {
				return true
			}
			for dns := aa.FirstDnsServerAddress; dns != nil; dns = dns.Next {
				ip := sockaddrIP(dns.Address)
				if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
					continue
				}
				s := ip.String()
				if _, ok := seen[s]; ok {
					continue
				}
				seen[s] = struct{}{}
				out = append(out, s)
			}
			return true
		},
	)
	if err != nil {
		status("GetAdaptersAddresses failed: %s", err)
		return nil
	}
	return out
}

func SetRouteAddr(prefix netip.Prefix, gateway net.IP) error {
	if prefix.Addr().Is6() {
		out, err := runHidden("route", "-6", "add", prefix.String(), gateway.String())
		if err != nil && !alreadyExists(string(out)) {
			return fmt.Errorf("route failed: %v, output: %s", err, out)
		}
		return nil
	}

	tunName := ""
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, ifi := range ifaces {
			addrs, err := ifi.Addrs()
			if err != nil {
				continue
			}
			for _, a := range addrs {
				ipnet, ok := a.(*net.IPNet)
				if ok && ipnet.IP.Equal(gateway) {
					tunName = ifi.Name
				}
			}
			if tunName != "" {
				break
			}
		}
	}
	if tunName == "" {
		return fmt.Errorf("tun iface not found for gateway %s", gateway)
	}

	out, err := runHidden(
		"netsh",
		"interface", "ipv4", "add", "route",
		"prefix="+prefix.String(),
		"interface="+tunName,
		"metric=1",
		"store=active",
	)
	if err != nil && !alreadyExists(string(out)) {
		return fmt.Errorf("netsh route failed: %v, output: %s", err, out)
	}
	return nil
}
