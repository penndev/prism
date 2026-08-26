package main

import (
	"desktop/internal/dns"
	"log"
)

func main() {
	err := dns.ListenUDP53("127.0.0.1")
	if err != nil {
		log.Fatal(err)
	}
}
