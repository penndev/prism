//go:build windows

package route

import (
	"fmt"
	"testing"
)

func TestSystemDNSIPv4(t *testing.T) {
	servers := systemDNSIPv4("")
	fmt.Printf("systemDNSIPv4(\"\") => %#v  (len=%d)\n", servers, len(servers))
	for i, s := range servers {
		fmt.Printf("  [%d] %s\n", i, s)
	}
}
