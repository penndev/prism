package route

import (
	"errors"
	"fmt"
	"log"
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
	if !prefix.Addr().Is4() {
		panic("not ipv4 prefix")
	}
	tunIP := prefix.Addr().String()
	args := []string{
		"interface", "ipv4", "set", "address",
		fmt.Sprintf(`name=%s`, tunName),
		"source=static",
		fmt.Sprintf("addr=%s", tunIP),
		fmt.Sprintf("mask=%s", prefixMask(prefix).String()),
	}
	log.Println("netsh", strings.Join(args, " "))

	cmd := exec.Command("netsh", args...)
	hideConsole(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh failed: %v, %s", err, string(out))
	}

	waitRouteReady := func() error {
		ticker := time.NewTicker(200 * time.Millisecond)
		timeout := time.After(30 * time.Second)
		for {
			select {
			case <-ticker.C:
				ic := exec.Command("ipconfig")
				hideConsole(ic)
				out, err := ic.CombinedOutput()
				if err != nil {
					continue
				}
				if strings.Contains(string(out), tunIP) {
					return nil
				} else {
					continue
				}
			case <-timeout:
				return errors.New("set dev static ip timeout")
			}
		}
	}
	if err := waitRouteReady(); err != nil {
		return err
	}
	setDevDNS(tunName)
	return nil
}

func setDevDNS(tunName string) {
	servers := systemDNSIPv4(tunName)
	if len(servers) == 0 {
		return
	}

	args := []string{
		"interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", tunName),
		"static", servers[0],
		"validate=no",
	}
	log.Println("netsh", strings.Join(args, " "))
	cmd := exec.Command("netsh", args...)
	hideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("set dns failed: %v, %s", err, string(out))
		return
	}

	for i, addr := range servers[1:] {
		args := []string{
			"interface", "ipv4", "add", "dnsservers",
			fmt.Sprintf("name=%s", tunName),
			"address=" + addr,
			fmt.Sprintf("index=%d", i+2),
			"validate=no",
		}
		log.Println("netsh", strings.Join(args, " "))
		cmd := exec.Command("netsh", args...)
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("add dns failed: %v, %s", err, string(out))
			return
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
			log.Println("GetAdaptersAddresses:", err)
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

	args := []string{
		"add",
		addr.Addr().String(),
		"mask",
		prefixMask(addr).String(),
		gateway.String(),
	}

	log.Println("route", strings.Join(args, " "))

	rc := exec.Command("route", args...)
	hideConsole(rc)
	out, err := rc.CombinedOutput()
	if err != nil {
		return fmt.Errorf("route failed: %v, output: %s", err, string(out))
	}
	return nil
}
