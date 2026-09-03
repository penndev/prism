package tun

import (
	"sync"

	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
)

type Tun struct {
	*channel.Endpoint

	// 用命名字段而不是匿名嵌入：否则 Do / Add / Done 会变成 Tun 的公开方法，
	// 而 Tun 是要当作 stack.LinkEndpoint 传出去的。
	startOnce sync.Once
	pumps     sync.WaitGroup

	mtu    uint32
	offset int // unxi设备会有这个 Packet Information (PI)
	dev    tun.Device
	devRM  sync.Mutex
	devWM  sync.Mutex
}

func (t *Tun) Name() string {
	name, _ := t.dev.Name()
	return name
}

func (t *Tun) Close() {
	t.dev.Close()
	t.Endpoint.Close()
}

func (t *Tun) Wait() {
	t.pumps.Wait()
}

// return stack.LinkEndpoint interface
func New(options Options) (*Tun, error) {
	dev, err := tun.CreateTUN(options.Name, options.MTU)
	if err != nil {
		return nil, err
	}
	mtu, err := dev.MTU()
	if err != nil {
		// 设备已经建出来了，这里不关会在 Windows 上留下孤儿 wintun 适配器。
		_ = dev.Close()
		return nil, err
	}

	return &Tun{
		mtu:      uint32(mtu),
		offset:   options.Offset,
		dev:      dev,
		Endpoint: channel.New(1024, uint32(mtu), ""),
	}, nil
}
