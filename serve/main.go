package main

import (
	"flag"
	"log"
	"net"
	"net/url"

	"github.com/penndev/prism/proxy"
	"github.com/penndev/prism/transport"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:1080", "listen address")
	user := flag.String("user", "", "proxy username")
	pass := flag.String("pass", "", "proxy password")
	proxyurl := flag.String("proxy", "", "remote proxy URL, e.g. socks5://user:pass@192.168.0.1:1080")
	flag.Parse()

	handle := transport.Local()

	if *proxyurl != "" {
		r, err := url.Parse(*proxyurl)
		if err != nil {
			panic(err)
		}
		h, err := transport.FromURL(r)
		if err != nil {
			panic(err)
		}
		handle = h
	}
	s := proxy.New(*addr, *user, *pass)
	s.HandleConnect = func(conn net.Conn, network, address string) error {
		log.Println("req ->", network, address)
		return handle(conn, network, address)
	}
	log.Printf("start -> %s %s %s", *addr, *user, *pass)
	err := s.ListenAndServe()
	if err != nil {
		log.Println("listen failed: ", err)
	}
}
