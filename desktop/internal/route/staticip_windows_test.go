//go:build windows

package route

import (
	"fmt"
	"testing"
)

func TestCurrentDNS(t *testing.T) {
	servers := CurrentDNS("")
	fmt.Printf("CurrentDNS(\"\") => %#v  (len=%d)\n", servers, len(servers))
	for i, s := range servers {
		fmt.Printf("  [%d] %s\n", i, s)
	}
}
