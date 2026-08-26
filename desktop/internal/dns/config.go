package dns

import "time"

var (
	upstream    = "223.5.5.5"
	dohUpstream = "https://8.8.8.8/dns-query"
)

const (
	udpTimeout   = 5 * time.Second
	dohTimeout   = 5 * time.Second
	raceWindow   = 150 * time.Millisecond // 首包后再等，观察抢答
	defaultProxy = "127.0.0.1:1080"
)

// SetUpstream 设置 UDP 上游 DNS。
func SetUpstream(up string) {
	if up != "" {
		upstream = up
	}
}
