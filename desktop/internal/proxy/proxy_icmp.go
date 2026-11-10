package proxy

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"time"

	"desktop/internal/route"
	"desktop/internal/tun"
)

const icmpDirectTimeout = 3 * time.Second

// openDirectICMP uses the physical source address selected by the direct UDP dialer,
// so Windows' default route to the TUN cannot send the raw packet back into itself.
func openDirectICMP(dst netip.Addr) (*net.IPConn, []byte, error) {
	localIP := route.SourceIP(dst)
	if localIP == nil {
		return nil, nil, fmt.Errorf("no physical source address for %s", dst)
	}
	icmpNetwork := "ip4:icmp"
	if dst.Is6() {
		icmpNetwork = "ip6:ipv6-icmp"
	}
	conn, err := net.ListenIP(icmpNetwork, &net.IPAddr{IP: localIP})
	if err != nil {
		return nil, nil, fmt.Errorf("open direct ICMP socket: %w", err)
	}
	return conn, append([]byte(nil), localIP...), nil
}

// handleICMP sends echo requests using the host's normal routing and injects
// the reply back into the host's TUN interface.
func (p *Proxy) handleICMP(dev *tun.Tun, src, dst netip.Addr, packet []byte) {
	if len(packet) < 20 {
		return
	}
	version := packet[0] >> 4
	headerLen := 0
	switch version {
	case 4:
		headerLen = int(packet[0]&0x0f) * 4
		if headerLen < 20 || len(packet) < headerLen+8 || packet[9] != 1 || packet[headerLen] != 8 || packet[headerLen+1] != 0 {
			return
		}
	case 6:
		headerLen = 40
		if len(packet) < headerLen+8 || packet[6] != 58 || packet[headerLen] != 128 || packet[headerLen+1] != 0 {
			return
		}
	default:
		return
	}
	request := append([]byte(nil), packet[headerLen:]...)
	conn, localIP, err := openDirectICMP(dst)
	if err != nil {
		return
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(icmpDirectTimeout)); err != nil {
		return
	}
	if version == 6 {
		setICMPv6Checksum(request, localIP, dst.AsSlice())
	}
	if _, err := conn.WriteTo(request, &net.IPAddr{IP: dst.AsSlice()}); err != nil {
		return
	}
	response := make([]byte, 65535)
	for {
		n, from, err := conn.ReadFrom(response)
		if err != nil {
			return
		}
		addr, ok := from.(*net.IPAddr)
		if !ok || !net.IP(addr.IP).Equal(net.IP(dst.AsSlice())) || n < 8 {
			continue
		}
		if version == 4 && (response[0] != 0 || response[1] != 0) || version == 6 && (response[0] != 129 || response[1] != 0) {
			continue
		}
		if response[4] != request[4] || response[5] != request[5] || response[6] != request[6] || response[7] != request[7] {
			continue
		}
		payload := append([]byte(nil), response[:n]...)
		if version == 4 {
			dev.WritePacket(buildIPv4Packet(dst.AsSlice(), src.AsSlice(), payload))
		} else {
			setICMPv6Checksum(payload, dst.AsSlice(), src.AsSlice())
			dev.WritePacket(buildIPv6Packet(dst.AsSlice(), src.AsSlice(), payload))
		}
		return
	}
}

func buildIPv4Packet(src, dst, payload []byte) []byte {
	packet := make([]byte, 20+len(payload))
	packet[0] = 0x45
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(packet)))
	packet[8], packet[9] = 64, 1
	copy(packet[12:16], src)
	copy(packet[16:20], dst)
	binary.BigEndian.PutUint16(packet[10:12], checksum(packet[:20]))
	copy(packet[20:], payload)
	return packet
}

func buildIPv6Packet(src, dst, payload []byte) []byte {
	packet := make([]byte, 40+len(payload))
	packet[0] = 0x60
	binary.BigEndian.PutUint16(packet[4:6], uint16(len(payload)))
	packet[6], packet[7] = 58, 64
	copy(packet[8:24], src)
	copy(packet[24:40], dst)
	copy(packet[40:], payload)
	return packet
}

func setICMPv6Checksum(message, src, dst []byte) {
	if len(message) < 4 {
		return
	}
	message[2], message[3] = 0, 0
	pseudo := make([]byte, 40+len(message))
	copy(pseudo[:16], src)
	copy(pseudo[16:32], dst)
	binary.BigEndian.PutUint32(pseudo[32:36], uint32(len(message)))
	pseudo[39] = 58
	copy(pseudo[40:], message)
	binary.BigEndian.PutUint16(message[2:4], checksum(pseudo))
}

func checksum(data []byte) uint16 {
	var sum uint32
	for len(data) >= 2 {
		sum += uint32(binary.BigEndian.Uint16(data[:2]))
		data = data[2:]
	}
	if len(data) == 1 {
		sum += uint32(data[0]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + sum>>16
	}
	return ^uint16(sum)
}
