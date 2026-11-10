package proxy

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"desktop/internal/route"
	"desktop/internal/tun"

	"github.com/penndev/prism/fakeip"
)

const (
	icmpDirectTimeout = 3 * time.Second
	maxICMPPacketSize = 65535

	ipv4MinHeaderLen  = 20
	ipv6HeaderLen     = 40
	icmpEchoHeaderLen = 8 // type, code, checksum, id, seq
	defaultTTL        = 64

	protoICMPv4 = 1
	protoICMPv6 = 58

	icmpv4EchoReply   = 0
	icmpv4EchoRequest = 8
	icmpv6EchoRequest = 128
	icmpv6EchoReply   = 129
)

var (
	fake4 = netip.MustParsePrefix(fakeip.FakeIPv4CIDR)
	fake6 = netip.MustParsePrefix(fakeip.FakeIPv6CIDR)
)

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

// handleICMP sends an echo request using the host's normal routing and injects
// the reply back into the host's TUN interface. Failures are silently dropped,
// exactly like a lost ping.
func (p *Proxy) handleICMP(dev *tun.Tun, src, dst netip.Addr, packet []byte) {
	req, ok := parseEchoRequest(packet)
	if !ok {
		return
	}
	target, err := p.resolveICMPTarget(dst)
	if err != nil {
		return
	}
	reply, err := exchangeEcho(target, req)
	if err != nil {
		return
	}
	writeEchoReply(dev, src, dst, reply, req.isV6)
}

// ---------------------------------------------------------------------------
// Request parsing
// ---------------------------------------------------------------------------

// echoRequest is an ICMP(v6) echo request with the IP header stripped.
type echoRequest struct {
	isV6    bool
	message []byte // owned copy of the ICMP message
}

// parseEchoRequest validates an IP packet and extracts its ICMP echo request.
func parseEchoRequest(packet []byte) (echoRequest, bool) {
	if len(packet) < ipv4MinHeaderLen {
		return echoRequest{}, false
	}

	var (
		headerLen   int
		proto       byte
		requestType byte
		isV6        bool
	)
	switch packet[0] >> 4 {
	case 4:
		headerLen = int(packet[0]&0x0f) * 4
		if headerLen < ipv4MinHeaderLen {
			return echoRequest{}, false
		}
		proto, requestType = packet[9], icmpv4EchoRequest
	case 6:
		headerLen = ipv6HeaderLen
		proto, requestType, isV6 = packet[6], icmpv6EchoRequest, true
	default:
		return echoRequest{}, false
	}

	wantProto := byte(protoICMPv4)
	if isV6 {
		wantProto = protoICMPv6
	}
	if len(packet) < headerLen+icmpEchoHeaderLen || proto != wantProto ||
		packet[headerLen] != requestType || packet[headerLen+1] != 0 {
		return echoRequest{}, false
	}

	return echoRequest{
		isV6:    isV6,
		message: append([]byte(nil), packet[headerLen:]...),
	}, true
}

// isReplyTo reports whether msg is the echo reply matching this request
// (same type/code and same identifier + sequence number).
func (r echoRequest) isReplyTo(msg []byte) bool {
	if len(msg) < icmpEchoHeaderLen {
		return false
	}
	wantType := byte(icmpv4EchoReply)
	if r.isV6 {
		wantType = icmpv6EchoReply
	}
	return msg[0] == wantType && msg[1] == 0 && bytes.Equal(msg[4:8], r.message[4:8])
}

// ---------------------------------------------------------------------------
// Target resolution
// ---------------------------------------------------------------------------

// resolveICMPTarget maps the packet's destination to the address we really ping.
// Fake IPs have no real host behind them, so they are redirected to the remote
// proxy server instead.
func (p *Proxy) resolveICMPTarget(dst netip.Addr) (netip.Addr, error) {
	if !fake4.Contains(dst) && !fake6.Contains(dst) {
		return dst, nil
	}
	u := p.remoteURL.Load()
	if u == nil {
		return netip.Addr{}, errors.New("remote URL not set")
	}
	return lookupAddr(u.Hostname(), dst.Is6())
}

// lookupAddr returns an IP of the requested family for host, which may be
// either an IP literal or a domain name.
func lookupAddr(host string, wantV6 bool) (netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.Unmap()
		if addr.Is6() != wantV6 {
			return netip.Addr{}, fmt.Errorf("%s is not of the requested IP family", host)
		}
		return addr, nil
	}

	network := "ip4"
	if wantV6 {
		network = "ip6"
	}
	ctx, cancel := context.WithTimeout(context.Background(), icmpDirectTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, network, host)
	if err != nil {
		return netip.Addr{}, err
	}
	for _, a := range addrs {
		if a = a.Unmap(); a.Is6() == wantV6 {
			return a, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("no %s address for %s", network, host)
}

// ---------------------------------------------------------------------------
// Sending / receiving
// ---------------------------------------------------------------------------

// openDirectICMP uses the physical source address selected by the direct UDP dialer,
// so Windows' default route to the TUN cannot send the raw packet back into itself.
func openDirectICMP(dst netip.Addr) (*net.IPConn, net.IP, error) {
	localIP := route.SourceIP(dst)
	if localIP == nil {
		return nil, nil, fmt.Errorf("no physical source address for %s", dst)
	}
	network := "ip4:icmp"
	if dst.Is6() {
		network = "ip6:ipv6-icmp"
	}
	conn, err := net.ListenIP(network, &net.IPAddr{IP: localIP})
	if err != nil {
		return nil, nil, fmt.Errorf("open direct ICMP socket: %w", err)
	}
	return conn, localIP, nil
}

// exchangeEcho sends req to target over the physical network and waits for the
// matching echo reply, returning the ICMP message (without IP header).
func exchangeEcho(target netip.Addr, req echoRequest) ([]byte, error) {
	conn, localIP, err := openDirectICMP(target)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(icmpDirectTimeout)); err != nil {
		return nil, err
	}

	targetIP := net.IP(target.AsSlice())
	if req.isV6 {
		// The pseudo-header changes with the new source address.
		setICMPv6Checksum(req.message, localIP, targetIP)
	}
	if _, err := conn.WriteTo(req.message, &net.IPAddr{IP: targetIP}); err != nil {
		return nil, err
	}

	buf := make([]byte, maxICMPPacketSize)
	for {
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			return nil, err
		}
		if ipAddr, ok := from.(*net.IPAddr); !ok || !ipAddr.IP.Equal(targetIP) {
			continue
		}
		if req.isReplyTo(buf[:n]) {
			return append([]byte(nil), buf[:n]...), nil
		}
	}
}

// writeEchoReply wraps reply in an IP header (from dst back to src, as if the
// original destination had answered) and writes it into the TUN device.
func writeEchoReply(dev *tun.Tun, src, dst netip.Addr, reply []byte, isV6 bool) {
	from, to := dst.AsSlice(), src.AsSlice()
	if isV6 {
		setICMPv6Checksum(reply, from, to)
		dev.WritePacket(buildIPv6Packet(from, to, reply))
		return
	}
	dev.WritePacket(buildIPv4Packet(from, to, reply))
}

// ---------------------------------------------------------------------------
// Packet building & checksums
// ---------------------------------------------------------------------------

func buildIPv4Packet(src, dst, payload []byte) []byte {
	packet := make([]byte, ipv4MinHeaderLen+len(payload))
	packet[0] = 0x45 // version 4, IHL 5
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(packet)))
	packet[8], packet[9] = defaultTTL, protoICMPv4
	copy(packet[12:16], src)
	copy(packet[16:20], dst)
	binary.BigEndian.PutUint16(packet[10:12], checksum(packet[:ipv4MinHeaderLen]))
	copy(packet[ipv4MinHeaderLen:], payload)
	return packet
}

func buildIPv6Packet(src, dst, payload []byte) []byte {
	packet := make([]byte, ipv6HeaderLen+len(payload))
	packet[0] = 0x60 // version 6
	binary.BigEndian.PutUint16(packet[4:6], uint16(len(payload)))
	packet[6], packet[7] = protoICMPv6, defaultTTL
	copy(packet[8:24], src)
	copy(packet[24:40], dst)
	copy(packet[ipv6HeaderLen:], payload)
	return packet
}

// setICMPv6Checksum recomputes the checksum of an ICMPv6 message in place,
// including the IPv6 pseudo-header (RFC 8200 §8.1).
func setICMPv6Checksum(message, src, dst []byte) {
	if len(message) < 4 {
		return
	}
	message[2], message[3] = 0, 0

	pseudo := make([]byte, ipv6HeaderLen+len(message))
	copy(pseudo[0:16], src)
	copy(pseudo[16:32], dst)
	binary.BigEndian.PutUint32(pseudo[32:36], uint32(len(message)))
	pseudo[39] = protoICMPv6
	copy(pseudo[ipv6HeaderLen:], message)

	binary.BigEndian.PutUint16(message[2:4], checksum(pseudo))
}

// checksum is the RFC 1071 Internet checksum.
func checksum(data []byte) uint16 {
	var sum uint32
	for len(data) >= 2 {
		sum += uint32(binary.BigEndian.Uint16(data))
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
