// Package storage 提供 Bolt 持久化及与前端 JSON 字段一致的数据模型。
package storage

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

type PACConfig struct {
	Domains     string `json:"domains"`
	Mode        string `json:"mode"`
	PACTemplate string `json:"pacTemplate"`
}

// RuleConfig 规则配置（Web 页维护；桌面端展示状态提示）。
type RuleConfig struct {
	// Mode: global=全局代理，proxy=代理某些区域，bypass=绕过某些区域。
	Mode string `json:"mode"`
	// AreaIDs 选中的地域 ID 列表（global 时可为空）。
	AreaIDs []uint32 `json:"areaIds"`
}

// RuleStatus 主页面展示用的地域规则摘要。
type RuleStatus struct {
	Mode      string   `json:"mode"`
	AreaNames []string `json:"areaNames"`
}

const (
	KeySettings       = "settings"
	KeyServers        = "servers"
	KeyPAC            = "pac"
	KeyRule           = "rule"
	KeySelectedServer = "selectedServer"
)
