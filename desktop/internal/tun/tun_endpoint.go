package tun

import (
	"context"
	"log"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

func (t *Tun) read(packet []byte) (int, error) {
	t.devRM.Lock()
	defer t.devRM.Unlock()
	bufs := make([][]byte, 1)
	bufs[0] = packet
	sizes := make([]int, 1)
	_, err := (*t.dev).Read(bufs, sizes, t.offset)
	return sizes[0], err
}

func (t *Tun) write(packet []byte) (int, error) {
	t.devWM.Lock()
	defer t.devWM.Unlock()
	bufs := make([][]byte, 1)
	bufs[0] = packet
	return (*t.dev).Write(bufs, t.offset)
}

// 从tun读取数据包，并注入到gvisor中
func (t *Tun) inbound(cancel context.CancelFunc) {
	defer t.Done()
	defer cancel()

	for {
		data := make([]byte, int(t.mtu)+t.offset)
		n, err := t.read(data)
		if err != nil {
			log.Println("read error:", err)
			break
		}
		if n == 0 || n > int(t.mtu) || !t.IsAttached() {
			continue
		}
		// sizes[0] 是 IP 载荷长度，不是缓冲区结束下标；须从 offset 起取 n 字节。
		payload := data[t.offset : t.offset+n]
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
}

// 读取gvisor中数据包，并写入tun设备
func (t *Tun) outbound(ctx context.Context) {
	defer t.Done()
	for {
		pkt := t.ReadContext(ctx)
		if pkt == nil {
			break
		}
		buf := pkt.ToBuffer()

		if t.offset != 0 {
			v := buffer.NewViewWithData(make([]byte, t.offset))
			_ = buf.Prepend(v)
		}

		_, err := t.write(buf.Flatten())
		if err != nil {
			log.Println("write error:", err)
			// break
		}
		buf.Release()
		pkt.DecRef()
	}
}

// 协议栈启动
func (t *Tun) Attach(dispatcher stack.NetworkDispatcher) {
	t.Endpoint.Attach(dispatcher)
	t.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		t.Add(2)
		go t.outbound(ctx)
		go t.inbound(cancel)
	})
}
