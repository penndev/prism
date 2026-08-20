package proxy

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/penndev/gopkg/util"
)

func (s *Server) verifyHTTPProxyAuth(req *http.Request) bool {
	// 未配置用户名密码时，不做鉴权。
	if s.Username == "" && s.Password == "" {
		return true
	}
	auth := req.Header.Get("Proxy-Authorization")
	if auth == "" {
		return false
	}
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(auth, prefix)))
	if err != nil {
		return false
	}
	userpass := string(raw)
	i := strings.IndexByte(userpass, ':')
	if i < 0 {
		return false
	}
	return userpass[:i] == s.Username && userpass[i+1:] == s.Password
}

// handleHTTPConnect 处理 CONNECT（常见于 HTTPS），建立双向隧道。
// client 必须为已 Hijack 的连接，否则 net/http 在 handler 返回后可能继续写回包，破坏 TLS。
func (s *Server) handleHTTPConnect(client net.Conn, req *http.Request) error {
	host := req.Host
	if host == "" {
		fmt.Fprintf(client, "HTTP/1.1 400 Bad Request\r\n\r\n")
		return fmt.Errorf("missing host")
	}
	fmt.Fprintf(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
	s.HandleConnect(client, "tcp", host)
	return nil
}

// handleHTTPProxyForward 处理带绝对 URL 的 HTTP 代理请求（如 GET http://host/path）。
func (s *Server) handleHTTPProxyForward(client net.Conn, req *http.Request) error {
	port := req.URL.Port()
	if port == "" {
		port = "80"
	}
	addr := net.JoinHostPort(req.URL.Hostname(), port)

	remote, local := net.Pipe()
	go func() {
		err := s.HandleConnect(remote, "tcp", addr)
		if err != nil {
			// log.Println("handleHTTPProxyForward", err)
		}
	}()

	// 复写请求
	cc := req.Clone(req.Context())
	cc.RequestURI = ""
	cc.URL = &url.URL{
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	if cc.URL.Path == "" {
		cc.URL.Path = "/"
	}
	cc.Host = req.URL.Host
	if cc.URL.Path == "" {
		cc.URL.Path = "/"
	}
	for _, key := range HttpProxyHeaders {
		cc.Header.Del(key)
	}
	if err := cc.Write(local); err != nil {
		return err
	}
	util.Pipe(client, local)
	return nil
}

func (s *Server) proxyHTTP(conn net.Conn) {
	listener := &HttpSingleConnListener{conn: conn}
	http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.verifyHTTPProxyAuth(r) {
			// http tcp代理
			if r.Method == http.MethodConnect {
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					http.Error(w, "hijacking not supported", http.StatusInternalServerError)
					return
				}
				client, _, err := hijacker.Hijack()
				if err != nil {
					log.Println("hijack failed: ", err)
					return
				}
				if err := s.handleHTTPConnect(client, r); err != nil {
					log.Println("connect failed: ", err)
				}
				return
			}
			// 传统http代理
			if r.URL.IsAbs() && strings.HasPrefix(r.URL.Scheme, "http") {
				if err := s.handleHTTPProxyForward(conn, r); err != nil {
					log.Println("http proxy forward failed: ", err)
				}
				return
			}
		}
		// httpweb请求
		if s.HandlerFunc != nil {
			s.HandlerFunc(w, r)
			return
		}
		http.NotFound(w, r)
	}))
}
