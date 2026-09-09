package transport

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/penndev/gopkg/util"
)

func httpConnet(conn, remote net.Conn, user, pass, address string) error {
	// 2. 构造 CONNECT 请求对象
	// 使用 http.NewRequest 可以自动处理 Host 和 URL 格式
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: address},
		Host:   address,
		Header: make(http.Header),
	}
	if user != "" || pass != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		req.Header.Set("Proxy-Authorization", "Basic "+auth)
	}
	if err := req.Write(remote); err != nil {
		return err
	}

	br := bufio.NewReader(remote)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
	}
	// 即使是官方库，http.ReadResponse 也会因为 bufio 的机制导致预读
	if n := br.Buffered(); n > 0 {
		peeked, _ := br.Peek(n)
		conn.Write(peeked) // 趁 Pipe 还没开始，先把“陈粮”塞给客户端
	}
	util.Pipe(conn, remote)
	return nil
}

func Http(host, user, pass string) HandleConnect {
	return func(conn net.Conn, network, address string) error {
		if network != "tcp" {
			return localHandle(conn, network, address)
		}
		remote, err := dialProxy(host)
		if err != nil {
			return err
		}
		return httpConnet(conn, remote, user, pass, address)
	}
}

func HttpOverTLS(host, user, pass string, conf *tls.Config) HandleConnect {
	tlsConf := tlsConfigFor(host, conf)
	return func(conn net.Conn, network, address string) error {
		if network != "tcp" {
			return localHandle(conn, network, address)
		}
		dialTCP, err := dialProxy(host)
		if err != nil {
			return err
		}
		remoteTLS := tls.Client(dialTCP, tlsConf)
		if err := remoteTLS.Handshake(); err != nil {
			remoteTLS.Close()
			return err
		}
		return httpConnet(conn, remoteTLS, user, pass, address)
	}
}
