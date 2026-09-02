// Package storage 提供 Bolt 持久化及与前端 JSON 字段一致的数据模型。
package storage

import (
	"encoding/json"
	"strings"
)

// Settings 与前端 settings store 持久化 JSON 一致。
type Settings struct {
	Proxy       ProxySettings       `json:"proxy"`
	LatencyTest LatencyTestSettings `json:"latencyTest"`
	System      SystemSettings      `json:"system"`
}

type ProxySettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LatencyTestSettings struct {
	// Host 延迟探测用的 HTTP 目标，形如 host、host:port（默认 80）。
	Host string `json:"host"`
	// SortAfterPing 测速完成后按延迟从低到高排序。
	SortAfterPing bool `json:"sortAfterPing"`
}

type SystemSettings struct {
	Language           string `json:"language"`
	ThemeMode          string `json:"themeMode"`
	StartupOnBoot      bool   `json:"startupOnBoot"`
	EnableLogRecording bool   `json:"enableLogRecording"`
}

// ServerEntry 与前端服务器列表项一致（不含运行时延迟字段）。
type ServerEntry struct {
	Host     string `json:"host"`
	Remark   string `json:"remark"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
}

// NormalizeProtocol maps stored/imported protocol names to socks5 / socks5s / http / https.
func NormalizeProtocol(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	switch key {
	case "socks5", "socks5s", "http", "https":
		return key
	default:
		return "socks5"
	}
}

func normalizeServer(v ServerEntry) ServerEntry {
	v.Protocol = NormalizeProtocol(v.Protocol)
	return v
}

func normalizeServers(servers []ServerEntry) []ServerEntry {
	out := make([]ServerEntry, len(servers))
	for i, s := range servers {
		out[i] = normalizeServer(s)
	}
	return out
}

// RuleConfig 规则配置（Web 页维护；桌面端展示状态提示）。
type RuleConfig struct {
	// AreaMode: global=全局代理，none=全不代理，proxy=代理某些区域，bypass=绕过某些区域。
	AreaMode string `json:"areaMode"`
	// AreaIDs 选中的地域 ID 列表（global/none 时可为空）。
	AreaIDs []uint32 `json:"areaIds"`
	// Domains 需要走代理的域名。空列表表示域名规则暂不生效。
	Domains []string `json:"domains"`
}

// UnmarshalJSON 兼容旧字段 mode。
func (r *RuleConfig) UnmarshalJSON(data []byte) error {
	type raw struct {
		AreaMode string   `json:"areaMode"`
		Mode     string   `json:"mode"`
		AreaIDs  []uint32 `json:"areaIds"`
		Domains  []string `json:"domains"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	r.AreaMode = strings.TrimSpace(v.AreaMode)
	if r.AreaMode == "" {
		r.AreaMode = strings.TrimSpace(v.Mode)
	}
	r.AreaIDs = v.AreaIDs
	r.Domains = v.Domains
	return nil
}

// RuleStatus 主页面展示用的规则摘要。
type RuleStatus struct {
	AreaMode  string   `json:"areaMode"`
	AreaNames []string `json:"areaNames"`
	Domains   []string `json:"domains"`
}

const (
	KeySettings       = "settings"
	KeyServers        = "servers"
	KeyRule           = "rule"
	KeySelectedServer = "selectedServer"
)
