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

var (
	fakeIPv4Network = mustParseCIDR(fakeip.FakeIPv4CIDR)
	fakeIPv6Network = mustParseCIDR(fakeip.FakeIPv6CIDR)
)

func mustParseCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return network
}

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

const (
	udpIdleTimeout = 60 * time.Second
	dnsTimeout     = 5 * time.Second
)

// relayDNS 读 UDP 上的 DNS 查询，交给 fakeip 回答，并把查询结果记到 DNS 日志。
func relayDNS(hook Hook, conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 4096)
	_ = conn.SetDeadline(time.Now().Add(dnsTimeout))
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			return
		}
		response, err := fakeip.Answer(buffer[:n])
		request := new(dns.Msg)
		if unpackErr := request.Unpack(buffer[:n]); unpackErr == nil && len(request.Question) > 0 {
			question := request.Question[0]
			name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(question.Name), "."))
			qtype := dns.Type(question.Qtype).String()
			result := ""
			if err != nil {
				result = err.Error()
			} else {
				answer := new(dns.Msg)
				if unpackErr := answer.Unpack(response); unpackErr != nil {
					result = "invalid"
				} else {
					var records []string
					for _, record := range answer.Answer {
						switch value := record.(type) {
						case *dns.A:
							records = append(records, value.A.String())
						case *dns.AAAA:
							records = append(records, value.AAAA.String())
						case *dns.CNAME:
							if target := strings.TrimSuffix(value.Target, "."); target != "" {
								records = append(records, target)
							}
						default:
							if header := record.Header(); header != nil && header.Rrtype != 0 {
								records = append(records, dns.Type(header.Rrtype).String())
							}
						}
					}
					result = strings.Join(records, ",")
					if result == "" && answer.Rcode != dns.RcodeSuccess {
						result = dns.RcodeToString[answer.Rcode]
					}
					if result == "" {
						result = "Null"
					}
				}
			}
			hook.OnDNS(name, qtype, result)
		}
		if err != nil || len(response) == 0 {
			continue
		}
		if _, err := conn.Write(response); err != nil {
			return
		}
		_ = conn.SetDeadline(time.Now().Add(dnsTimeout))
	}
}

// Relay 处理一条连接：DNS 交给 fakeip，假 IP 换回域名走代理，其余问 h.UseProxy。
// h 一定非 nil。失败时关掉 conn 并返回错误；成功路径不关 conn。
func Relay(proxyConnect, directConnect transport.HandleConnect, hook Hook, conn net.Conn, network, address string) error {
	defer func() {
		if r := recover(); r != nil {
			hook.OnStatus(fmt.Sprintf("relay panic %s %s: %v", network, address, r))
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
			relayDNS(hook, conn)
			return nil
		}
		// 目标是之前发出去的假 IP，换回真实域名再走代理。
		if ip := net.ParseIP(host); ip != nil && (fakeIPv4Network.Contains(ip) || fakeIPv6Network.Contains(ip)) {
			if domain, ok := fakeip.Lookup(ip); ok {
				address = net.JoinHostPort(domain, port)
				useProxy = true
				hook.OnConnection(network, address, "")
			}
		}
	}
	if !useProxy {
		useProxy = hook.UseProxy(network, address)
	}
	// UDP 没有 FIN，不设空闲超时的话 Pipe 两头的 Read 永远不返回，连接和缓冲一直挂着。
	if network == "udp" || network == "udp4" || network == "udp6" {
		conn = &idleTimeoutConn{Conn: conn}
	}

	// 直连不计入代理流量，所以只给代理路径包 WrapConn。
	connect := directConnect
	if useProxy {
		connect = proxyConnect
		conn = pkg.WrapConn(conn, hook.OnProxyRead, hook.OnProxyWrite)
	}
	if err := connect(conn, network, address); err != nil {
		hook.OnConnection(network, address, err.Error())
		_ = conn.Close()
		return err
	}
	return nil
}

// idleConn 在两个方向有数据时都把读超时往后推，空闲到期后 Read 返回超时错误。
type idleTimeoutConn struct {
	net.Conn
}

func (conn *idleTimeoutConn) Read(buffer []byte) (int, error) {
	_ = conn.Conn.SetReadDeadline(time.Now().Add(udpIdleTimeout))
	return conn.Conn.Read(buffer)
}

func (conn *idleTimeoutConn) Write(buffer []byte) (int, error) {
	_ = conn.Conn.SetReadDeadline(time.Now().Add(udpIdleTimeout))
	return conn.Conn.Write(buffer)
}
