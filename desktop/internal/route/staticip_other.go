//go:build !windows && !darwin
// +build !windows,!darwin

package route

import (
	"fmt"
	"net"
	"net/netip"
	"runtime"

	"github.com/miekg/dns"
)

func SetDevAddr(tunName string, prefix netip.Prefix) error {
	return fmt.Errorf("static ip is not supported on %s", runtime.GOOS)
}

func SetRouteAddr(addr netip.Prefix, gateway net.IP) error {
	return fmt.Errorf("static route is not supported on %s", runtime.GOOS)
}

func CurrentDNS(string) []string {
	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	return conf.Servers
}

func RestoreDNS() {}

func setDevDNS(string) {}
