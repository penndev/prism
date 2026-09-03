package transport

import (
	"crypto/tls"
	"net"

	"github.com/penndev/prism/transport/dialer"
)

func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// dialProxy 连到上游代理。回环地址不走自定义 dialer：
// dialer 会把源地址绑到物理出口网卡上，绑了就连不上本机。
func dialProxy(host string) (net.Conn, error) {
	if isLoopback(host) {
		return net.Dial("tcp", host)
	}
	return dialer.TCPDialer.Dial("tcp", host)
}

// tlsConfigFor 按上游地址补全 ServerName，返回一份独立副本。
// 必须在构造 HandleConnect 时算一次，不能放进返回的闭包里：
// conf 被该节点的所有连接共享，在闭包里改它就是并发写，
// 而且这个结果只取决于 host，每条连接算出来都一样。
func tlsConfigFor(host string, conf *tls.Config) *tls.Config {
	if conf == nil {
		conf = &tls.Config{}
	} else {
		conf = conf.Clone()
	}
	if conf.ServerName != "" || conf.InsecureSkipVerify {
		return conf
	}
	domain, _, err := net.SplitHostPort(host)
	if err != nil {
		// host 不带端口时整串就是域名。这里原来是改成 InsecureSkipVerify，
		// 等于静默关掉证书校验。
		domain = host
	}
	conf.ServerName = domain
	return conf
}

type HandleConnect func(conn net.Conn, network string, address string) error
