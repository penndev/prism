// Package autostart 管理各平台的开机自启动注册。
package autostart

import "errors"

const appName = "Prism"

var errUnsupported = errors.New("autostart: unsupported platform")

// Autostart Wails 绑定服务。
type Autostart struct{}

// Enable 注册开机自启动。
func (a *Autostart) Enable() error {
	return enable()
}

// Disable 取消开机自启动。
func (a *Autostart) Disable() error {
	return disable()
}
