package main

import (
	"desktop/internal/dns"
	"log"
)

func main() {
	if err := dns.StartUDP53("127.0.0.1", 53); err != nil {
		log.Fatal(err)
	}
	select {}
}
