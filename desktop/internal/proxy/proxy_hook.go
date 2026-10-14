package proxy

import (
	"context"
	"desktop/internal"
	"desktop/internal/storage"
	"net"
	"time"

	"github.com/penndev/prism/engine"
	"github.com/penndev/prism/transport"
)

func (p *Proxy) handleConnectHook(handle transport.HandleConnect, callback func(line string)) transport.HandleConnect {
	h := connectHook{p: p, callback: callback}
	return func(conn net.Conn, network, address string) error {
		// 本地代理端口靠这里收尾。Relay 只在失败时关 conn，成功路径不关。
		defer conn.Close()
		return engine.Relay(handle, localHandle, h, conn, network, address)
	}
}

type connectHook struct {
	p        *Proxy
	callback func(line string)
}

func (h connectHook) OnLog(line string) {
	if h.callback != nil && line != "" {
		h.callback(line)
	}
}

func (h connectHook) OnProxyRead(n int64)  { h.p.readBytes.Add(n) }
func (h connectHook) OnProxyWrite(n int64) { h.p.writeBytes.Add(n) }

func (h connectHook) UseProxy(network, address string) bool {
	if st := storage.DefaultStorage; st != nil {
		if cfg := st.CachedRuleConfig(); cfg != nil {
			bypass := false
			switch cfg.AreaMode {
			case "none":
				bypass = true
			case "proxy":
				bypass = len(cfg.AreaIDs) == 0 || !inAreas(address, cfg.AreaIDs)
			case "bypass":
				bypass = len(cfg.AreaIDs) > 0 && inAreas(address, cfg.AreaIDs)
			}
			if bypass {
				h.OnLog("bypass -> " + network + " " + address)
				return false
			}
		}
	}
	h.OnLog("proxy -> " + network + " " + address)
	return true
}

func inAreas(address string, areaIDs []uint32) bool {
	if len(areaIDs) == 0 {
		return false
	}
	path, err := storage.IpregionDBPath()
	if err != nil {
		return false
	}
	if err := internal.EnsureSearcher(path); err != nil {
		return false
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		cancel()
		if err != nil || len(ips) == 0 {
			return false
		}
		ip = ips[0]
	}
	searcher := internal.AcquireSearcher()
	defer internal.ReleaseSearcher()
	if searcher == nil {
		return false
	}
	info, err := searcher.Find(ip.String())
	if err != nil || info.Area.ID == 0 {
		return false
	}
	set := make(map[uint32]struct{}, len(areaIDs))
	for _, id := range areaIDs {
		if id != 0 {
			set[id] = struct{}{}
		}
	}
	if len(set) == 0 {
		return false
	}
	for a := &info.Area; a != nil; a = a.Parent {
		if _, ok := set[a.ID]; ok {
			return true
		}
	}
	return false
}
