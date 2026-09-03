package proxy

import (
	"context"
	"desktop/internal"
	"desktop/internal/storage"
	"net"
	"time"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/pkg"
	"github.com/penndev/prism/transport"
)

func (p *Proxy) handleConnectHook(handle transport.HandleConnect, callback func(network, address string)) transport.HandleConnect {
	return func(conn net.Conn, network, address string) error {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}

		// 拦截udp协议 进行中间人攻击替换IP
		if network == "udp" || network == "udp4" || network == "udp6" {
			if port == "53" {
				fakeip.Handle(conn, network, address)
				callback("fakedns -> "+network, address)
				return nil
			}
		}

		// 判断是否是fakeIP。是的话直接替换然后代理。
		if domain, ok := fakeip.Lookup(host); ok {
			fakeaddr := net.JoinHostPort(domain, port)
			if callback != nil {
				callback("fakeip -> "+network, address+" - "+fakeaddr)
			}
			conn = pkg.WrapConn(conn, func(n int64) { p.readBytes.Add(n) }, func(n int64) { p.writeBytes.Add(n) })
			return handle(conn, network, fakeaddr)
		}

		// 验证地域规则
		if st := storage.DefaultStorage; st != nil {
			bypass := false
			if cfg := st.CachedRuleConfig(); cfg != nil {
				switch cfg.AreaMode {
				case "none":
					bypass = true
				case "proxy":
					bypass = len(cfg.AreaIDs) == 0 || !inAreas(address, cfg.AreaIDs)
				case "bypass":
					bypass = len(cfg.AreaIDs) > 0 && inAreas(address, cfg.AreaIDs)
				}
			}
			if bypass {
				if callback != nil {
					callback("bypass -> "+network, address)
				}
				return localHandle(conn, network, address)
			}
		}

		if callback != nil {
			callback("proxy -> "+network, address)
		}
		conn = pkg.WrapConn(conn, func(n int64) { p.readBytes.Add(n) }, func(n int64) { p.writeBytes.Add(n) })
		return handle(conn, network, address)
	}
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
