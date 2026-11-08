package proxy

import (
	"desktop/internal"
	"desktop/internal/ipregion"
	"desktop/internal/storage"
	"net"

	"github.com/penndev/prism/engine"
	"github.com/penndev/prism/transport"
)

func (p *Proxy) handleConnectHook(handle transport.HandleConnect) transport.HandleConnect {
	h := connectHook{p: p}
	return func(conn net.Conn, network, address string) error {
		// 本地代理端口靠这里收尾。Relay 只在失败时关 conn，成功路径不关。
		defer conn.Close()
		return engine.Relay(handle, localHandle, h, conn, network, address)
	}
}

type connectHook struct {
	p *Proxy
}

func (h connectHook) OnStatus(line string) {
	if line == "" || internal.App == nil {
		return
	}
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, line)
}

func (h connectHook) OnConnection(network, address, err string) {
	if internal.App == nil || address == "" {
		return
	}
	line := "proxy -> " + network + " " + address
	if err != "" {
		line = network + " " + address + ": " + err
	}
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, line)
}

func (h connectHook) OnDNS(name, qtype, result string) {
	if internal.App == nil || (name == "" && qtype == "") {
		return
	}
	line := name + " " + qtype
	if result != "" {
		line += " " + result
	}
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_DNS, line)
}

func (h connectHook) OnProxyRead(n int64) { h.p.readBytes.Add(n) }

func (h connectHook) OnProxyWrite(n int64) { h.p.writeBytes.Add(n) }

func (h connectHook) UseProxy(network, address string) bool {
	if st := storage.DefaultStorage; st != nil {
		if cfg := st.CachedRuleConfig(); cfg != nil {
			// AreaIDs 为空就不做地域判断，一律走代理。名单里的地区直连。
			if len(cfg.AreaIDs) > 0 && ipregion.Contains(address, cfg.AreaIDs) {
				if internal.App != nil {
					internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "bypass -> "+network+" "+address)
				}
				return false
			}
		}
	}
	if internal.App != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "proxy -> "+network+" "+address)
	}
	return true
}
