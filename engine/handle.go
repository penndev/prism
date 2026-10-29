package engine

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/pkg"
	"github.com/penndev/prism/transport"
)

// Hook 是单条连接的业务回调。Android / iOS 的 Handler 已经包含这些方法。
// 日志只传字段，怎么排版由客户端决定。
type Hook interface {
	OnStatus(line string)
	OnConnection(network, address, err string)
	OnDNS(name, qtype, result string)
	UseProxy(network, address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
}

// Relay 处理一条连接：DNS 交给 fakeip，假 IP 换回域名走代理，其余问 h.UseProxy。
// h 一定非 nil。失败时关掉 conn 并返回错误；成功路径不关 conn。
func Relay(proxy, local transport.HandleConnect, h Hook, conn net.Conn, network, address string) error {
	defer func() {
		if r := recover(); r != nil {
			h.OnStatus(fmt.Sprintf("relay panic %s %s: %v", network, address, r))
		}
	}()
	if conn == nil {
		return nil
	}

	useProxy := false
	if host, port, err := net.SplitHostPort(address); err == nil {
		// DNS 一律本地劫持，由 fakeip 决定返回假 IP 还是转发给上游。
		// 读结束时的 EOF 是正常收尾，不往外传。
		if port == "53" && (network == "udp" || network == "udp4" || network == "udp6") {
			relayDNS(h, conn)
			return nil
		}
		// 目标是之前发出去的假 IP，换回真实域名再走代理。
		if domain, ok := fakeip.Lookup(host); ok {
			address = net.JoinHostPort(domain, port)
			useProxy = true
			h.OnConnection(network, address, "")
		}
	}
	if !useProxy {
		useProxy = h.UseProxy(network, address)
	}
	// UDP 没有 FIN，不设空闲超时的话 Pipe 两头的 Read 永远不返回，连接和缓冲一直挂着。
	if network == "udp" || network == "udp4" || network == "udp6" {
		conn = &idleConn{Conn: conn}
	}

	// 直连不计入代理流量，所以只给代理路径包 WrapConn。
	connect := local
	if useProxy {
		connect = proxy
		conn = pkg.WrapConn(conn, h.OnProxyRead, h.OnProxyWrite)
	}
	if err := connect(conn, network, address); err != nil {
		h.OnConnection(network, address, err.Error())
		_ = conn.Close()
		return err
	}
	return nil
}

const (
	udpIdleTimeout = 60 * time.Second
	dnsTimeout     = 5 * time.Second
)

// relayDNS 读 UDP 上的 DNS 查询，交给 fakeip 回答，并把查询结果记到 DNS 日志。
func relayDNS(h Hook, conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	_ = conn.SetDeadline(time.Now().Add(dnsTimeout))
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		out, err := fakeip.Answer(buf[:n])
		dnsQueryLog(h, buf[:n], out, err)
		if err != nil || len(out) == 0 {
			continue
		}
		if _, err := conn.Write(out); err != nil {
			return
		}
		_ = conn.SetDeadline(time.Now().Add(dnsTimeout))
	}
}

func dnsQueryLog(h Hook, query, response []byte, fail error) {
	req := new(dns.Msg)
	if err := req.Unpack(query); err != nil || len(req.Question) == 0 {
		return
	}
	q := req.Question[0]
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(q.Name), "."))
	qtype := dns.Type(q.Qtype).String()
	if fail != nil {
		h.OnDNS(name, qtype, fail.Error())
		return
	}
	var parts []string
	msg := new(dns.Msg)
	if err := msg.Unpack(response); err == nil {
		for _, rr := range msg.Answer {
			switch v := rr.(type) {
			case *dns.A:
				parts = append(parts, v.A.String())
			case *dns.AAAA:
				parts = append(parts, v.AAAA.String())
			case *dns.CNAME:
				parts = append(parts, strings.TrimSuffix(v.Target, "."))
			}
		}
	}
	h.OnDNS(name, qtype, strings.Join(parts, ","))
}

// idleConn 在两个方向有数据时都把读超时往后推，空闲到期后 Read 返回超时错误。
type idleConn struct {
	net.Conn
}

func (c *idleConn) Read(b []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(udpIdleTimeout))
	return c.Conn.Read(b)
}

func (c *idleConn) Write(b []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(udpIdleTimeout))
	return c.Conn.Write(b)
}
