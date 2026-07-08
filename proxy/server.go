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
			var err error
			host := req.Addr()
			network := ""
			switch req.CMD {
			case socks5.CMD_CONNECT:
				network = "tcp"
			case socks5.CMD_UDP_ASSOCIATE:
				network = "udp"
			default:
				rep(socks5.REP_COMMAND_NOT_SUPPORTED)
			}
			rep(socks5.REP_SUCCEEDED)
			err = s.HandleConnect(c, network, host)
			return err
		},
	}
	go s.socks5Server.UDPListen()
}

func (s *Server) ListenAndServe() error {
	var err error
	s.ln, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer s.ln.Close()
	s.initSocks5Server()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			log.Println("accept failed: ", err)
			continue
		}
		go s.handleConn(NewConn(conn))
	}
	return err
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
