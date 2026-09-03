package proxy

import (
	"bufio"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const authRealm = `Basic realm="Prism"`

// verifyBasicAuth 校验一条 Basic 凭据头。未配置用户名密码时不做鉴权。
// 代理请求取 Proxy-Authorization，本地 web 管理页取 Authorization。
func (s *Server) verifyBasicAuth(auth string) bool {
	if s.Username == "" && s.Password == "" {
		return true
	}
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(auth, prefix)))
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(s.Username))
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(s.Password))
	return userOK&passOK == 1
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
// 这里不 Hijack，而是解析上游响应后经 w 写回，逐跳头与 keep-alive 交给 net/http 处理。
// 反过来（Hijack 后裸管道转发）不可行：net/http 会在 handler 返回时补写一个自己的空响应，
// 而裸管道也无法区分客户端在同一条连接上复用发来的、指向别的主机的后续请求。
func (s *Server) handleHTTPProxyForward(w http.ResponseWriter, req *http.Request) error {
	port := req.URL.Port()
	if port == "" {
		port = "80"
	}
	addr := net.JoinHostPort(req.URL.Hostname(), port)

	remote, local := net.Pipe()
	go func() {
		if err := s.HandleConnect(remote, "tcp", addr); err != nil {
			// 关掉管道，让下面的 Write / ReadResponse 立刻失败而不是一直等。
			remote.Close()
		}
	}()
	defer local.Close()

	// 复写请求：代理收到的是绝对 URL，转发给上游要改成 origin-form。
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
	// req.Close 会让 Request.Write 重新写出 Connection: close，抵消下面的逐跳头清理。
	cc.Close = false
	for _, key := range HttpProxyHeaders {
		cc.Header.Del(key)
	}
	if err := cc.Write(local); err != nil {
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(local), cc)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	for _, key := range HttpProxyHeaders {
		resp.Header.Del(key)
	}
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	return err
}

func (s *Server) proxyHTTP(conn net.Conn) {
	listener := &HttpSingleConnListener{conn: conn}
	http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CONNECT 与绝对 URL 是代理请求，其余按本地 web 管理页处理。
		isProxyReq := r.Method == http.MethodConnect ||
			(r.URL.IsAbs() && strings.HasPrefix(r.URL.Scheme, "http"))
		if isProxyReq {
			if !s.verifyBasicAuth(r.Header.Get("Proxy-Authorization")) {
				w.Header().Set("Proxy-Authenticate", authRealm)
				http.Error(w, "proxy authentication required", http.StatusProxyAuthRequired)
				return
			}
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
			if err := s.handleHTTPProxyForward(w, r); err != nil {
				log.Println("http proxy forward failed: ", err)
			}
			return
		}
		// 本地 web 管理页复用代理凭据，未通过时回 401 让浏览器弹出账号密码输入框。
		if !s.verifyBasicAuth(r.Header.Get("Authorization")) {
			w.Header().Set("WWW-Authenticate", authRealm)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if s.HandlerFunc != nil {
			s.HandlerFunc(w, r)
			return
		}
		http.NotFound(w, r)
	}))
}
