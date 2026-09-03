package proxy

import (
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/penndev/gopkg/socks5"
	"github.com/penndev/prism/transport"
)

type Server struct {
	Addr          string
	HandleConnect transport.HandleConnect
	HandlerFunc   http.HandlerFunc
	Username      string
	Password      string
	socks5Server  *socks5.Server
	ln            net.Listener
}

func (s *Server) handleConn(conn *Conn) {
	// 判断协议类型如果是05则代理socks5，否则代理http
	buf, err := conn.Peek(1)
	if err != nil {
		log.Println("read failed: ", err)
		conn.Close()
		return
	}
	if buf[0] == 0x05 {
		s.proxySocks5(conn)
		return
	}
	s.proxyHTTP(conn)
}

func (s *Server) Close() {
	if s.socks5Server != nil {
		s.socks5Server.Close()
	}
	if s.ln != nil {
		s.ln.Close()
	}
}

func (s *Server) initSocks5Server() {
	// 绑定socks5到server
	s.socks5Server = &socks5.Server{
		Addr:     s.Addr,
		Username: s.Username,
		Password: s.Password,
		HandleConnect: func(c net.Conn, req socks5.Requests, rep socks5.HandleReply) error {
			host := req.Addr()
			var network string
			switch req.CMD {
			case socks5.CMD_CONNECT:
				network = "tcp"
			case socks5.CMD_UDP_ASSOCIATE:
				network = "udp"
			default:
				// 这里不 return 就会紧接着再回一个 REP_SUCCEEDED，
				// 并且拿空 network 去调 HandleConnect。
				rep(socks5.REP_COMMAND_NOT_SUPPORTED)
				return nil
			}
			rep(socks5.REP_SUCCEEDED)
			return s.HandleConnect(c, network, host)
		},
	}
	go s.socks5Server.UDPListen()
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	defer ln.Close()
	s.initSocks5Server()
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Close() 关掉监听后走这里，属于正常退出
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Println("accept failed: ", err)
			continue
		}
		go s.handleConn(NewConn(conn))
	}
}

func New(addr, username, password string) *Server {
	s := &Server{
		Addr:          addr,
		HandleConnect: transport.Local(),
		Username:      username,
		Password:      password,
	}
	return s
}
