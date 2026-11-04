//go:build ios

package engine

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"
	"time"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// pktun 是 iOS 的 LinkEndpoint：Swift 推包进来，outbound 泵回调写出。
type packetTunnel struct {
	*channel.Endpoint

	startOnce       sync.Once
	outboundWorkers sync.WaitGroup
	writeBatch      func([]byte)
	cancelOutbound  context.CancelFunc
}

func newPacketTunnel(mtu uint32, writeBatch func([]byte)) *packetTunnel {
	return &packetTunnel{
		Endpoint:   channel.New(4096, mtu, ""),
		writeBatch: writeBatch,
	}
}

func (tunnel *packetTunnel) Close() {
	if tunnel.cancelOutbound != nil {
		tunnel.cancelOutbound()
	}
	tunnel.Endpoint.Close()
}

func (tunnel *packetTunnel) Attach(dispatcher stack.NetworkDispatcher) {
	tunnel.Endpoint.Attach(dispatcher)
	tunnel.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		tunnel.cancelOutbound = cancel
		tunnel.outboundWorkers.Add(1)
		go tunnel.writeOutboundPackets(ctx)
	})
}

func (tunnel *packetTunnel) injectPacket(packet []byte) {
	if len(packet) == 0 || !tunnel.IsAttached() {
		return
	}
	packetBuffer := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(packet),
	})
	switch header.IPVersion(packet) {
	case header.IPv4Version:
		tunnel.InjectInbound(header.IPv4ProtocolNumber, packetBuffer)
	case header.IPv6Version:
		tunnel.InjectInbound(header.IPv6ProtocolNumber, packetBuffer)
	}
	packetBuffer.DecRef()
}

func (tunnel *packetTunnel) writeOutboundPackets(ctx context.Context) {
	defer tunnel.outboundWorkers.Done()
	var packets [][]byte
	flush := func() {
		if len(packets) == 0 {
			return
		}
		var batch bytes.Buffer
		var lengthHeader [4]byte
		for _, packet := range packets {
			binary.BigEndian.PutUint32(lengthHeader[:], uint32(len(packet)))
			batch.Write(lengthHeader[:])
			batch.Write(packet)
		}
		tunnel.writeBatch(batch.Bytes())
		packets = packets[:0]
	}
	for {
		packetBuffer := tunnel.ReadContext(ctx)
		if packetBuffer == nil {
			flush()
			return
		}
		packets = append(packets, copyPacket(packetBuffer))
		if len(packets) >= 32 {
			flush()
			continue
		}
		child, cancel := context.WithTimeout(ctx, 200*time.Microsecond)
		for len(packets) < 32 {
			nextPacket := tunnel.ReadContext(child)
			if nextPacket == nil {
				break
			}
			packets = append(packets, copyPacket(nextPacket))
		}
		cancel()
		flush()
		if ctx.Err() != nil {
			return
		}
	}
}

func copyPacket(packetBuffer *stack.PacketBuffer) []byte {
	buffer := packetBuffer.ToBuffer()
	packet := append([]byte(nil), buffer.Flatten()...)
	buffer.Release()
	packetBuffer.DecRef()
	return packet
}
