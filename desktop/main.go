package main

import (
	"desktop/build"
	"desktop/internal"
	"desktop/internal/autostart"
	"desktop/internal/dns"
	"desktop/internal/lang"
	"desktop/internal/proxy"
	"desktop/internal/storage"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	var err error
	go func() {
		err := dns.ListenUDP53("127.0.0.1")
		if err != nil {
			log.Println(err)
		}
	}()

	// 语言文件
	lang.DefaultLang, err = lang.New()
	if err != nil {
		panic(err)
	}
	// 处理存储
	storage.DefaultStorage, err = storage.New()
	if err != nil {
		panic(err)
	}
	// -
	app, err := internal.SetUPApp([]application.Service{
		application.NewService(lang.DefaultLang),
		application.NewService(storage.DefaultStorage),
		application.NewService(proxy.New()),
		application.NewService(&internal.AppConst{}),
		application.NewService(&proxy.ProxyPing{}),
		application.NewService(&autostart.Autostart{}),
	})
	if err != nil {
		panic(err)
	}

	// 设置app托管栏
	systrayMenu := app.NewMenu()
	systrayMenuShowMain := systrayMenu.Add(lang.DefaultLang.T("systray.showMainWindow"))
	systrayMenuQuit := systrayMenu.Add(lang.DefaultLang.T("systray.quit"))

	// 设置app托管栏功能
	systrayMenuShowMain.OnClick(func(ctx *application.Context) {
		internal.MainWindowSetPosition()
	})
	systrayMenuQuit.OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	// 设置任务托管栏功能
	systray := app.SystemTray.New()
	systray.SetTooltip(lang.DefaultLang.T("app.title"))
	systray.SetIcon(build.Icon)
	systray.SetMenu(systrayMenu)
	systray.OnClick(func() {
		internal.MainWindow.Show()
		internal.MainWindow.Focus()
	})

	// 设置任务托管栏文案随 Lang 切换更新
	app.Event.On(internal.AppConfig.EventNameLocaleChanged, func(_ *application.CustomEvent) {
		systrayMenuShowMain.SetLabel(lang.DefaultLang.T("systray.showMainWindow"))
		systrayMenuQuit.SetLabel(lang.DefaultLang.T("systray.quit"))
	})

	// 监听退出
	app.Event.On(internal.AppConfig.EventNameServiceAppQuit, func(_ *application.CustomEvent) {
		app.Quit()
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
