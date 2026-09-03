package internal

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/browser"
)

type appConst struct {
	// 应用运行状态日志名称
	LogTypeName_STATUS string
	// 连接日志列表的消息名称
	LogTypeName_LOG string
	// 语言切换事件（payload 为语言标识字符串）
	EventNameLocaleChanged string
	// 节点列表变更事件（payload 为更新时间戳）
	EventNameServersChanged string
	// 服务退出事件
	EventNameServiceAppQuit string
}

var AppConfig = appConst{
	LogTypeName_STATUS:      "logServerStatus",
	LogTypeName_LOG:         "logProxyList",
	EventNameLocaleChanged:  "localeChanged",
	EventNameServersChanged: "serversChanged",
	EventNameServiceAppQuit: "serviceAppQuit",
}

type AppConst struct{}

func (a AppConst) AppConfig() appConst {
	return AppConfig
}

func (a AppConst) ProxyScheme() []string {
	return []string{
		"socks5",
		"socks5s",
		"http",
		"https",
	}
}

func (a AppConst) OpenExternalURL(rawURL string) error {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return nil
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}
	return browser.OpenURL(parsed.String())
}
