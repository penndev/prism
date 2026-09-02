package fakeip

import (
	"encoding/binary"
	"fmt"
	"net"
)

const defaultNet = "198.18.0.0/15"

type pool struct {
	next  uint32
	start uint32
	end   uint32
}

func newPool(cidr string) (*pool, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("fake net: %w", err)
	}
	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("fake net must be IPv4 CIDR")
	}
	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones > 30 {
		return nil, fmt.Errorf("fake net too small: %s", cidr)
	}
	base := binary.BigEndian.Uint32(ip4)
	size := uint32(1) << uint(32-ones)
	return &pool{
		start: base + 1,
		end:   base + size - 1,
		next:  base + 1,
	}, nil
}

func (p *pool) nextIP() net.IP {
	n := p.end - p.start
	for i := uint32(0); i < n; i++ {
		v := p.next
		p.next++
		if p.next >= p.end {
			p.next = p.start
		}
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, v)
		if last := ip[3]; last != 0 && last != 255 {
			return ip
		}
	}
	return nil
}
