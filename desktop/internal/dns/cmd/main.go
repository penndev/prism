package main

import "desktop/internal/dns"

func main() {
	dns.ListenUDP53("127.0.0.1")
}
