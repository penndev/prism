package proxy

import (
	"desktop/internal"
	"desktop/internal/ipregion"
	"desktop/internal/storage"
	"net"

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
					bypass = len(cfg.AreaIDs) == 0 || !ipregion.InAreas(address, cfg.AreaIDs)
				case "bypass":
					bypass = len(cfg.AreaIDs) > 0 && ipregion.InAreas(address, cfg.AreaIDs)
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
