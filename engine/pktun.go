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
type pktun struct {
	*channel.Endpoint

	startOnce sync.Once
	pumps     sync.WaitGroup
	write     func([]byte)
	cancel    context.CancelFunc
}

func newPktun(mtu uint32, write func([]byte)) *pktun {
	return &pktun{
		Endpoint: channel.New(4096, mtu, ""),
		write:    write,
	}
}

func (t *pktun) Close() {
	if t.cancel != nil {
		t.cancel()
	}
	t.Endpoint.Close()
}

func (t *pktun) Attach(dispatcher stack.NetworkDispatcher) {
	t.Endpoint.Attach(dispatcher)
	t.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		t.cancel = cancel
		t.pumps.Add(1)
		go t.outbound(ctx)
	})
}

func (t *pktun) input(payload []byte) {
	if len(payload) == 0 || !t.IsAttached() {
		return
	}
	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(payload),
	})
	switch header.IPVersion(payload) {
	case header.IPv4Version:
		t.InjectInbound(header.IPv4ProtocolNumber, pkt)
	case header.IPv6Version:
		t.InjectInbound(header.IPv6ProtocolNumber, pkt)
	}
	pkt.DecRef()
}

func (t *pktun) outbound(ctx context.Context) {
	defer t.pumps.Done()
	var pkts [][]byte
	flush := func() {
		if len(pkts) == 0 {
			return
		}
		var buf bytes.Buffer
		var hdr [4]byte
		for _, p := range pkts {
			binary.BigEndian.PutUint32(hdr[:], uint32(len(p)))
			buf.Write(hdr[:])
			buf.Write(p)
		}
		t.write(buf.Bytes())
		pkts = pkts[:0]
	}
	for {
		pkt := t.ReadContext(ctx)
		if pkt == nil {
			flush()
			return
		}
		pkts = append(pkts, copyPacket(pkt))
		if len(pkts) >= 32 {
			flush()
			continue
		}
		child, cancel := context.WithTimeout(ctx, 200*time.Microsecond)
		for len(pkts) < 32 {
			nxt := t.ReadContext(child)
			if nxt == nil {
				break
			}
			pkts = append(pkts, copyPacket(nxt))
		}
		cancel()
		flush()
		if ctx.Err() != nil {
			return
		}
	}
}

func copyPacket(pkt *stack.PacketBuffer) []byte {
	buf := pkt.ToBuffer()
	b := append([]byte(nil), buf.Flatten()...)
	buf.Release()
	pkt.DecRef()
	return b
}
