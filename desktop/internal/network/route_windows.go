package network

import (
	"context"
	"desktop/internal/route"
	"math/bits"
	"net"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const ipUnicastIf = 31 // IP_UNICAST_IF 与 IPV6_UNICAST_IF 的值相同

func dialOutbound(network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return boundDial.Dial(network, address)
	}
	host, _, _ = strings.Cut(host, "%")
	if ip := net.ParseIP(host); ip != nil {
		return dialBound(context.Background(), network, address, ip)
	}
	return dialName(network, host, port)
}

// dialName 解析域名后，对每个有出口的地址族各取一个地址并发连接，先成功的胜出。
func dialName(network, host, port string) (net.Conn, error) {
	resolveCtx, cancelResolve := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelResolve()
	ips, err := boundDial.Resolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		return nil, err
	}

	state := route.Load()
	var picked [2]net.IP // [0] 是 v4，[1] 是 v6
	for _, item := range ips {
		fam := 0
		if item.IP.To4() == nil {
			fam = 1
		}
		if picked[fam] == nil && state.Iface(fam == 1) != nil {
			picked[fam] = item.IP
		}
	}
	var targets []net.IP
	for _, ip := range []net.IP{picked[1], picked[0]} { // v6 优先
		if ip != nil {
			targets = append(targets, ip)
		}
	}

	// 有一个连上就取消其余的；已经连上的落选连接在下面关掉。
	dialCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, len(targets))
	for _, ip := range targets {
		go func() {
			conn, err := dialBound(dialCtx, network, net.JoinHostPort(ip.String(), port), ip)
			ch <- result{conn, err}
		}()
	}

	dialErr := errUnreachable(network, net.JoinHostPort(host, port))
	for i := range targets {
		r := <-ch
		if r.err != nil {
			dialErr = r.err
			continue
		}
		go func(n int) {
			for ; n > 0; n-- {
				if r := <-ch; r.conn != nil {
					r.conn.Close()
				}
			}
		}(len(targets) - i - 1)
		return r.conn, nil
	}
	return nil, dialErr
}

func bindSocket(ipv6 bool, index int, raw syscall.RawConn) error {
	level, value := windows.IPPROTO_IP, int(bits.ReverseBytes32(uint32(index))) // IPv4 要网络字节序
	if ipv6 {
		level, value = windows.IPPROTO_IPV6, index
	}
	var sockErr error
	err := raw.Control(func(fd uintptr) {
		sockErr = windows.SetsockoptInt(windows.Handle(fd), level, ipUnicastIf, value)
	})
	if err != nil {
		return err
	}
	return sockErr
}
