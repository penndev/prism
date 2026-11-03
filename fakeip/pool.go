package fakeip

import (
	"encoding/binary"
	"fmt"
	"net"
)

// addressPool allocates synthetic addresses from a configured IPv4 or IPv6 range.
type addressPool struct {
	nextIPv4   uint32
	firstIPv4  uint32
	limitIPv4  uint32
	prefixIPv6 [16]byte
	nextIPv6   uint64
	ipv6       bool
}

func newIPv4Pool(cidr string) (*addressPool, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("fake IPv4 range: %w", err)
	}
	ipv4 := network.IP.To4()
	if ipv4 == nil {
		return nil, fmt.Errorf("fake IPv4 range must be an IPv4 CIDR")
	}
	prefixBits, addressBits := network.Mask.Size()
	if addressBits != 32 || prefixBits > 30 {
		return nil, fmt.Errorf("fake IPv4 range is too small: %s", cidr)
	}
	first := binary.BigEndian.Uint32(ipv4) + 1
	limit := binary.BigEndian.Uint32(ipv4) + (uint32(1) << uint(32-prefixBits)) - 1
	return &addressPool{nextIPv4: first, firstIPv4: first, limitIPv4: limit}, nil
}

func newIPv6Pool(cidr string) (*addressPool, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil || network == nil || network.IP.To16() == nil {
		return nil, fmt.Errorf("invalid fake IPv6 range: %s", cidr)
	}
	prefixBits, addressBits := network.Mask.Size()
	if addressBits != 128 || prefixBits != 64 {
		return nil, fmt.Errorf("fake IPv6 range must be a /64: %s", cidr)
	}
	var prefix [16]byte
	copy(prefix[:], network.IP.To16())
	return &addressPool{prefixIPv6: prefix, nextIPv6: 1, ipv6: true}, nil
}

func (pool *addressPool) nextIP() net.IP {
	if pool.ipv6 {
		address := make(net.IP, net.IPv6len)
		copy(address, pool.prefixIPv6[:])
		binary.BigEndian.PutUint64(address[8:], pool.nextIPv6)
		pool.nextIPv6++
		if pool.nextIPv6 == 0 {
			pool.nextIPv6 = 1
		}
		return address
	}

	available := pool.limitIPv4 - pool.firstIPv4
	for checked := uint32(0); checked < available; checked++ {
		value := pool.nextIPv4
		pool.nextIPv4++
		if pool.nextIPv4 >= pool.limitIPv4 {
			pool.nextIPv4 = pool.firstIPv4
		}
		address := make(net.IP, net.IPv4len)
		binary.BigEndian.PutUint32(address, value)
		if address[3] != 0 && address[3] != 255 {
			return address
		}
	}
	return nil
}
