package transport

import (
	"crypto/tls"
	"net"

	"github.com/penndev/gopkg/socks5"
	"github.com/penndev/gopkg/util"
)

// relaySocks5 在已建立的上游连接上完成 socks5 协商并转发。
// dialTcp 的所有权交给本函数：失败时关掉，成功时由 socks.Close() 收尾。
func relaySocks5(conn, dialTcp net.Conn, user, pass, network, address string) error {
	socks := &socks5.Client{
		Username: user,
		Password: pass,
		Conn:     dialTcp,
	}
	if err := socks.Negotiation(); err != nil {
		dialTcp.Close()
		return err
	}
	remote, err := socks.Dial(network, address)
	if err != nil {
		dialTcp.Close()
		return err
	}
	// UDP 关联时 Dial 返回的是新建的 UDP 连接，dialTcp 是必须一直保持的控制连接，
	// util.Pipe 不会碰它，只有 socks.Close() 会把两者一起关掉。
	defer socks.Close()
	util.Pipe(conn, remote)
	return nil
}

// socks5标准请求
func Socks5(host, user, pass string) HandleConnect {
	return func(conn net.Conn, network, address string) error {
		dialTcp, err := dialProxy(host)
		if err != nil {
			return err
		}
		return relaySocks5(conn, dialTcp, user, pass, network, address)
	}
}

// socks5 tls
func Socks5OverTLS(host, user, pass string, conf *tls.Config) HandleConnect {
	tlsConf := tlsConfigFor(host, conf)
	return func(conn net.Conn, network, address string) error {
		dialTcp, err := dialProxy(host)
		if err != nil {
			return err
		}
		dialTls := tls.Client(dialTcp, tlsConf)
		if err := dialTls.Handshake(); err != nil {
			dialTls.Close()
			return err
		}
		return relaySocks5(conn, dialTls, user, pass, network, address)
	}
}
