package proxy

import (
	"desktop/internal"
	"desktop/internal/storage"
	"net"

	"github.com/penndev/prism/ipregion"
	"github.com/penndev/prism/transport"
)

func (p *Proxy) handleConnectHook(handle transport.HandleConnect, callback func(network, address string)) transport.HandleConnect {
	return func(conn net.Conn, network, address string) error {
		// 每次请求判断地域规则：需绕过则走本地，否则走远程代理。
		bypass := false
		if st := storage.DefaultStorage; st != nil {
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
		}
		if bypass {
			if internal.App != nil {
				internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "bypass -> "+network+" "+address)
			}
			return localHandle(conn, network, address)
		}
		if callback != nil {
			callback(network, address)
		}
		// 只有走代理服务器才走包装
		conn = p.wrapConn(conn)
		return handle(conn, network, address)
	}
}

func inAreas(address string, areaIDs []uint32) bool {
	if len(areaIDs) == 0 {
		return false
	}
	if err := storage.OpenIpregion(); err != nil {
		return false
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
	info, err := ipregion.Find(ip.String())
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
