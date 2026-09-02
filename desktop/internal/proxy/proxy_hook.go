package proxy

import (
	"desktop/internal"
	"desktop/internal/storage"
	"net"

	"github.com/penndev/gopkg/ipregion"
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
			if cfg, err := st.GetRuleConfig(); err == nil && cfg != nil {
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
				if internal.App != nil {
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
	if internal.Searcher == nil {
		path, err := storage.IpregionDBPath()
		if err != nil {
			return false
		}
		s, err := ipregion.Open(path)
		if err != nil {
			return false
		}
		internal.Searcher = s
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		ip = ips[0]
	}
	info, err := internal.Searcher.Find(ip.String())
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
